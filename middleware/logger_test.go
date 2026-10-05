package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLogPath(t *testing.T) {
	const geminiKeyCanary = "GEMINI_KEY_CANARY_sk-secret12345"
	const waffoKeyCanary = "WAFFO_PRIVATE_KEY_CANARY_RSA12345"
	const tokenCanary = "TOKEN_CANARY_xyz987"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Gemini query key redacted",
			input:    "/v1beta/models/gemini-1.5-flash:generateContent?key=" + geminiKeyCanary,
			expected: "/v1beta/models/gemini-1.5-flash:generateContent?key=[REDACTED]",
		},
		{
			name:     "Gemini query key with multiple params",
			input:    "/v1beta/models/gemini-1.5-pro:generateContent?key=" + geminiKeyCanary + "&alt=sse",
			expected: "/v1beta/models/gemini-1.5-pro:generateContent?key=[REDACTED]&alt=sse",
		},
		{
			name:     "Case-insensitive KEY param",
			input:    "/v1beta/models/gemini-1.5-flash:generateContent?KEY=" + geminiKeyCanary,
			expected: "/v1beta/models/gemini-1.5-flash:generateContent?KEY=[REDACTED]",
		},
		{
			name:     "Waffo Pancake private_key query redacted",
			input:    "/api/option/waffo-pancake/catalog?merchant_id=123&private_key=" + waffoKeyCanary,
			expected: "/api/option/waffo-pancake/catalog?merchant_id=123&private_key=[REDACTED]",
		},
		{
			name:     "OAuth callback entirely stripped query",
			input:    "/api/oauth/github?code=secret_code_123&state=secret_state_456",
			expected: "/api/oauth/github",
		},
		{
			name:     "Non-sensitive query params preserved",
			input:    "/api/user?page=1&size=20&keyword=hello",
			expected: "/api/user?page=1&size=20&keyword=hello",
		},
		{
			name:     "Multiple sensitive query params redacted",
			input:    "/relay?token=" + tokenCanary + "&api_key=secret_api_key&secret=my_secret",
			expected: "/relay?token=[REDACTED]&api_key=[REDACTED]&secret=[REDACTED]",
		},
		{
			name:     "Path without query string",
			input:    "/v1/chat/completions",
			expected: "/v1/chat/completions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := SanitizeLogPath(tc.input)
			assert.Equal(t, tc.expected, actual)
			assert.NotContains(t, actual, geminiKeyCanary)
			assert.NotContains(t, actual, waffoKeyCanary)
			assert.NotContains(t, actual, tokenCanary)
		})
	}
}
