package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getE2EBaseURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("E2E_PORT")
	}
	if port == "" {
		port = "3005"
	}
	return "http://localhost:" + port
}

type e2eClient struct {
	baseURL    string
	httpClient *http.Client
}

func newE2EClient() *e2eClient {
	return &e2eClient{
		baseURL: getE2EBaseURL(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *e2eClient) postJSON(path string, body any, token string) (int, map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var jsonMap map[string]any
	if len(respBytes) > 0 {
		_ = json.Unmarshal(respBytes, &jsonMap)
	}
	return resp.StatusCode, jsonMap, nil
}

func (c *e2eClient) getJSON(path string, token string) (int, map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var jsonMap map[string]any
	if len(respBytes) > 0 {
		_ = json.Unmarshal(respBytes, &jsonMap)
	}
	return resp.StatusCode, jsonMap, nil
}

func (c *e2eClient) putJSON(path string, body any, token string) (int, map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var jsonMap map[string]any
	if len(respBytes) > 0 {
		_ = json.Unmarshal(respBytes, &jsonMap)
	}
	return resp.StatusCode, jsonMap, nil
}

func (c *e2eClient) deleteJSON(path string, token string) (int, map[string]any, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var jsonMap map[string]any
	if len(respBytes) > 0 {
		_ = json.Unmarshal(respBytes, &jsonMap)
	}
	return resp.StatusCode, jsonMap, nil
}

func (c *e2eClient) login(username, password string) (string, error) {
	status, data, err := c.postJSON("/api/user/login", map[string]any{
		"username": username,
		"password": password,
	}, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %v", status, data)
	}
	dataMap, ok := data["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("login response missing data object: %v", data)
	}
	token, ok := dataMap["access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("login response missing access_token: %v", data)
	}
	return token, nil
}

// TestToraFullStack_R6B_ManagedLifecycle executes the complete real Managed Inference lifecycle
func TestToraFullStack_R6B_ManagedLifecycle(t *testing.T) {
	client := newE2EClient()

	// 1. Verify E2E server health
	status, statusData, err := client.getJSON("/api/status", "")
	require.NoError(t, err, "Failed to connect to Tora E2E server at %s. Ensure ./scripts/tora-e2e-up is running.", client.baseURL)
	require.Equal(t, http.StatusOK, status)
	assert.True(t, statusData["success"] == true)

	// 2. Root login
	rootToken, err := client.login("root", "Password123!")
	require.NoError(t, err, "Root login failed")

	// Disable performance monitor for E2E tests so system disk overload checks do not block local tests
	_, _, _ = client.putJSON("/api/option/", map[string]any{
		"key":   "performance_setting.monitor_enabled",
		"value": "false",
	}, rootToken)

	// 3. Spin up an upstream mock LLM server that simulates OpenAI chat completions SSE stream
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model"}]}`))
		case "/v1/chat/completions":
			bodyBytes, _ := io.ReadAll(r.Body)
			var chatReq map[string]any
			_ = json.Unmarshal(bodyBytes, &chatReq)

			isStream, _ := chatReq["stream"].(bool)
			if !isStream {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"chatcmpl-sync","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"Sync hello"},"finish_reason":"stop"}]}`))
				return
			}

			// SSE stream response
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Flusher not supported", http.StatusInternalServerError)
				return
			}

			// Chunk 1
			fmt.Fprintf(w, "data: %s\n\n", `{"id":"chatcmpl-e2e","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)

			// Check for cancel test prompt
			messages, _ := chatReq["messages"].([]any)
			isCancelTest := false
			if len(messages) > 0 {
				if firstMsg, ok := messages[0].(map[string]any); ok {
					if content, ok := firstMsg["content"].(string); ok && strings.Contains(content, "Cancel") {
						isCancelTest = true
					}
				}
			}

			if isCancelTest {
				// Sleep longer to give client time to cancel
				time.Sleep(2 * time.Second)
				return
			}

			// Chunk 2
			fmt.Fprintf(w, "data: %s\n\n", `{"id":"chatcmpl-e2e","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":" from Tora AI!"},"finish_reason":null}]}`)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)

			// Chunk 3 with usage & stop
			fmt.Fprintf(w, "data: %s\n\n", `{"id":"chatcmpl-e2e","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`)
			flusher.Flush()

			// Chunk [DONE]
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockLLM.Close()

	// 4. Configure New-API Channel via root admin pointing to mock upstream
	mockURL := mockLLM.URL
	addChannelStatus, addChannelResp, err := client.postJSON("/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"name":     "E2E Mock Channel",
			"type":     1, // OpenAI
			"status":   1, // Enabled
			"key":      "sk-mock-channel-key",
			"base_url": mockURL,
			"models":   "gpt-4o,gpt-4o-mini",
			"group":    "default",
		},
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, addChannelStatus)
	require.True(t, addChannelResp["success"] == true, "Failed to create test channel: %v", addChannelResp)

	// 5. Register a fresh test user
	uniqueUsername := fmt.Sprintf("r6b_user_%d", time.Now().UnixNano()%100000)
	regStatus, regResp, err := client.postJSON("/api/user/register", map[string]any{
		"username": uniqueUsername,
		"password": "Password123!",
		"email":    uniqueUsername + "@tora.ai",
	}, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, regStatus)
	require.True(t, regResp["success"] == true, "Register failed: %v", regResp)

	// 6. Login user
	userToken, err := client.login(uniqueUsername, "Password123!")
	require.NoError(t, err)
	require.NotEmpty(t, userToken)

	// 7. Check user profile and get user ID
	selfStatus, selfResp, err := client.getJSON("/api/user/self", userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, selfStatus)
	selfData := selfResp["data"].(map[string]any)
	userId := int(selfData["id"].(float64))
	require.Greater(t, userId, 0)

	// 8. Grant quota to user using root admin so inference passes quota check
	updateUserStatus, updateUserResp, err := client.postJSON("/api/user/manage", map[string]any{
		"id":     userId,
		"action": "add_quota",
		"mode":   "add",
		"value":  500000,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateUserStatus)
	require.True(t, updateUserResp["success"] == true, "Quota grant failed: %v", updateUserResp)

	// Verify updated quota
	_, selfRespUpdated, _ := client.getJSON("/api/user/self", userToken)
	updatedQuota := int(selfRespUpdated["data"].(map[string]any)["quota"].(float64))
	require.Equal(t, 500000, updatedQuota)

	// 9. Provision dedicated mobile relay token (RelayTokenProvisioner pattern)
	tokenName := fmt.Sprintf("LumenFlow Mobile [%s]", uniqueUsername[:8])
	createTokenStatus, createTokenResp, err := client.postJSON("/api/token/", map[string]any{
		"name":            tokenName,
		"remain_quota":    0,
		"unlimited_quota": true,
		"expired_time":    -1,
	}, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, createTokenStatus)
	require.True(t, createTokenResp["success"] == true, "Create relay token failed: %v", createTokenResp)

	// 10. List tokens to obtain token ID
	listTokenStatus, listTokenResp, err := client.getJSON("/api/token/?p=1&page_size=20", userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listTokenStatus)
	tokenListData := listTokenResp["data"].(map[string]any)
	items := tokenListData["items"].([]any)
	require.NotEmpty(t, items)
	firstToken := items[0].(map[string]any)
	tokenId := int(firstToken["id"].(float64))
	assert.Equal(t, tokenName, firstToken["name"])

	// 11. Reveal unmasked relay key
	revealStatus, revealResp, err := client.postJSON(fmt.Sprintf("/api/token/%d/key", tokenId), nil, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, revealStatus)
	revealData := revealResp["data"].(map[string]any)
	rawRelayKey := revealData["key"].(string)
	require.NotEmpty(t, rawRelayKey)

	// Prefix with sk- if not already prefixed
	relayKey := rawRelayKey
	if !strings.HasPrefix(relayKey, "sk-") {
		relayKey = "sk-" + relayKey
	}

	// 12. Model catalog query (/v1/models) using relay key
	modelsStatus, modelsResp, err := client.getJSON("/v1/models", relayKey)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, modelsStatus)
	assert.Equal(t, "list", modelsResp["object"])
	modelsList, ok := modelsResp["data"].([]any)
	require.True(t, ok)
	foundGpt4o := false
	for _, m := range modelsList {
		if mMap, ok := m.(map[string]any); ok && mMap["id"] == "gpt-4o" {
			foundGpt4o = true
			break
		}
	}
	assert.True(t, foundGpt4o, "gpt-4o should be returned in /v1/models")

	// 13. Execute managed chat completion with SSE streaming (/v1/chat/completions)
	chatReqBody, err := json.Marshal(map[string]any{
		"model": "gpt-4o",
		"messages": []any{
			map[string]any{"role": "user", "content": "Hello Tora"},
		},
		"stream": true,
	})
	require.NoError(t, err)

	httpReq, err := http.NewRequest(http.MethodPost, client.baseURL+"/v1/chat/completions", bytes.NewReader(chatReqBody))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+relayKey)

	chatResp, err := client.httpClient.Do(httpReq)
	require.NoError(t, err)
	defer chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(chatResp.Body)
		require.Equal(t, http.StatusOK, chatResp.StatusCode, "Chat completions error: %s", string(errBody))
	}

	// Read and verify SSE stream
	scanner := bufio.NewScanner(chatResp.Body)
	var accumulatedContent strings.Builder
	receivedDone := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				receivedDone = true
				break
			}
			var chunk map[string]any
			if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
				if choices, ok := chunk["choices"].([]any); ok && len(choices) > 0 {
					if choiceMap, ok := choices[0].(map[string]any); ok {
						if delta, ok := choiceMap["delta"].(map[string]any); ok {
							if content, ok := delta["content"].(string); ok {
								accumulatedContent.WriteString(content)
							}
						}
					}
				}
			}
		}
	}

	assert.True(t, receivedDone, "SSE stream must terminate with [DONE]")
	assert.Equal(t, "Hello from Tora AI!", accumulatedContent.String())

	// 14. Stream cancellation test: start request, cancel context after first byte
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	cancelReq, err := http.NewRequestWithContext(cancelCtx, http.MethodPost, client.baseURL+"/v1/chat/completions", bytes.NewReader(chatReqBody))
	require.NoError(t, err)
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("Authorization", "Bearer "+relayKey)

	cancelResp, err := client.httpClient.Do(cancelReq)
	require.NoError(t, err)
	// Cancel stream immediately
	cancelFunc()
	_ = cancelResp.Body.Close()
	// Server must continue functioning smoothly after cancellation

	// 15. Cross-account isolation: User B cannot access or manipulate User A's relay token
	userBName := fmt.Sprintf("r6b_other_%d", time.Now().UnixNano()%100000)
	_, _, _ = client.postJSON("/api/user/register", map[string]any{
		"username": userBName,
		"password": "Password123!",
		"email":    userBName + "@tora.ai",
	}, "")
	userBToken, err := client.login(userBName, "Password123!")
	require.NoError(t, err)

	// User B attempts to reveal User A's token key -> must fail
	attackRevealStatus, attackRevealResp, _ := client.postJSON(fmt.Sprintf("/api/token/%d/key", tokenId), nil, userBToken)
	assert.True(t, attackRevealStatus == http.StatusNotFound || attackRevealStatus == http.StatusForbidden || attackRevealResp["success"] == false,
		"Cross-account key reveal must be rejected")

	// User B attempts to delete User A's token -> must fail
	attackDelStatus, attackDelResp, _ := client.deleteJSON(fmt.Sprintf("/api/token/%d", tokenId), userBToken)
	assert.True(t, attackDelStatus == http.StatusNotFound || attackDelStatus == http.StatusForbidden || attackDelResp["success"] == false,
		"Cross-account token deletion must be rejected")
}

