//go:build integration

package rtorrent

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

const (
	sleepDuration   = time.Millisecond * 100
	maxRetries      = 60
	testTorrentFile = "ubuntu-24.04.3-live-server-amd64.iso.torrent"
	testTorrentUrl  = "https://www.releases.ubuntu.com/noble/" + testTorrentFile
)

var client *Client

func TestMain(m *testing.M) {
	/*
		These tests rely on a local instance of rtorrent to be running in a clean state.
		Use the included `test.sh` script to run these tests.
	*/
	addr := os.Getenv("RTORRENT_TEST_URL")

	slog.Info("RTORRENT_TEST_URL from env", slog.String("addr", addr))

	if addr == "" {
		addr = "http://localhost:8000/RPC2"

		slog.Info("RTORRENT_TEST_URL from env empty, fallback to default", slog.String("addr", addr))
	}
	client = NewClient(Config{Addr: addr, TLSSkipVerify: false})
	os.Exit(m.Run())
}

func TestRTorrent(t *testing.T) {
	ctx := context.Background()

	t.Run("get ip", func(t *testing.T) {
		ctx := context.Background()
		ip, err := client.IP(ctx)
		require.NoError(t, err)
		require.Regexp(t, `\d+.\d+.\d+.\d+`, ip)
	})

	t.Run("get name", func(t *testing.T) {
		ctx := context.Background()
		name, err := client.Name(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, name)
	})

	t.Run("down total", func(t *testing.T) {
		total, err := client.DownTotal(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, total, 0)
	})

	t.Run("up total", func(t *testing.T) {
		total, err := client.UpTotal(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, total, 0)
	})

	t.Run("down rate", func(t *testing.T) {
		rate, err := client.DownRate(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, rate, 0)
	})

	t.Run("up rate", func(t *testing.T) {
		rate, err := client.UpRate(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, rate, 0)
	})

	t.Run("get no torrents", RtorrentTest(func(t *testing.T) {
		assertNoTorrentsPresent(t)
	}))

	t.Run("add torrent by file and deletion", RtorrentTest(func(t *testing.T) {
		// read torrent file
		b, err := os.ReadFile("testdata/" + testTorrentFile)
		require.NoError(t, err)
		require.NotEmpty(t, b)

		// add torrent
		err = client.AddTorrent(ctx, b)
		require.NoError(t, err)

		// check torrent presence
		torrent, err := getTorrent(t, ctx, "A1DFEFEC1A9DD7FA8A041EBEEEA271DB55126D2F")
		require.NoError(t, err)

		// delete torrent
		err = client.Delete(ctx, torrent)
		require.NoError(t, err)
		assertNoTorrentsPresent(t)
	}))

	t.Run("add torrent by url", RtorrentTest(func(t *testing.T) {
		// add torrent
		err := client.Add(ctx, testTorrentUrl)
		require.NoError(t, err)

		// check torrent presence
		_, err = getTorrent(t, ctx, "A1DFEFEC1A9DD7FA8A041EBEEEA271DB55126D2F")
		require.NoError(t, err)
	}))

	t.Run("add stopped torrent", RtorrentTest(func(t *testing.T) {
		// add torrent
		err := client.AddStopped(ctx, testTorrentUrl)
		require.NoError(t, err)

		// check torrent presence
		_, err = getTorrentFromView(t, ctx, "A1DFEFEC1A9DD7FA8A041EBEEEA271DB55126D2F", ViewStopped)
		require.NoError(t, err)
	}))

	t.Run("retrieve single torrent info", RtorrentTestTorrentPresent(ctx, func(t *testing.T, hash string) {
		err := client.Add(ctx, testTorrentUrl)
		require.NoError(t, err)

		// check torrent presence
		torrent, err := client.GetTorrent(ctx, hash)
		require.Equal(t, "ubuntu-24.04.3-live-server-amd64.iso", torrent.Name)
		require.Equal(t, "", torrent.Label)
		require.Equal(t, 3303444480, torrent.Size)
		require.Equal(t, "/downloads/temp", torrent.Path)
		require.False(t, torrent.Completed)
		require.NoError(t, err)

		t.Run("get files", func(t *testing.T) {
			files, err := client.GetFiles(ctx, torrent)
			require.NoError(t, err)
			require.NotEmpty(t, files)
			require.Len(t, files, 1)
			for _, f := range files {
				require.NotEmpty(t, f.Path)
				require.NotZero(t, f.Size)
			}
		})

		t.Run("get status", func(t *testing.T) {
			status, err := client.GetStatus(ctx, torrent)
			require.NoError(t, err)

			require.False(t, status.Completed)
			require.GreaterOrEqual(t, status.CompletedBytes, 0)
			require.GreaterOrEqual(t, status.DownRate, 0)
			require.GreaterOrEqual(t, status.UpRate, 0)
			require.GreaterOrEqual(t, status.Ratio, 0.0)
			require.NotZero(t, status.Size)
		})
	}))

	t.Run("torrent interactions", func(t *testing.T) {
		t.Run("change label", RtorrentTestTorrentPresent(ctx, func(t *testing.T, hash string) {
			err := client.SetLabel(ctx, Torrent{Hash: hash}, "TestLabel")
			require.NoError(t, err)

			torrent, err := getTorrent(t, ctx, hash)
			require.NoError(t, err)
			require.Equal(t, "TestLabel", torrent.Label)
		}))

		t.Run("stop and start torrent", RtorrentTestTorrentPresent(ctx, func(t *testing.T, hash string) {
			torrent := Torrent{Hash: hash}

			_, err := getTorrentFromView(t, ctx, hash, ViewStarted)
			require.NoError(t, err)

			assertTorrentStatus(t, ctx, hash, true, true, 1)

			// stop torrent
			err = client.StopTorrent(ctx, torrent)
			require.NoError(t, err)

			_, err = getTorrentFromView(t, ctx, hash, ViewStopped)
			require.NoError(t, err)

			assertTorrentStatus(t, ctx, hash, false, false, 0)

			// start torrent again
			err = client.StartTorrent(ctx, torrent)
			require.NoError(t, err)

			_, err = getTorrentFromView(t, ctx, hash, ViewStarted)
			require.NoError(t, err)

			assertTorrentStatus(t, ctx, hash, true, true, 1)
		}))

		t.Run("pause and resume torrent", RtorrentTestTorrentPresent(ctx, func(t *testing.T, hash string) {
			torrent := Torrent{Hash: hash}

			err := client.PauseTorrent(ctx, torrent)
			require.NoError(t, err)

			assertTorrentStatus(t, ctx, hash, true, false, 1)

			err = client.ResumeTorrent(ctx, torrent)
			require.NoError(t, err)

			_, err = getTorrentFromView(t, ctx, hash, ViewStarted)

			assertTorrentStatus(t, ctx, hash, true, true, 1)
		}))
	})
}

