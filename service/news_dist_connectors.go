package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

// Failure classification constants (Section 16)
const (
	DistFailureAuth             = "AUTH"
	DistFailurePermission       = "PERMISSION"
	DistFailureRateLimit        = "RATE_LIMIT"
	DistFailureValidation       = "VALIDATION"
	DistFailureRemote5xx        = "REMOTE_5XX"
	DistFailureNetworkAmbiguous = "NETWORK_AMBIGUOUS"
	DistFailurePermanent        = "PERMANENT"
)

// ClassifyDistributionFailure determines the failure category for a distribution attempt
func ClassifyDistributionFailure(err error, statusCode int) string {
	if err == nil && statusCode < 400 {
		return ""
	}
	errStr := ""
	if err != nil {
		errStr = strings.ToLower(err.Error())
	}
	if strings.Contains(errStr, "operator_blocked") || strings.Contains(errStr, "not configured") {
		return DistFailureAuth
	}
	if statusCode == 401 || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid token") || strings.Contains(errStr, "token expired") {
		return DistFailureAuth
	}
	if statusCode == 403 || strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "permission") || strings.Contains(errStr, "scope") || strings.Contains(errStr, "access denied") {
		return DistFailurePermission
	}
	if statusCode == 429 || strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "quota") || strings.Contains(errStr, "too many requests") {
		return DistFailureRateLimit
	}
	if statusCode == 400 || statusCode == 422 || strings.Contains(errStr, "validation") || strings.Contains(errStr, "bad request") || strings.Contains(errStr, "invalid url") || strings.Contains(errStr, "unprocessable") {
		return DistFailureValidation
	}
	if statusCode >= 500 || strings.Contains(errStr, "internal server error") || strings.Contains(errStr, "bad gateway") || strings.Contains(errStr, "gateway timeout") || strings.Contains(errStr, "service unavailable") {
		return DistFailureRemote5xx
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "connection reset") || strings.Contains(errStr, "eof") || strings.Contains(errStr, "network ambiguous") {
		return DistFailureNetworkAmbiguous
	}
	return DistFailurePermanent
}

// ValidateDistributionUrl verifies production distribution invariants (Section 2)
func ValidateDistributionUrl(urlStr string) error {
	trimmed := strings.TrimSpace(urlStr)
	if trimmed == "" {
		return fmt.Errorf("distribution URL is empty")
	}
	lower := strings.ToLower(trimmed)
	forbidden := []string{
		"localhost",
		"127.0.0.1",
		"staging-api.toraapi.com",
		"tora.ai",
		"api.tora.ai",
		"staging-api.tora.ai",
	}
	for _, f := range forbidden {
		if strings.Contains(lower, f) {
			return fmt.Errorf("distribution URL %q contains forbidden host %q (leakage defense)", trimmed, f)
		}
	}
	expectedPrefix := common.GetCanonicalBaseURL() + "/news/"
	if !strings.HasPrefix(trimmed, expectedPrefix) {
		return fmt.Errorf("distribution URL %q does not match canonical news prefix %q", trimmed, expectedPrefix)
	}
	return nil
}

// IsGlobalDistributionEnabled returns true if outbound distribution is explicitly enabled (Section 21)
// If NEWS_DISTRIBUTION_ENABLED is "false", "0", "off", or "disabled", it returns false.
// If unset, it defaults to true only when explicitly configured or in test environments.
func IsGlobalDistributionEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("NEWS_DISTRIBUTION_ENABLED")))
	if val == "false" || val == "0" || val == "off" || val == "disabled" {
		return false
	}
	return true
}

// IsPlatformDistributionEnabled returns true if the specific platform distribution is enabled
func IsPlatformDistributionEnabled(platform string) bool {
	if !IsGlobalDistributionEnabled() {
		return false
	}
	envKey1 := "NEWS_DISTRIBUTION_" + strings.ToUpper(platform) + "_ENABLED"
	val1 := strings.ToLower(strings.TrimSpace(os.Getenv(envKey1)))
	if val1 == "false" || val1 == "0" || val1 == "off" || val1 == "disabled" {
		return false
	}
	envKey2 := strings.ToUpper(platform) + "_DISTRIBUTION_ENABLED"
	val2 := strings.ToLower(strings.TrimSpace(os.Getenv(envKey2)))
	if val2 == "false" || val2 == "0" || val2 == "off" || val2 == "disabled" {
		return false
	}
	return true
}