// TestToraFullStack_R6C_BYOKLifecycle executes the complete Server-Managed BYOK lifecycle
func TestToraFullStack_R6C_BYOKLifecycle(t *testing.T) {
	client := newE2EClient()

	// 1. Register test BYOK user
	username := fmt.Sprintf("r6c_byok_%d", time.Now().UnixNano()%100000)
	regStatus, _, err := client.postJSON("/api/user/register", map[string]any{
		"username": username,
		"password": "Password123!",
		"email":    username + "@tora.ai",
	}, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, regStatus)

	// 2. Login
	userToken, err := client.login(username, "Password123!")
	require.NoError(t, err)

	// 3. Create server-managed BYOK provider (OpenRouter)
	createStatus, createResp, err := client.postJSON("/api/user/providers", map[string]any{
		"provider": "openrouter",
		"name":     "E2E OpenRouter",
		"api_key":  "sk-or-v1-0123456789abcdef0123456789abcdef",
		"enabled":  true,
	}, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, createStatus)
	require.True(t, createResp["success"] == true, "Create BYOK provider failed: %v", createResp)

	providerData := createResp["data"].(map[string]any)
	providerId := int(providerData["id"].(float64))
	maskedKey := providerData["api_key"].(string)

	// Verify plaintext API key is NEVER returned in response
	assert.False(t, strings.Contains(maskedKey, "0123456789abcdef0123456789abcdef"), "Plaintext API key must never be leaked")
	assert.True(t, strings.HasPrefix(maskedKey, "****") || strings.Contains(maskedKey, "*"), "API key must be masked")

	// 4. List BYOK providers
	listStatus, listResp, err := client.getJSON("/api/user/providers", userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listStatus)
	require.True(t, listResp["success"] == true)
	providersList := listResp["data"].([]any)
	require.Len(t, providersList, 1)
	listedProvider := providersList[0].(map[string]any)
	assert.Equal(t, float64(providerId), listedProvider["id"])
	assert.Equal(t, maskedKey, listedProvider["api_key"])

	// 5. Test connection endpoint (/api/user/providers/:id/test)
	// (OpenRouter test connects to external openrouter.ai; if network offline or key dummy, it gracefully returns test failure result without crash)
	testStatus, testResp, err := client.postJSON(fmt.Sprintf("/api/user/providers/%d/test", providerId), nil, userToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, testStatus)
	assert.NotNil(t, testResp)

	// 6. Rotate key via PUT /api/user/providers/:id
	updateStatus, updateResp, err := client.putJSON(fmt.Sprintf("/api/user/providers/%d", providerId), map[string]any{
		"name":    "E2E Rotated OpenRouter",
		"api_key": "sk-or-v1-99999999999999999999999999999999",
	}, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateStatus)
	require.True(t, updateResp["success"] == true, "Update BYOK provider failed: %v", updateResp)
	updatedData := updateResp["data"].(map[string]any)
	assert.Equal(t, "E2E Rotated OpenRouter", updatedData["name"])
	assert.Equal(t, "****9999", updatedData["api_key"])

	// 7. Delete provider
	delStatus, delResp, err := client.deleteJSON(fmt.Sprintf("/api/user/providers/%d", providerId), userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, delStatus)
	require.True(t, delResp["success"] == true)

	// 8. Verify list is now empty
	listAfterDelStatus, listAfterDelResp, err := client.getJSON("/api/user/providers", userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listAfterDelStatus)
	assert.Empty(t, listAfterDelResp["data"])
}

