package tunnel

import (
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// nolint: gomnd
// same with http.DefaultTransport
func TransportOnTunnel(tunnel *TunnelServer, dest string) *http.Transport {
	defaultTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           tunnel.DialerOn(dest).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	// add http client support
	http2.ConfigureTransports(defaultTransport)
	return defaultTransport
}
