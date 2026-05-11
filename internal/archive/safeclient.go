package archive

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// safeClient returns an HTTP client that blocks connections to private,
// loopback, and link-local IP addresses. This prevents SSRF attacks
// where a user-supplied URL could reach internal services or cloud
// metadata endpoints (169.254.169.254, etc).
//
// The check happens at the dialer level after DNS resolution, so it
// also handles DNS rebinding attacks where a hostname initially resolves
// to a public IP but later resolves to a private one.
func safeClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   blockPrivateIPs,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:  10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// blockPrivateIPs is a net.Dialer Control function that rejects connections
// to private, loopback, link-local, and other non-public IP addresses.
func blockPrivateIPs(network, address string, conn syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", host)
	}

	if !isPublicIP(ip) {
		return fmt.Errorf("connections to non-public IP addresses are blocked: %s", ip)
	}

	return nil
}

func isPublicIP(ip net.IP) bool {
	// Normalize IPv4-mapped IPv6 addresses to IPv4
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	// Block loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return false
	}

	// Block private ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return false
	}

	// Block link-local (169.254.0.0/16, fe80::/10) -- includes AWS metadata 169.254.169.254
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	// Block unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return false
	}

	// Block multicast
	if ip.IsMulticast() {
		return false
	}

	return true
}

var httpClient = safeClient()

// SetHTTPClient overrides the HTTP client used for downloads. This is
// intended for testing only, where the test fixture server runs on localhost.
func SetHTTPClient(c *http.Client) {
	httpClient = c
}
