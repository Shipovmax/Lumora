package fetcher

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// newSSRFSafeHTTPClient returns an http.Client whose Transport refuses to
// dial loopback/private/link-local/unspecified/multicast addresses. Sources
// (Этап 5) accept an arbitrary user-supplied URL, and RSSFetcher fetches it
// server-side from cmd/worker — without this, a user could register a source
// pointing at an internal-only address (e.g. cloud metadata endpoints,
// localhost services) and read the response back through their own briefing.
//
// The check runs inside DialContext, not as a one-time check on the parsed
// URL before the request: DialContext fires for the initial connection *and*
// for every redirect hop, so this closes both the "check host, then let a
// redirect jump to a private address" gap and, by validating the IP that is
// actually about to be dialed (rather than re-resolving the hostname
// separately from the connection), the DNS-rebinding gap a check-then-fetch
// validation would otherwise leave open.
func newSSRFSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}

			for _, ip := range ips {
				if !isPubliclyRoutable(ip) {
					return nil, fmt.Errorf("refusing to connect to non-public address %s", ip)
				}
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}

	return &http.Client{Transport: transport, Timeout: timeout}
}

func isPubliclyRoutable(ip net.IP) bool {
	switch {
	case ip.IsLoopback(), ip.IsPrivate(), ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(),
		ip.IsUnspecified(), ip.IsMulticast():
		return false
	default:
		return true
	}
}
