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
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

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

func (p *FacebookPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.PageToken == "" || p.PageID == "" {
		return "", "", errors.New("FACEBOOK_PAGE_ACCESS_TOKEN or FACEBOOK_PAGE_ID not configured (OPERATOR_BLOCKED)")
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

func (p *LinkedInPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.AccessToken == "" || p.OrgID == "" {
		return "", "", errors.New("LINKEDIN_ACCESS_TOKEN or LINKEDIN_ORG_ID not configured (OPERATOR_BLOCKED)")
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

// DevToPublisher publishes cross-posts to DEV.to / Forem platform
type DevToPublisher struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewDevToPublisher() *DevToPublisher {
	return &DevToPublisher{
		APIKey:  os.Getenv("DEVTO_API_KEY"),
		BaseURL: os.Getenv("DEVTO_BASE_URL"),
		Client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *DevToPublisher) PlatformName() string { return model.DistPlatformDevTo }

func (p *DevToPublisher) Publish(ctx context.Context, post *model.NewsPost, payload string) (string, string, error) {
	if p.APIKey == "" {
		return "", "", errors.New("DEVTO_API_KEY not configured (OPERATOR_BLOCKED)")
	}

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

// ProcessPendingDistributions executes the distribution pipeline with idempotent retries and multi-channel canary order
func ProcessPendingDistributions(ctx context.Context, maxBatch int) (processed int, err error) {
	// Multi-channel canary order: DEV.to -> Facebook -> LinkedIn
	var pendings []model.NewsDistribution
	orderClause := "CASE platform WHEN 'devto' THEN 1 WHEN 'facebook' THEN 2 WHEN 'linkedin' THEN 3 ELSE 4 END ASC, id ASC"
	err = model.DB.Where("status = ?", "pending").Order(orderClause).Limit(maxBatch).Find(&pendings).Error
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

		post, postErr := model.GetNewsPostById(d.PostId)
		if postErr != nil || post == nil {
			d.Status = "failed"
			d.ErrorMessage = "associated news post not found"
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
			d.ErrorMessage = fmt.Sprintf("no active distribution adapter for platform %q (OPERATOR_BLOCKED)", d.Platform)
			_ = model.UpdateNewsDistribution(d)
			continue
		}

		extId, extUrl, pubErr := pub.Publish(ctx, post, d.ContentPayload)
		if pubErr != nil {
			if strings.Contains(pubErr.Error(), "OPERATOR_BLOCKED") {
				d.Status = "operator_blocked"
				d.ErrorMessage = pubErr.Error()
			} else {
				if d.AttemptCount >= 3 {
					d.Status = "failed"
				} else {
					d.Status = "pending" // allow retry on next pass
				}
				d.ErrorMessage = pubErr.Error()
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
