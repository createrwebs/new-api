package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type CreateUserProviderRequest struct {
	Provider string `json:"provider" binding:"required"`
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key" binding:"required"`
	Enabled  *bool  `json:"enabled"`
}

type UpdateUserProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Enabled *bool  `json:"enabled"`
}

// ProviderTester abstracts outbound lightweight credential validation.
type ProviderTester interface {
	Test(ctx context.Context, provider *model.UserProvider, apiKey string) error
}

type defaultHTTPProviderTester struct {
	client *http.Client
}

func (t *defaultHTTPProviderTester) Test(ctx context.Context, provider *model.UserProvider, apiKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	testURL := ""
	if provider.Provider == model.ProviderOpenRouter {
		testURL = "https://openrouter.ai/api/v1/models"
	} else {
		baseURL := strings.TrimRight(provider.BaseURL, "/")
		testURL = baseURL + "/models"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return errors.New("failed to initialize test request")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "ToraAI-BYOK-Tester/1.0")

	client := t.client
	if client == nil {
		client = service.NewBYOKHTTPClient(7 * time.Second)
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("connection timed out (7s limit)")
		}
		if strings.Contains(err.Error(), "private IP address not allowed") ||
			strings.Contains(err.Error(), "SSRF") ||
			strings.Contains(err.Error(), "domain in blacklist") ||
			strings.Contains(err.Error(), "blocked") {
			return fmt.Errorf("SSRF protection: %w", err)
		}
		return errors.New("network error connecting to provider host")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("authentication failed: invalid API key")
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("endpoint not found (verify base URL)")
	}
	return fmt.Errorf("provider returned HTTP status %d", resp.StatusCode)
}

// DefaultProviderTester is the active provider tester; replaceable in unit tests.
var DefaultProviderTester ProviderTester = &defaultHTTPProviderTester{}

// ListUserProviders handles GET /api/user/providers
func ListUserProviders(c *gin.Context) {
	userID := c.GetInt("id")
	providers, err := model.GetUserProviders(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, providers)
}

// CreateUserProvider handles POST /api/user/providers
func CreateUserProvider(c *gin.Context) {
	userID := c.GetInt("id")
	var req CreateUserProviderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request parameters: "+err.Error())
		return
	}

	cleanBaseURL, err := model.ValidateUserProviderInput(req.Provider, req.Name, req.BaseURL)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	provider := &model.UserProvider{
		UserId:   userID,
		Provider: strings.ToLower(strings.TrimSpace(req.Provider)),
		Name:     strings.TrimSpace(req.Name),
		BaseURL:  cleanBaseURL,
		Enabled:  enabled,
	}

	if err := provider.SetAPIKey(req.APIKey); err != nil {
		if errors.Is(err, common.ErrBYOKKeyNotConfigured) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "BYOK encryption is not configured on server",
			})
			return
		}
		common.ApiError(c, err)
		return
	}

	if err := model.CreateUserProvider(provider); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, provider)
}

// UpdateUserProvider handles PUT /api/user/providers/:id
func UpdateUserProvider(c *gin.Context) {
	userID := c.GetInt("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid provider id")
		return
	}

	provider, err := model.GetUserProviderByID(userID, id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req UpdateUserProviderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "invalid request parameters: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Name) != "" {
		provider.Name = strings.TrimSpace(req.Name)
	}

	if provider.Provider == model.ProviderCustom && strings.TrimSpace(req.BaseURL) != "" {
		cleanURL, err := model.NormalizeBaseURL(req.BaseURL)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		provider.BaseURL = cleanURL
	}

	if strings.TrimSpace(req.APIKey) != "" {
		if err := provider.SetAPIKey(req.APIKey); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}

	if err := model.UpdateUserProvider(provider); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, provider)
}

// DeleteUserProvider handles DELETE /api/user/providers/:id
func DeleteUserProvider(c *gin.Context) {
	userID := c.GetInt("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid provider id")
		return
	}

	if err := model.DeleteUserProvider(userID, id); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"id": id})
}

// TestUserProvider handles POST /api/user/providers/:id/test
func TestUserProvider(c *gin.Context) {
	userID := c.GetInt("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid provider id")
		return
	}

	provider, err := model.GetUserProviderByID(userID, id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	apiKey, err := provider.DecryptAPIKey()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "failed to decrypt provider credentials",
		})
		return
	}

	if err := DefaultProviderTester.Test(c.Request.Context(), provider, apiKey); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Connection test failed: " + err.Error(),
		})
		return
	}

	common.ApiSuccess(c, gin.H{
		"message": "Provider connection successful",
	})
}
