package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

var (
	byokClientsMu sync.RWMutex
	byokClients   = make(map[string]*http.Client)
)

// CheckBYOKRedirect validates every redirect destination against strict BYOK SSRF rules.
func CheckBYOKRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return fmt.Errorf("stopped after 3 redirects")
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("invalid redirect request")
	}
	urlStr := req.URL.String()
	if err := common.BYOKSSRFProtection.ValidateURL(urlStr); err != nil {
		return fmt.Errorf("redirect to %s blocked: %w", urlStr, err)
	}
	return nil
}

// byokRoundTripper wraps an http.Transport with unconditional BYOK SSRF validation.
type byokRoundTripper struct {
	transport http.RoundTripper
}

func (rt *byokRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("invalid request")
	}
	if err := common.BYOKSSRFProtection.ValidateURL(req.URL.String()); err != nil {
		return nil, fmt.Errorf("BYOK outbound request rejected by SSRF protection: %w", err)
	}
	return rt.transport.RoundTrip(req)
}

// NewBYOKHTTPClient returns an HTTP client equipped with strict, unconditional SSRF protection
// and a DNS-rebinding safe dialer for user-provided BYOK endpoints.
func NewBYOKHTTPClient(timeout time.Duration) *http.Client {
	return NewBYOKHTTPClientWithDialer(net.DefaultResolver, nil, timeout)
}

// NewBYOKHTTPClientWithDialer creates an SSRF-protected HTTP client with a custom resolver/dialer for testing.
func NewBYOKHTTPClientWithDialer(resolver ssrfResolver, dialContext func(ctx context.Context, network, address string) (net.Conn, error), timeout time.Duration) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if dialContext == nil {
		netDialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		dialContext = netDialer.DialContext
	}

	dialer := &protectedFetchDialer{
		resolver:    resolver,
		dialContext: dialContext,
		getProtection: func() (*common.SSRFProtection, bool, error) {
			return common.BYOKSSRFProtection, true, nil
		},
	}

	transport := &http.Transport{
		MaxIdleConns:        common.RelayMaxIdleConns,
		MaxIdleConnsPerHost: common.RelayMaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(common.RelayIdleConnTimeout) * time.Second,
		ForceAttemptHTTP2:   true,
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         dialer.DialContext,
	}

	client := &http.Client{
		Transport: &byokRoundTripper{
			transport: transport,
		},
		CheckRedirect: CheckBYOKRedirect,
		Timeout:       timeout,
	}
	return client
}

// GetBYOKHttpClient returns an SSRF-protected HTTP client for BYOK relay requests.
func GetBYOKHttpClient(rawProxyURL string, settings dto.ChannelSettings) (*http.Client, error) {
	trimmedProxyURL := strings.TrimSpace(rawProxyURL)
	key := trimmedProxyURL

	byokClientsMu.RLock()
	if client, ok := byokClients[key]; ok {
		byokClientsMu.RUnlock()
		return client, nil
	}
	byokClientsMu.RUnlock()

	byokClientsMu.Lock()
	defer byokClientsMu.Unlock()

	if client, ok := byokClients[key]; ok {
		return client, nil
	}

	var proxyFunc func(*http.Request) (*url.URL, error)
	if trimmedProxyURL != "" {
		parsedProxyURL, _, err := common.ParseProxyURLRuntime(trimmedProxyURL)
		if err != nil {
			return nil, err
		}
		proxyFunc = http.ProxyURL(parsedProxyURL)
	} else {
		proxyFunc = http.ProxyFromEnvironment
	}

	dialer := &protectedFetchDialer{
		resolver: net.DefaultResolver,
		dialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		getProtection: func() (*common.SSRFProtection, bool, error) {
			return common.BYOKSSRFProtection, true, nil
		},
	}

	transport := &http.Transport{
		MaxIdleConns:        common.RelayMaxIdleConns,
		MaxIdleConnsPerHost: common.RelayMaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(common.RelayIdleConnTimeout) * time.Second,
		ForceAttemptHTTP2:   true,
		Proxy:               proxyFunc,
		DialContext:         dialer.DialContext,
	}

	client := &http.Client{
		Transport: &byokRoundTripper{
			transport: transport,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if common.RelayTimeout != 0 {
		client.Timeout = time.Duration(common.RelayTimeout) * time.Second
	}

	byokClients[key] = client
	return client, nil
}