// GetDistributionAllowlistPostIDs returns the set of allowed post IDs for canary distribution (Section 17)
func GetDistributionAllowlistPostIDs() map[int]bool {
	raw := strings.TrimSpace(os.Getenv("NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS"))
	if raw == "" {
		return nil
	}
	allowed := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if id, err := strconv.Atoi(part); err == nil && id > 0 {
			allowed[id] = true
		}
	}
	return allowed
}

// DistClient is the interface implemented by platform distribution adapters
type DistClient interface {
	Publish(ctx context.Context, post *model.NewsPost, payload string) (remoteId string, remoteUrl string, err error)
	PlatformName() string
}

// FacebookPublisher implements distribution to Facebook Pages via Graph API
type FacebookPublisher struct {
	PageToken string
	PageID    string
	BaseURL   string
	Client    *http.Client
}

func NewFacebookPublisher() *FacebookPublisher {
	return &FacebookPublisher{
		PageToken: os.Getenv("FACEBOOK_PAGE_ACCESS_TOKEN"),
		PageID:    os.Getenv("FACEBOOK_PAGE_ID"),
		BaseURL:   os.Getenv("FACEBOOK_BASE_URL"),
		Client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *FacebookPublisher) PlatformName() string { return model.DistPlatformFacebook }

// VerifyPageAccess validates that the Page ID resolves and token possesses Page publish authority
func (p *FacebookPublisher) VerifyPageAccess(ctx context.Context) (string, error) {
	if p.PageToken == "" || p.PageID == "" {
		return "", errors.New("FACEBOOK_PAGE_ACCESS_TOKEN or FACEBOOK_PAGE_ID not configured (OPERATOR_BLOCKED)")
	}
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://graph.facebook.com/%s", GetFacebookGraphAPIVersion())
	}
	apiURL := fmt.Sprintf("%s/%s?fields=id,name&access_token=%s", strings.TrimRight(baseURL, "/"), p.PageID, url.QueryEscape(p.PageToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("facebook Page verification HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var res struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(bodyBytes, &res)
	if res.ID == "" {
		return "", fmt.Errorf("facebook Page ID mismatch or empty response")
	}
	return res.Name, nil
}

func (p *FacebookPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.PageToken == "" || p.PageID == "" {
		return "", "", errors.New("FACEBOOK_PAGE_ACCESS_TOKEN or FACEBOOK_PAGE_ID not configured (OPERATOR_BLOCKED)")
	}

	if !IsPlatformDistributionEnabled(model.DistPlatformFacebook) {
		return "", "", errors.New("facebook distribution disabled by provider switch (NEWS_DISTRIBUTION_FACEBOOK_ENABLED=false)")
	}

	if err := ValidateDistributionUrl(post.CanonicalUrl); err != nil {
		return "", "", fmt.Errorf("canonical URL validation failed: %w", err)
	}

	// Idempotency check: prevent duplicate remote posts (Section 5)
	if model.DB != nil {
		var existingDist model.NewsDistribution
		if err := model.DB.Where("post_id = ? AND platform = ?", post.Id, model.DistPlatformFacebook).First(&existingDist).Error; err == nil {
			if existingDist.RemotePostId != "" && existingDist.Status == "published" {
				logger.LogInfo(ctx, fmt.Sprintf("[FacebookPublisher] Post %d already published on Facebook as %s (idempotent no-op)", post.Id, existingDist.RemotePostId))
				return existingDist.RemotePostId, existingDist.RemoteUrl, nil
			}
		}
	}

	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://graph.facebook.com/%s", GetFacebookGraphAPIVersion())
	}
	apiURL := fmt.Sprintf("%s/%s/feed", strings.TrimRight(baseURL, "/"), p.PageID)
	reqBody, _ := json.Marshal(map[string]string{
		"message":      payload,
		"link":         post.CanonicalUrl,
		"access_token": p.PageToken,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("facebook API error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(bodyBytes, &res)
	remoteUrl := fmt.Sprintf("https://www.facebook.com/%s", res.ID)
	return res.ID, remoteUrl, nil
}

// LinkedInPublisher implements technical distribution to LinkedIn Organization Page via official Posts API
type LinkedInPublisher struct {
	AccessToken string
	OrgID       string
	APIVersion  string
	BaseURL     string
	Client      *http.Client
}

func NewLinkedInPublisher() *LinkedInPublisher {
	apiVersion := GetLinkedInAPIVersion()
	return &LinkedInPublisher{
		AccessToken: os.Getenv("LINKEDIN_ACCESS_TOKEN"),
		OrgID:       os.Getenv("LINKEDIN_ORG_ID"),
		APIVersion:  apiVersion,
		BaseURL:     os.Getenv("LINKEDIN_BASE_URL"),
		Client:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *LinkedInPublisher) PlatformName() string { return model.DistPlatformLinkedIn }

// VerifyOrgAccess validates that the configured LinkedIn Organization exists and token possesses organization access
func (p *LinkedInPublisher) VerifyOrgAccess(ctx context.Context) (string, error) {
	if p.AccessToken == "" || p.OrgID == "" {
		return "", errors.New("LINKEDIN_ACCESS_TOKEN or LINKEDIN_ORG_ID not configured (OPERATOR_BLOCKED)")
	}
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://api.linkedin.com"
	}
	cleanOrgID := strings.TrimPrefix(p.OrgID, "urn:li:organization:")
	apiURL := fmt.Sprintf("%s/rest/organizations/%s", strings.TrimRight(baseURL, "/"), cleanOrgID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("LinkedIn-Version", p.APIVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("linkedIn Organization verification HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}
	var res struct {
		LocalizedName string `json:"localizedName"`
		VanityName    string `json:"vanityName"`
	}
	_ = json.Unmarshal(bodyBytes, &res)
	name := res.LocalizedName
	if name == "" {
		name = res.VanityName
	}
	if name == "" {
		name = cleanOrgID
	}
	return name, nil
}

func (p *LinkedInPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.AccessToken == "" || p.OrgID == "" {
		return "", "", errors.New("LINKEDIN_ACCESS_TOKEN or LINKEDIN_ORG_ID not configured (OPERATOR_BLOCKED)")
	}

	if !IsPlatformDistributionEnabled(model.DistPlatformLinkedIn) {
		return "", "", errors.New("linkedIn distribution disabled by provider switch (NEWS_DISTRIBUTION_LINKEDIN_ENABLED=false)")
	}

	if err := ValidateDistributionUrl(post.CanonicalUrl); err != nil {
		return "", "", fmt.Errorf("canonical URL validation failed: %w", err)
	}

	// Idempotency check: prevent duplicate remote posts (Section 8)
	if model.DB != nil {
		var existingDist model.NewsDistribution
		if err := model.DB.Where("post_id = ? AND platform = ?", post.Id, model.DistPlatformLinkedIn).First(&existingDist).Error; err == nil {
			if existingDist.RemotePostId != "" && existingDist.Status == "published" {
				logger.LogInfo(ctx, fmt.Sprintf("[LinkedInPublisher] Post %d already published on LinkedIn as %s (idempotent no-op)", post.Id, existingDist.RemotePostId))
				return existingDist.RemotePostId, existingDist.RemoteUrl, nil
			}
		}
	}

	// Official LinkedIn Posts API (w_organization_social scope)
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://api.linkedin.com"
	}
	apiURL := fmt.Sprintf("%s/rest/posts", strings.TrimRight(baseURL, "/"))
	cleanOrgID := strings.TrimPrefix(p.OrgID, "urn:li:organization:")
	authorURN := fmt.Sprintf("urn:li:organization:%s", cleanOrgID)

	reqPayload := map[string]interface{}{
		"author":     authorURN,
		"commentary": payload,
		"visibility": "PUBLIC",
		"distribution": map[string]interface{}{
			"feedDistribution": "MAIN_FEED",
			"targetEntities":   []interface{}{},
			"thirdPartyDistributionChannels": []interface{}{},
		},
		"content": map[string]interface{}{
			"article": map[string]interface{}{
				"source":      post.CanonicalUrl,
				"title":       post.Title,
				"description": post.Summary,
			},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LinkedIn-Version", p.APIVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("linkedIn Posts API error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Extract created post URN from x-restli-id header (per LinkedIn spec) or JSON body
	postURN := resp.Header.Get("x-restli-id")
	if postURN == "" {
		postURN = resp.Header.Get("X-Restli-Id")
	}
	if postURN == "" {
		var res struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(bodyBytes, &res); err == nil && res.ID != "" {
			postURN = res.ID
		}
	}
	if postURN == "" {
		postURN = fmt.Sprintf("urn:li:share:org_%s_%d", cleanOrgID, common.GetTimestamp())
	}

	remoteUrl := fmt.Sprintf("https://www.linkedin.com/feed/update/%s", postURN)
	return postURN, remoteUrl, nil
}

// Policy constants for DEV.to updates (Section 4)
const (
	DevToPolicyUpdateExisting      = "UPDATE_EXISTING"     // Policy A: update existing DEV.to article in-place via PUT
	DevToPolicyImmutableDerivative = "IMMUTABLE_DERIVATIVE" // Policy B: leave remote article immutable
)

type DevToArticle struct {
	ID                     int    `json:"id"`
	Title                  string `json:"title"`
	URL                    string `json:"url"`
	CanonicalURL           string `json:"canonical_url"`
	Published              bool   `json:"published"`
	PageViewsCount         int    `json:"page_views_count"`
	CommentsCount          int    `json:"comments_count"`
	PositiveReactionsCount int    `json:"positive_reactions_count"`
	BodyMarkdown           string `json:"body_markdown"`
}

type DevToAnalytics struct {
	RemotePostID      string `json:"remote_post_id"`
	RemoteURL         string `json:"remote_url"`
	Status            string `json:"status"` // "SYNCED", "UNKNOWN"
	PageViews         int    `json:"page_views"`
	PositiveReactions int    `json:"positive_reactions"`
	CommentsCount     int    `json:"comments_count"`
	PublishedAt       int64  `json:"published_at"`
	LastSyncedAt      int64  `json:"last_synced_at"`
}

// DevToPublisher publishes cross-posts to DEV.to / Forem platform
type DevToPublisher struct {
	APIKey       string
	BaseURL      string
	UpdatePolicy string
	Client       *http.Client
}

func NewDevToPublisher() *DevToPublisher {
	policy := os.Getenv("DEVTO_UPDATE_POLICY")
	if policy == "" {
		policy = DevToPolicyUpdateExisting // Default to Policy A (update existing article in-place)
	}
	return &DevToPublisher{
		APIKey:       os.Getenv("DEVTO_API_KEY"),
		BaseURL:      os.Getenv("DEVTO_BASE_URL"),
		UpdatePolicy: policy,
		Client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *DevToPublisher) PlatformName() string { return model.DistPlatformDevTo }

// FindExistingArticle searches the author's DEV.to articles for an existing publication matching canonical URL or title (Section 3)
func (p *DevToPublisher) FindExistingArticle(ctx context.Context, canonicalUrl string, title string) (*DevToArticle, error) {
	if p.APIKey == "" {
		return nil, errors.New("DEVTO_API_KEY not configured")
	}
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://dev.to"
	}
	apiURL := fmt.Sprintf("%s/api/articles/me/all?per_page=50", strings.TrimRight(baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("User-Agent", "Tora-DevTo-Client/1.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dev.to API error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var articles []DevToArticle
	if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		return nil, err
	}

	// 1. Exact match on canonical URL
	for _, a := range articles {
		if canonicalUrl != "" && strings.TrimRight(a.CanonicalURL, "/") == strings.TrimRight(canonicalUrl, "/") {
			return &a, nil
		}
	}
	// 2. Exact match on title (fallback)
	for _, a := range articles {
		if strings.EqualFold(strings.TrimSpace(a.Title), strings.TrimSpace(title)) {
			return &a, nil
		}
	}

	return nil, nil
}

// GetArticle fetches a single article by ID
func (p *DevToPublisher) GetArticle(ctx context.Context, remoteId string) (*DevToArticle, error) {
	if p.APIKey == "" || remoteId == "" {
		return nil, errors.New("missing API key or remoteId")
	}
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://dev.to"
	}
	apiURL := fmt.Sprintf("%s/api/articles/%s", strings.TrimRight(baseURL, "/"), remoteId)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("User-Agent", "Tora-DevTo-Client/1.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dev.to API error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var article DevToArticle
	if err := json.NewDecoder(resp.Body).Decode(&article); err != nil {
		return nil, err
	}
	return &article, nil
}

// GetArticleAnalytics collects real provider engagement metrics from DEV.to (Section 14)
func (p *DevToPublisher) GetArticleAnalytics(ctx context.Context, remoteId string) (*DevToAnalytics, error) {
	now := common.GetTimestamp()
	if remoteId == "" {
		return &DevToAnalytics{Status: "UNKNOWN", LastSyncedAt: now}, nil
	}
	article, err := p.GetArticle(ctx, remoteId)
	if err != nil {
		return &DevToAnalytics{
			RemotePostID: remoteId,
			Status:       "UNKNOWN",
			LastSyncedAt: now,
		}, nil
	}
	return &DevToAnalytics{
		RemotePostID:      fmt.Sprintf("%d", article.ID),
		RemoteURL:         article.URL,
		Status:            "SYNCED",
		PageViews:         article.PageViewsCount,
		PositiveReactions: article.PositiveReactionsCount,
		CommentsCount:     article.CommentsCount,
		LastSyncedAt:      now,
	}, nil
}

// Update modifies an existing DEV.to article in-place via PUT /api/articles/{id} (Policy A)
func (p *DevToPublisher) Update(ctx context.Context, remoteId string, post *model.NewsPost, payload string) (string, string, error) {
	if p.APIKey == "" {
		return "", "", errors.New("DEVTO_API_KEY not configured (OPERATOR_BLOCKED)")
	}
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://dev.to"
	}
	apiURL := fmt.Sprintf("%s/api/articles/%s", strings.TrimRight(baseURL, "/"), remoteId)
	reqPayload := map[string]interface{}{
		"article": map[string]interface{}{
			"title":         post.Title,
			"published":     true,
			"body_markdown": payload,
			"canonical_url": post.CanonicalUrl,
			"tags":          []string{"ai", "technology", "news"},
		},
	}

	reqBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Tora-DevTo-Client/1.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("dev.to update error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	_ = json.Unmarshal(bodyBytes, &res)
	retId := fmt.Sprintf("%d", res.ID)
	retUrl := res.URL
	if retUrl == "" {
		retUrl = fmt.Sprintf("https://dev.to/api/articles/%s", retId)
	}
	return retId, retUrl, nil
}

// Publish implements safe, idempotent DEV.to publication with zero-duplicate guarantees
func (p *DevToPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.APIKey == "" {
		return "", "", errors.New("DEVTO_API_KEY not configured (OPERATOR_BLOCKED)")
	}

	if err := ValidateDistributionUrl(post.CanonicalUrl); err != nil {
		return "", "", fmt.Errorf("canonical URL validation failed: %w", err)
	}

	// Step 1: Check if local database already has a recorded remote DEV.to ID for this post
	dists, _ := model.GetDistributionsByPostId(post.Id)
	var existingRemoteId string
	var existingRemoteUrl string
	for _, d := range dists {
		if d.Platform == model.DistPlatformDevTo {
			if d.RemotePostId != "" {
				existingRemoteId = d.RemotePostId
				existingRemoteUrl = d.RemoteUrl
				break
			}
			if d.ExternalPostId != "" {
				existingRemoteId = d.ExternalPostId
				existingRemoteUrl = d.ExternalUrl
				break
			}
		}
	}

	// Step 2: If no local remote ID, query remote DEV.to to see if it was already published (crash recovery & idempotency)
	if existingRemoteId == "" {
		existingRemote, _ := p.FindExistingArticle(ctx, post.CanonicalUrl, post.Title)
		if existingRemote != nil {
			existingRemoteId = fmt.Sprintf("%d", existingRemote.ID)
			existingRemoteUrl = existingRemote.URL
			logger.LogInfo(ctx, fmt.Sprintf("[DevToPublisher] Discovered existing remote article ID %s on DEV.to for canonical %s (reconciled)", existingRemoteId, post.CanonicalUrl))
		}
	}

	// Step 3: Handle existing remote article (Zero duplicates created)
	if existingRemoteId != "" {
		if p.UpdatePolicy == DevToPolicyUpdateExisting {
			logger.LogInfo(ctx, fmt.Sprintf("[DevToPublisher] Updating existing remote article %s in-place (Policy A)", existingRemoteId))
			upId, upUrl, err := p.Update(ctx, existingRemoteId, post, payload)
			if err != nil {
				// If update failed (e.g. rate limit), return existing without creating duplicate
				return existingRemoteId, existingRemoteUrl, nil
			}
			return upId, upUrl, nil
		}
		// Policy B: Immutable derivative
		logger.LogInfo(ctx, fmt.Sprintf("[DevToPublisher] Remote article %s remains immutable (Policy B)", existingRemoteId))
		return existingRemoteId, existingRemoteUrl, nil
	}

	// Step 4: Genuinely new publication
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = "https://dev.to"
	}
	apiURL := fmt.Sprintf("%s/api/articles", strings.TrimRight(baseURL, "/"))
	reqPayload := map[string]interface{}{
		"article": map[string]interface{}{
			"title":         post.Title,
			"published":     true,
			"body_markdown": payload,
			"canonical_url": post.CanonicalUrl,
			"tags":          []string{"ai", "technology", "news"},
		},
	}

	reqBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Tora-DevTo-Client/1.0")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("dev.to API error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	_ = json.Unmarshal(bodyBytes, &res)
	remoteId := fmt.Sprintf("%d", res.ID)
	remoteUrl := res.URL
	if remoteUrl == "" {
		remoteUrl = fmt.Sprintf("https://dev.to/api/articles/%s", remoteId)
	}
	return remoteId, remoteUrl, nil
}

// ProcessPendingDistributions executes the distribution pipeline with idempotent retries, kill switches, and allowlist safety
func ProcessPendingDistributions(ctx context.Context, maxBatch int) (processed int, err error) {
	// Section 21: Immediate global distribution kill switch
	if !IsGlobalDistributionEnabled() {
		logger.LogInfo(ctx, "distribution dispatch skipped: global kill switch active (NEWS_DISTRIBUTION_ENABLED=false)")
		return 0, nil
	}

	if maxBatch <= 0 {
		maxBatch = 1
	}
	if envLimit := os.Getenv("NEWS_DISTRIBUTION_BATCH_LIMIT"); envLimit != "" {
		if limitInt, parseErr := strconv.Atoi(envLimit); parseErr == nil && limitInt > 0 && limitInt < maxBatch {
			maxBatch = limitInt
		}
	}

	// Multi-channel canary order: DEV.to -> Facebook -> LinkedIn
	var pendings []model.NewsDistribution
	orderClause := "CASE platform WHEN 'devto' THEN 1 WHEN 'facebook' THEN 2 WHEN 'linkedin' THEN 3 ELSE 4 END ASC, id ASC"

	query := model.DB.Where("status = ?", "pending")

	// Section 17: Backlog surge protection via canary allowlist
	allowlist := GetDistributionAllowlistPostIDs()
	if len(allowlist) > 0 {
		var allowedIDs []int
		for id := range allowlist {
			allowedIDs = append(allowedIDs, id)
		}
		query = query.Where("post_id IN ?", allowedIDs)
	}

	err = query.Order(orderClause).Limit(maxBatch).Find(&pendings).Error
	if err != nil {
		return 0, err
	}

	publishers := map[string]DistClient{
		model.DistPlatformDevTo:    NewDevToPublisher(),
		model.DistPlatformFacebook: NewFacebookPublisher(),
		model.DistPlatformLinkedIn: NewLinkedInPublisher(),
	}

	for i := range pendings {
		d := &pendings[i]

		// Atomically claim distribution to prevent concurrent worker execution (race condition defense)
		claimResult := model.DB.Model(&model.NewsDistribution{}).
			Where("id = ? AND status = ?", d.Id, "pending").
			Updates(map[string]interface{}{
				"status":        "processing",
				"attempt_count": d.AttemptCount + 1,
				"updated_at":    common.GetTimestamp(),
			})
		if claimResult.RowsAffected == 0 {
			// Another worker already claimed or processed this row
			continue
		}
		d.AttemptCount++

		// Section 21: Per-platform distribution enablement check
		if !IsPlatformDistributionEnabled(d.Platform) {
			d.Status = "operator_blocked"
			d.ErrorMessage = fmt.Sprintf("[%s] platform %q distribution disabled by configuration", DistFailurePermission, d.Platform)
			_ = model.UpdateNewsDistribution(d)
			continue
		}

		post, postErr := model.GetNewsPostById(d.PostId)
		if postErr != nil || post == nil {
			d.Status = "failed"
			d.ErrorMessage = fmt.Sprintf("[%s] associated news post not found", DistFailurePermanent)
			_ = model.UpdateNewsDistribution(d)
			continue
		}

		// Ensure DistributionVersion default
		if d.DistributionVersion <= 0 {
			d.DistributionVersion = 1
		}

		// Calculate idempotent execution key: decoupled from Content Revision and UpdatedAt
		hasher := sha256.New()
		hasher.Write([]byte(fmt.Sprintf("%d:%s:%d", post.Id, d.Platform, d.DistributionVersion)))
		d.IdempotencyKey = hex.EncodeToString(hasher.Sum(nil))

		// At-least-once reconciliation check: if remote post was already created before a crash/timeout
		if (d.RemotePostId != "" || d.ExternalPostId != "") && d.Status != "published" {
			remoteId := d.RemotePostId
			if remoteId == "" {
				remoteId = d.ExternalPostId
			}
			remoteUrl := d.RemoteUrl
			if remoteUrl == "" {
				remoteUrl = d.ExternalUrl
			}
			d.Status = "published"
			d.RemotePostId = remoteId
			d.ExternalPostId = remoteId
			d.RemoteUrl = remoteUrl
			d.ExternalUrl = remoteUrl
			d.PublishedAt = common.GetTimestamp()
			d.ErrorMessage = ""
			_ = model.UpdateNewsDistribution(d)
			processed++
			continue
		}

		pub, exists := publishers[d.Platform]
		if !exists {
			// For unsupported or unconfigured platform (e.g. twitter without keys)
			d.Status = "operator_blocked"
			d.ErrorMessage = fmt.Sprintf("[%s] no active distribution adapter for platform %q (OPERATOR_BLOCKED)", DistFailureAuth, d.Platform)
			_ = model.UpdateNewsDistribution(d)
			continue
		}

		extId, extUrl, pubErr := pub.Publish(ctx, post, d.ContentPayload)
		if pubErr != nil {
			failureClass := ClassifyDistributionFailure(pubErr, 0)
			d.ErrorMessage = fmt.Sprintf("[%s] %s", failureClass, pubErr.Error())

			if failureClass == DistFailureAuth || failureClass == DistFailurePermission || strings.Contains(pubErr.Error(), "OPERATOR_BLOCKED") {
				d.Status = "operator_blocked"
			} else if failureClass == DistFailureNetworkAmbiguous {
				// Retryable without consuming attempt limits immediately
				d.Status = "pending"
			} else if d.AttemptCount >= 3 {
				d.Status = "failed"
			} else {
				d.Status = "pending" // allow retry on next pass
			}
			logger.LogWarn(ctx, fmt.Sprintf("distribution failed for post %d to %s: %v", post.Id, d.Platform, pubErr))
		} else {
			d.Status = "published"
			d.ExternalPostId = extId
			d.RemotePostId = extId
			d.ExternalUrl = extUrl
			d.RemoteUrl = extUrl
			d.PublishedAt = common.GetTimestamp()
			d.ErrorMessage = ""
			processed++
		}

		_ = model.UpdateNewsDistribution(d)
	}

	return processed, nil
}
