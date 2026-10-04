package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBYOKHTTPClientRejectsDirectPrivateIPs(t *testing.T) {
	client := NewBYOKHTTPClient(2 * time.Second)

	privateURLs := []string{
		"http://127.0.0.1:8080/models",
		"http://127.0.0.2:80/models",
		"http://10.0.0.1:80/models",
		"http://172.16.0.5:8000/models",
		"http://192.168.1.100:80/models",
		"http://169.254.169.254/models",
		"http://localhost:8080/models",
		"http://[::1]:8080/models",
		"http://[fc00::1]:80/models",
		"http://[fe80::1]:80/models",
	}

	for _, u := range privateURLs {
		t.Run(u, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, u, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			assert.Error(t, err, "outbound request to %s must be rejected", u)
		})
	}
}

func TestBYOKHTTPClientRejectsHostnameResolvingToPrivateIP(t *testing.T) {
	// Mock resolver simulating DNS resolution to a private IP (DNS rebinding simulation)
	mockResolver := staticSSRFResolver{
		"rebind.example.com": {
			{IP: net.ParseIP("127.0.0.1")},
		},
		"metadata.internal.com": {
			{IP: net.ParseIP("169.254.169.254")},
		},
		"intranet.corp.com": {
			{IP: net.ParseIP("10.10.10.10")},
		},
	}

	dialCalled := false
	client := NewBYOKHTTPClientWithDialer(mockResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		dialCalled = true
		return nil, fmt.Errorf("dial should not be reached for %s", address)
	}, 2*time.Second)

	testCases := []string{
		"http://rebind.example.com:80/models",
		"http://metadata.internal.com:80/models",
		"http://intranet.corp.com:80/models",
	}

	for _, u := range testCases {
		t.Run(u, func(t *testing.T) {
			dialCalled = false
			req, err := http.NewRequest(http.MethodGet, u, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			assert.Error(t, err)
			assert.False(t, dialCalled, "dial must never be attempted for private resolved IP: %s", u)
		})
	}
}

func TestBYOKHTTPClientBlocksRedirectToPrivateIP(t *testing.T) {
	client := NewBYOKHTTPClient(3 * time.Second)

	// Test directly via CheckBYOKRedirect
	via := []*http.Request{
		httptest.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil),
	}

	testCases := []struct {
		dest        string
		expectedErr string
	}{
		{"http://127.0.0.1:8080/internal/secret", "private IP address not allowed"},
		{"http://169.254.169.254/latest/meta-data", "private IP address not allowed: 169.254.169.254"},
		{"http://localhost:8080/admin", "domain in blacklist: localhost"},
		{"http://[::1]:8080/secret", "private IP address not allowed: ::1"},
		{"http://[fc00::1]:8080/secret", "private IP address not allowed: fc00::1"},
	}

	for _, tc := range testCases {
		redirectReq, err := http.NewRequest(http.MethodGet, tc.dest, nil)
		require.NoError(t, err)

		err = client.CheckRedirect(redirectReq, via)
		assert.Error(t, err, "destination %s should be blocked", tc.dest)
		assert.Contains(t, err.Error(), tc.expectedErr)
	}

	// Test through http.Client.Do redirect loop
	type mockRedirectTransport struct {
		location string
	}
	redirectTransport := &mockRedirectTransportRoundTripper{
		location: "http://127.0.0.1:8080/internal/secret",
	}
	clientWithMock := &http.Client{
		Transport:     redirectTransport,
		CheckRedirect: CheckBYOKRedirect,
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	resp, err := clientWithMock.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect to http://127.0.0.1:8080/internal/secret blocked: private IP address not allowed")
}

type mockRedirectTransportRoundTripper struct {
	location string
}

func (m *mockRedirectTransportRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusFound,
		Header: http.Header{
			"Location": []string{m.location},
		},
		Request: req,
	}, nil
}

func TestBYOKHTTPClientBlocksRedirectChainLimit(t *testing.T) {
	client := NewBYOKHTTPClient(3 * time.Second)

	via3 := []*http.Request{
		httptest.NewRequest(http.MethodGet, "https://api.openai.com/v1/step1", nil),
		httptest.NewRequest(http.MethodGet, "https://api.openai.com/v1/step2", nil),
		httptest.NewRequest(http.MethodGet, "https://api.openai.com/v1/step3", nil),
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/step4", nil)
	require.NoError(t, err)

	err = client.CheckRedirect(req, via3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after 3 redirects")
}