func assertTorrentStatus(t *testing.T, ctx context.Context, hash string, isOpenExpected, isActiveExpected bool, stateExpected int) {
	torrent := Torrent{Hash: hash}
	isOpen, err := client.IsOpen(ctx, torrent)
	require.NoError(t, err)
	require.Equal(t, isOpenExpected, isOpen)

	isActive, err := client.IsActive(ctx, torrent)
	require.NoError(t, err)
	require.Equal(t, isActiveExpected, isActive)

	state, err := client.State(ctx, torrent)
	require.NoError(t, err)
	require.Equal(t, stateExpected, state)
}

func RtorrentTest(testFunc func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		cleanup()
		testFunc(t)
		cleanup()
	}
}

func RtorrentTestTorrentPresent(ctx context.Context, testFunc func(t *testing.T, hash string)) func(t *testing.T) {
	return func(t *testing.T) {
		cleanup()
		// read torrent file
		b, err := os.ReadFile("testdata/" + testTorrentFile)
		require.NoError(t, err)
		require.NotEmpty(t, b)

		// add torrent
		err = client.AddTorrent(ctx, b)
		require.NoError(t, err)

		expectedHash := "A1DFEFEC1A9DD7FA8A041EBEEEA271DB55126D2F"
		// check torrent presence
		torrent, err := getTorrentFromView(t, ctx, expectedHash, ViewStarted)
		require.NoError(t, err)
		testFunc(t, torrent.Hash)
		cleanup()
	}
}

func getTorrent(t *testing.T, ctx context.Context, hash string) (Torrent, error) {
	return getTorrentFromView(t, ctx, hash, ViewMain)
}

func getTorrentFromView(t *testing.T, ctx context.Context, hash string, view View) (Torrent, error) {
	for i := 0; i < maxRetries; i++ {
		torrents, err := client.GetTorrents(ctx, view)
		require.NoError(t, err)
		for _, torrentElem := range torrents {
			if torrentElem.Hash == hash {
				return torrentElem, nil
			}
		}
		time.Sleep(sleepDuration)
	}
	return Torrent{}, errors.Errorf("failed to find torrent with hash %s", hash)
}

func assertNoTorrentsPresent(t *testing.T) {
	torrents, err := client.GetTorrents(context.Background(), ViewMain)
	require.NoError(t, err)
	require.Empty(t, torrents, "expected no torrents to be present")
}

func cleanup() {
	deleteAllTorrents := func() (int, error) {
		torrents, err := client.GetTorrents(context.Background(), ViewMain)
		if err != nil {
			return -1, fmt.Errorf("error getting torrents: %w", err)
		}
		for _, torrent := range torrents {
			_ = client.SetForceDelete(context.Background(), torrent, true)
			_ = client.Delete(context.Background(), torrent)
		}
		return len(torrents), nil
	}
	for i := 0; i < maxRetries; i++ {
		amountExisting, err := deleteAllTorrents()
		if err != nil {
			log.Println("[WARN] failed to cleanup torrents: ", err)
			return
		}
		if amountExisting == 0 {
			return
		}
	}
	log.Println("[WARN] failed to cleanup torrents due to torrents remaining after multiple retries")
}
