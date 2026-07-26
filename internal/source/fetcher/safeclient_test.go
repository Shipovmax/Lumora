package fetcher

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSSRFSafeClientBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached: request should be blocked at dial time")
	}))
	defer srv.Close()

	client := newSSRFSafeHTTPClient(time.Second)

	_, err := client.Get(srv.URL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to connect")
}

func TestSSRFSafeClientAllowsPublicAddress(t *testing.T) {
	// A local listener bound to a non-loopback-looking address isn't
	// available in a sandboxed test environment, so instead verify the
	// classifier directly: it must not reject public IPs, only the
	// non-routable ranges the RFCs reserve for internal use.
	require.True(t, isPubliclyRoutable(net.ParseIP("8.8.8.8")))
	require.True(t, isPubliclyRoutable(net.ParseIP("1.1.1.1")))
}

func TestIsPubliclyRoutableRejectsInternalRanges(t *testing.T) {
	cases := []string{
		"127.0.0.1",       // loopback
		"::1",             // loopback (v6)
		"10.0.0.1",        // private
		"172.16.0.1",      // private
		"192.168.1.1",     // private
		"169.254.169.254", // link-local (cloud metadata endpoints)
		"0.0.0.0",         // unspecified
		"224.0.0.1",       // multicast
	}

	for _, ip := range cases {
		require.False(t, isPubliclyRoutable(net.ParseIP(ip)), "expected %s to be rejected", ip)
	}
}

func TestSSRFSafeClientRejectsUnresolvableHost(t *testing.T) {
	client := newSSRFSafeHTTPClient(time.Second)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://this-host-should-not-resolve.invalid/", nil)
	require.NoError(t, err)

	_, err = client.Do(req)
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), "handler must not be reached"))
}
