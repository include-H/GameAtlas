package files

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateRemoteImageURLBlocksLookupFailures(t *testing.T) {
	t.Parallel()

	_, err := validateRemoteImageURL("https://example.com/demo.png", func(context.Context, string) ([]net.IP, error) {
		return nil, errors.New("lookup failed")
	})
	if !errors.Is(err, ErrBlockedRemoteURL) {
		t.Fatalf("validateRemoteImageURL() error = %v, want %v", err, ErrBlockedRemoteURL)
	}
}

func TestDownloadRemoteAssetBlocksDNSRebinding(t *testing.T) {
	t.Parallel()

	store := NewAssetStore(t.TempDir(), "", 5*time.Second)
	var lookups atomic.Int32
	store.lookupHost = func(ctx context.Context, host string) ([]net.IP, error) {
		if host != "rebind.example" {
			return nil, errors.New("unexpected host lookup: " + host)
		}
		if lookups.Add(1) == 1 {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}

	_, err := store.DownloadRemoteAsset(
		"demo-game",
		"cover",
		"11111111-1111-4111-8111-111111111111",
		"http://rebind.example:80/demo.png",
	)
	if !errors.Is(err, ErrBlockedRemoteURL) {
		t.Fatalf("DownloadRemoteAsset() error = %v, want %v", err, ErrBlockedRemoteURL)
	}
	if lookups.Load() < 2 {
		t.Fatalf("lookup count = %d, want at least 2", lookups.Load())
	}
}
