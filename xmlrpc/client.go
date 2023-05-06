package xmlrpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Client implements a basic XMLRPC client
type Client struct {
	addr       string
	httpClient *http.Client

	BasicUser string
	BasicPass string

	log *log.Logger
}

type Config struct {
	Addr          string
	TLSSkipVerify bool

	BasicUser string
	BasicPass string

	Log *log.Logger
}

// NewClient returns a new instance of Client
func NewClient(cfg Config) *Client {
	c := &Client{
		addr:      cfg.Addr,
		BasicUser: cfg.BasicUser,
		BasicPass: cfg.BasicPass,
		log:       log.New(io.Discard, "", log.LstdFlags),
	}
	transport := &http.Transport{}
	if cfg.TLSSkipVerify {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	c.httpClient = &http.Client{Transport: transport, Timeout: 60 * time.Second}

	// override logger if we pass one
	if cfg.Log != nil {
		c.log = cfg.Log
	}

	return c
}

// NewClientWithHTTPClient returns a new instance of Client.
// This allows you to use a custom http.Client setup for your needs.
func NewClientWithHTTPClient(addr string, client *http.Client) *Client {
	return &Client{
		addr:       addr,
		httpClient: client,
	}
}

// Call calls the method with "name" with the given args
// Returns the result, and an error for communication errors
func (c *Client) Call(ctx context.Context, name string, args ...interface{}) (interface{}, error) {
	data := bytes.NewBuffer(nil)
	if err := Marshal(data, name, args...); err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequestWithContext(ctx, c.addr, "text/xml", data)
	if err != nil {
		return nil, errors.Wrap(err, "creating request failed")
	}

	c.addBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "POST failed")
	}
	defer resp.Body.Close()

	_, val, fault, err := Unmarshal(resp.Body)
	if fault != nil {
		err = errors.Errorf("Error: %v: %v", err, fault)
	}
	return val, err
}

func (c *Client) addBasicAuth(req *http.Request) {
	if c.BasicUser != "" && c.BasicPass != "" {
		req.SetBasicAuth(c.BasicUser, c.BasicPass)
	}
}