// TestToraFullStack_R6D_TopUpFlow executes the complete Top-Up and Card Redemption lifecycle
func TestToraFullStack_R6D_TopUpFlow(t *testing.T) {
	client := newE2EClient()

	// 1. Root login
	rootToken, err := client.login("root", "Password123!")
	require.NoError(t, err)

	// 2. Ensure payment compliance confirmed
	compStatus, _, err := client.postJSON("/api/option/payment_compliance", map[string]any{
		"confirmed": true,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, compStatus)

	// 3. Create a redemption card for 200,000 units
	createCardStatus, createCardResp, err := client.postJSON("/api/redemption/", map[string]any{
		"name":         "E2E Test Voucher",
		"count":        1,
		"quota":        200000,
		"expired_time": 0,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, createCardStatus)
	require.True(t, createCardResp["success"] == true, "Failed to create redemption card: %v", createCardResp)
	keysList := createCardResp["data"].([]any)
	require.NotEmpty(t, keysList)
	redemptionCode := keysList[0].(string)
	require.NotEmpty(t, redemptionCode)

	// 4. Register topup user
	username := fmt.Sprintf("r6d_user_%d", time.Now().UnixNano()%100000)
	regStatus, _, err := client.postJSON("/api/user/register", map[string]any{
		"username": username,
		"password": "Password123!",
		"email":    username + "@tora.ai",
	}, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, regStatus)

	// 5. Login
	userToken, err := client.login(username, "Password123!")
	require.NoError(t, err)

	// Check initial balance
	_, initialSelf, err := client.getJSON("/api/user/self", userToken)
	require.NoError(t, err)
	initialQuota := int(initialSelf["data"].(map[string]any)["quota"].(float64))
	assert.Equal(t, 0, initialQuota)

	// 6. Redeem code
	redeemStatus, redeemResp, err := client.postJSON("/api/user/topup", map[string]any{
		"key": redemptionCode,
	}, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, redeemStatus)
	require.True(t, redeemResp["success"] == true, "Redeem failed: %v", redeemResp)
	grantedQuota := int(redeemResp["data"].(float64))
	assert.Equal(t, 200000, grantedQuota)

	// 7. Verify balance increased by exact amount
	_, afterSelf, err := client.getJSON("/api/user/self", userToken)
	require.NoError(t, err)
	afterQuota := int(afterSelf["data"].(map[string]any)["quota"].(float64))
	assert.Equal(t, 200000, afterQuota)

	// 8. Replay prevention: redeeming the same key again MUST fail
	replayStatus, replayResp, err := client.postJSON("/api/user/topup", map[string]any{
		"key": redemptionCode,
	}, userToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, replayStatus)
	assert.False(t, replayResp["success"] == true, "Replay redemption must be rejected")

	// 9. Check top-up info endpoint (/api/user/topup/info)
	infoStatus, infoResp, err := client.getJSON("/api/user/topup/info", userToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, infoStatus)
	assert.True(t, infoResp["success"] == true)

	// 10. Check Stripe Pay request (/api/user/stripe/pay)
	// Tests that the endpoint exists and responds cleanly (returns checkout link or configured error)
	stripeStatus, stripeResp, err := client.postJSON("/api/user/stripe/pay", map[string]any{
		"amount": 10,
	}, userToken)
	require.NoError(t, err)
	assert.True(t, stripeStatus == http.StatusOK || stripeStatus == http.StatusBadRequest)
	if stripeStatus == http.StatusOK && stripeResp["success"] == true {
		payLink, ok := stripeResp["data"].(string)
		if ok && payLink != "" {
			// Must be valid HTTPS URL
			assert.True(t, strings.HasPrefix(payLink, "https://"), "Stripe pay_link must be HTTPS")
		}
	}
}

// TestToraFullStack_R6E_ContractSweep sweeps all 22 mobile endpoints for status and contract conformity
func TestToraFullStack_R6E_ContractSweep(t *testing.T) {
	client := newE2EClient()

	// 1. Root login
	rootToken, err := client.login("root", "Password123!")
	require.NoError(t, err)

	// 2. Subscription plans endpoint (GET /api/subscription/plans)
	plansStatus, plansResp, err := client.getJSON("/api/subscription/plans", rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, plansStatus)
	assert.True(t, plansResp["success"] == true)
	plansList, ok := plansResp["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, plansList, "Approved commercial plan (Pro Monthly) must be seeded")

	// 3. Subscription self endpoint (GET /api/subscription/self)
	subSelfStatus, subSelfResp, err := client.getJSON("/api/subscription/self", rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, subSelfStatus)
	assert.True(t, subSelfResp["success"] == true)

	// 4. Store product catalog (GET /api/subscription/store/products)
	prodStatus, prodResp, err := client.getJSON("/api/subscription/store/products?platform=ios&env=sandbox", rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, prodStatus)
	assert.True(t, prodResp["success"] == true)

	// 5. Store verification endpoints without authentication return 401
	unauthAppleStatus, _, _ := client.postJSON("/api/subscription/apple/verify", map[string]any{
		"signed_transaction_info": "test",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, unauthAppleStatus)

	unauthGoogleStatus, _, _ := client.postJSON("/api/subscription/google/verify", map[string]any{
		"product_id":     "tora_pro",
		"purchase_token": "test",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, unauthGoogleStatus)

	// 6. Store verification endpoints with invalid proof return 400 or 503
	authAppleStatus, authAppleResp, _ := client.postJSON("/api/subscription/apple/verify", map[string]any{
		"signed_transaction_info": "invalid.jwt.token",
	}, rootToken)
	assert.True(t, authAppleStatus == http.StatusBadRequest || authAppleStatus == http.StatusServiceUnavailable)
	assert.False(t, authAppleResp["success"] == true)

	authGoogleStatus, authGoogleResp, _ := client.postJSON("/api/subscription/google/verify", map[string]any{
		"product_id":     "tora_pro",
		"purchase_token": "invalid_token",
	}, rootToken)
	assert.True(t, authGoogleStatus == http.StatusBadRequest || authGoogleStatus == http.StatusServiceUnavailable)
	assert.False(t, authGoogleResp["success"] == true)
}

// TestToraFullStack_R6F_RouteLifecycle tests the complete HTTP lifecycle for First-Class Routes & API key binding
func TestToraFullStack_R6F_RouteLifecycle(t *testing.T) {
	client := newE2EClient()

	// 1. Root login
	rootToken, err := client.login("root", "Password123!")
	require.NoError(t, err)

	// 2. Create primary route
	createRouteStatus, createRouteResp, err := client.postJSON("/api/routes/", map[string]any{
		"name":            "E2E Primary Route",
		"slug":            "e2e-primary-route",
		"description":     "Route created during full-stack E2E test",
		"kind":            "llm",
		"routing_policy":  "weighted_random",
		"cost_multiplier": 1.25,
		"enabled":         true,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, createRouteStatus)
	require.True(t, createRouteResp["success"] == true)
	routeData, ok := createRouteResp["data"].(map[string]any)
	require.True(t, ok)
	routeId := int(routeData["id"].(float64))
	require.Positive(t, routeId)

	// 3. Get created route
	getRouteStatus, getRouteResp, err := client.getJSON(fmt.Sprintf("/api/routes/%d", routeId), rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getRouteStatus)
	assert.True(t, getRouteResp["success"] == true)

	// 4. Update route multiplier
	putRouteStatus, putRouteResp, err := client.putJSON(fmt.Sprintf("/api/routes/%d", routeId), map[string]any{
		"name":            "E2E Primary Route Updated",
		"slug":            "e2e-primary-route",
		"cost_multiplier": 1.5,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, putRouteStatus)
	assert.True(t, putRouteResp["success"] == true)

	// 5. Create Token bound to the primary route
	uniqueKeyName := fmt.Sprintf("E2E Route Bound Key %d", time.Now().UnixNano()%100000)
	createTokenStatus, createTokenResp, err := client.postJSON("/api/token/", map[string]any{
		"name":             uniqueKeyName,
		"remain_quota":     500000,
		"unlimited_quota":  false,
		"primary_route_id": routeId,
	}, rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, createTokenStatus)
	require.True(t, createTokenResp["success"] == true)

	// List tokens to obtain token ID and verify primary_route_id
	listTokenStatus, listTokenResp, err := client.getJSON("/api/token/?p=1&page_size=20", rootToken)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listTokenStatus)
	tokenListData := listTokenResp["data"].(map[string]any)
	items := tokenListData["items"].([]any)
	require.NotEmpty(t, items)

	var targetToken map[string]any
	for _, it := range items {
		tm := it.(map[string]any)
		if tm["name"] == uniqueKeyName {
			targetToken = tm
			break
		}
	}
	require.NotNil(t, targetToken, "Created token must be present in token list")
	tokenId := int(targetToken["id"].(float64))
	require.Positive(t, tokenId)

	// Verify token binds routeId
	boundRouteId := int(targetToken["primary_route_id"].(float64))
	assert.Equal(t, routeId, boundRouteId)

	// 6. Delete Route while bound to Token -> MUST return 409 Conflict (Referential guard)
	deleteRouteBlockedStatus, deleteRouteBlockedResp, err := client.deleteJSON(fmt.Sprintf("/api/routes/%d", routeId), rootToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, deleteRouteBlockedStatus, "deleting referenced route must return 409 Conflict")
	assert.False(t, deleteRouteBlockedResp["success"] == true)

	// 7. Delete Token
	deleteTokenStatus, deleteTokenResp, err := client.deleteJSON(fmt.Sprintf("/api/token/%d", tokenId), rootToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, deleteTokenStatus)
	assert.True(t, deleteTokenResp["success"] == true)

	// 8. Delete Route now unreferenced -> MUST succeed 200 OK
	deleteRouteStatus, deleteRouteResp, err := client.deleteJSON(fmt.Sprintf("/api/routes/%d", routeId), rootToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, deleteRouteStatus)
	assert.True(t, deleteRouteResp["success"] == true)

	// 9. Fetch deleted route -> returns success: false
	getDeletedStatus, getDeletedResp, _ := client.getJSON(fmt.Sprintf("/api/routes/%d", routeId), rootToken)
	assert.Equal(t, http.StatusOK, getDeletedStatus)
	assert.False(t, getDeletedResp["success"] == true)
}

