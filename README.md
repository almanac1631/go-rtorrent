# go-rtorrent
[![GoDoc](https://godoc.org/github.com/mrobinsn/go-rtorrent/rtorrent?status.svg)](https://godoc.org/github.com/mrobinsn/go-rtorrent/rtorrent)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrobinsn/go-rtorrent)](https://goreportcard.com/report/github.com/mrobinsn/go-rtorrent)
[![CircleCI](https://circleci.com/gh/mrobinsn/go-rtorrent/tree/master.svg?style=svg)](https://circleci.com/gh/mrobinsn/go-rtorrent/tree/master)
[![Coverage Status](https://coveralls.io/repos/github/mrobinsn/go-rtorrent/badge.svg?branch=master)](https://coveralls.io/github/mrobinsn/go-rtorrent?branch=master)
[![MIT license](http://img.shields.io/badge/license-MIT-brightgreen.svg)](http://opensource.org/licenses/MIT)


> rTorrent XMLRPC Bindings for Go (golang)

Fork of [github.com/mrobinsn/go-rtorrent](github.com/mrobinsn/go-rtorrent).

## Documentation
[GoDoc](https://godoc.org/github.com/mrobinsn/go-rtorrent/rtorrent)

## Features
- Get IP, Name, Up/Down totals
- Get torrents within a view
- Get torrent by hash
- Get files for torrents
- Set the label on a torrent
- Add a torrent by URL or by metadata
- Delete a torrent (including files)

## Installation
To install the package, run `go get github.com/mrobinsn/go-rtorrent`

To use it in application, import `"github.com/mrobinsn/go-rtorrent/rtorrent"`

To install the command line utility, run `go install "github.com/mrobinsn/go-rtorrent"`

## Library Usage

```
conn, _ := rtorrent.New("http://my-rtorrent.com/RPC2", false)
name, _ := conn.Name()
fmt.Printf("My rTorrent's name: %v", name)
```

You can connect to a server using Basic Authentication by including the credentials in the endpoint URL:
```
conn, _ := rtorrent.New("https://user:pass@my-rtorrent.com/RPC2", false)
```

## Contributing
Pull requests are welcome, please ensure you add relevant tests for any new/changed functionality.
