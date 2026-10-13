package channel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoRequestBYOKBlocksPrivateDestinations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateURLs := []string{
		"http://127.0.0.1:8000/v1/chat/completions",
		"http://127.0.0.2:80/v1/chat/completions",
		"http://10.0.0.1:80/v1/chat/completions",
		"http://172.16.0.5:8000/v1/chat/completions",
		"http://192.168.1.1:80/v1/chat/completions",
		"http://169.254.169.254/v1/chat/completions",
		"http://localhost:8080/v1/chat/completions",
		"http://[::1]:8080/v1/chat/completions",
		"http://[fc00::1]:80/v1/chat/completions",
		"http://[fe80::1]:80/v1/chat/completions",
	}

	for _, targetURL := range privateURLs {
		t.Run(targetURL, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

			req, err := http.NewRequest(http.MethodPost, targetURL, nil)
			require.NoError(t, err)

			info := &relaycommon.RelayInfo{
				IsBYOK:      true,
				ChannelMeta: &relaycommon.ChannelMeta{},
			}

			resp, err := doRequest(ctx, req, info)
			if resp != nil {
				resp.Body.Close()
			}
			assert.Error(t, err, "BYOK relay must reject target URL: %s", targetURL)
		})
	}
}
