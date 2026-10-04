package common

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBYOKSSRFProtectionRejectsLocalhostAndPrivateTargets(t *testing.T) {
	protection := NewBYOKSSRFProtection()

	blockedTargets := []struct {
		name string
		host string
		port int
	}{
		{"localhost", "localhost", 80},
		{"loopback IPv4 127.0.0.1", "127.0.0.1", 8080},
		{"loopback IPv4 range 127.0.1.1", "127.0.1.1", 80},
		{"loopback IPv6 ::1", "::1", 443},
		{"RFC1918 10.0.0.1", "10.0.0.1", 80},
		{"RFC1918 172.16.0.1", "172.16.0.1", 8000},
		{"RFC1918 192.168.1.1", "192.168.1.1", 80},
		{"link-local 169.254.169.254", "169.254.169.254", 80},
		{"private IPv6 ULA fc00::1", "fc00::1", 443},
		{"private IPv6 link-local fe80::1", "fe80::1", 80},
		{"IPv4-mapped loopback", "::ffff:127.0.0.1", 80},
		{"unspecified 0.0.0.0", "0.0.0.0", 80},
		{"internal domain .local", "myhost.local", 80},
		{"internal domain .internal", "service.internal", 80},
		{"internal domain .lan", "router.lan", 80},
	}

	for _, tc := range blockedTargets {
		t.Run(tc.name, func(t *testing.T) {
			err := protection.ValidateNetworkTarget(tc.host, tc.port)
			assert.Error(t, err, "target %s:%d must be rejected by SSRF protection", tc.host, tc.port)
		})
	}
}

func TestBYOKSSRFProtectionRejectsResolvedPrivateIP(t *testing.T) {
	protection := NewBYOKSSRFProtection()

	privateIPs := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.0.1",
		"169.254.169.254",
		"::1",
		"fc00::1",
		"fe80::1",
	}

	for _, ipStr := range privateIPs {
		t.Run(ipStr, func(t *testing.T) {
			ip := net.ParseIP(ipStr)
			require.NotNil(t, ip)
			err := protection.ValidateResolvedIP("safe-looking.com", ip)
			assert.Error(t, err, "resolved IP %s must be rejected", ipStr)
		})
	}
}

func TestBYOKSSRFProtectionAllowsPublicEndpoints(t *testing.T) {
	protection := NewBYOKSSRFProtection()

	publicHosts := []string{
		"openrouter.ai",
		"api.openai.com",
		"api.together.xyz",
		"api.groq.com",
		"8.8.8.8",
		"1.1.1.1",
	}

	for _, host := range publicHosts {
		t.Run(host, func(t *testing.T) {
			err := protection.ValidateNetworkTarget(host, 443)
			assert.NoError(t, err, "public host %s must be allowed", host)
		})
	}

	publicIP := net.ParseIP("104.18.2.1")
	require.NoError(t, protection.ValidateResolvedIP("openrouter.ai", publicIP))
}
