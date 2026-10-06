package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewsCanary_EndToEndGrowthLoop(t *testing.T) {
	setupServiceNewsTestDB(t)

	// 1. Mock upstream authoritative release feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>OpenAI Canary Feed</title>
    <link>https://openai.com</link>
    <item>
      <title>OpenAI Announces O3-Mini High Reasoning Model for Real-Time Coding</title>
      <link>https://openai.com/index/o3-mini-reasoning-canary</link>
      <description>OpenAI launches o3-mini delivering STEM reasoning, coding accuracy and sub-second token streaming for enterprise developers.</description>
      <pubDate>` + time.Now().Format(time.RFC1123) + `</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	src := &model.NewsSource{
		Name:                   "OpenAI Canary Feed",
		Slug:                   "openai-canary-feed",
		FeedUrl:                server.URL,
		SiteUrl:                "https://openai.com",
		SourceType:             "rss",
		TrustTier:              "tier_1_official",
		Enabled:                true,
		PollingIntervalMinutes: 15,
	}
	require.NoError(t, model.CreateNewsSource(src))

	// 2. DISCOVER & SCOUT
	newPosts, err := SyncSingleNewsSource(context.Background(), src)
	require.NoError(t, err)
	assert.Equal(t, 1, newPosts, "Canary feed should draft 1 new high-relevance post")

	// 3. VERIFY EDITORIAL & METADATA
	posts, total, err := model.GetPublishedNewsPosts(1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	canaryPost := posts[0]

	assert.Equal(t, model.NewsStatusPublished, canaryPost.Status)
	assert.Equal(t, model.FactCheckVerified, canaryPost.FactCheckStatus)
	assert.Equal(t, model.ContentTypeNews, canaryPost.ContentType)
	assert.Contains(t, canaryPost.CanonicalUrl, canaryPost.Slug)
	assert.Contains(t, canaryPost.OgImageUrl, "/og.png")
	assert.NotEmpty(t, canaryPost.ContentHTML)
	assert.NotEmpty(t, canaryPost.Summary)

	// 4. VERIFY SCHEMA.ORG JSON-LD
	jsonLD := GenerateNewsArticleJSONLD(canaryPost)
	assert.Contains(t, jsonLD, `"@type":"NewsArticle"`)
	assert.Contains(t, jsonLD, canaryPost.CanonicalUrl)
	assert.Contains(t, jsonLD, "Tora AI")

	// 5. VERIFY 1200x630 RASTER PNG CARD
	pngBytes, err := GenerateOGCardPNG(canaryPost)
	require.NoError(t, err)
	require.NotEmpty(t, pngBytes)
	assert.Equal(t, byte(0x89), pngBytes[0])
	assert.Equal(t, byte('P'), pngBytes[1])
	assert.Equal(t, byte('N'), pngBytes[2])
	assert.Equal(t, byte('G'), pngBytes[3])

	// 6. VERIFY GOOGLE NEWS SITEMAP INCLUSION
	sitemapXML := GenerateNewsSitemapXMLWithTime([]*model.NewsPost{canaryPost}, time.Now())
	assert.Contains(t, sitemapXML, canaryPost.CanonicalUrl, "Fresh canary news must be in Google News sitemap")

	// 7. VERIFY RECURRING GROWTH REVIEW & REPORTING
	review, err := GenerateDailyGrowthReview(time.Now().Format("2006-01-02"))
	require.NoError(t, err)
	assert.Equal(t, 1, review.NewsCount)
	assert.Equal(t, 1, review.PostsPublishedToday)
}

func TestLinkedInPublisher_PostsAPIContract(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedVersion string
	var capturedRestli string
	var capturedAuth string
	var capturedContentType string
	var capturedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedVersion = r.Header.Get("LinkedIn-Version")
		capturedRestli = r.Header.Get("X-Restli-Protocol-Version")
		capturedAuth = r.Header.Get("Authorization")
		capturedContentType = r.Header.Get("Content-Type")

		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &capturedPayload)

		// LinkedIn Posts API returns 201 Created with x-restli-id header
		w.Header().Set("x-restli-id", "urn:li:share:7123456789012345678")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"urn:li:share:7123456789012345678"}`))
	}))
	defer server.Close()

	pub := &LinkedInPublisher{
		AccessToken: "test_token_live_123",
		OrgID:       "1234567",
		APIVersion:  "202401",
		BaseURL:     server.URL,
		Client:      server.Client(),
	}

	post := &model.NewsPost{
		Id:           42,
		Title:        "Tora AI Engineering Update",
		Summary:      "Overview of autonomous routing architecture",
		CanonicalUrl: common.GetCanonicalBaseURL() + "/news/tora-engineering-update",
	}

	remoteId, remoteUrl, err := pub.Publish(context.Background(), post, "Technical breakdown for software engineers")
	require.NoError(t, err)

	// Contract verifications
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/rest/posts", capturedPath)
	assert.Equal(t, "202401", capturedVersion)
	assert.Equal(t, "2.0.0", capturedRestli)
	assert.Equal(t, "Bearer test_token_live_123", capturedAuth)
	assert.Equal(t, "application/json", capturedContentType)

	// Body schema assertions
	assert.Equal(t, "urn:li:organization:1234567", capturedPayload["author"])
	assert.Equal(t, "Technical breakdown for software engineers", capturedPayload["commentary"])
	assert.Equal(t, "PUBLIC", capturedPayload["visibility"])
	assert.Equal(t, "PUBLISHED", capturedPayload["lifecycleState"])

	distMap, ok := capturedPayload["distribution"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "MAIN_FEED", distMap["feedDistribution"])

	contentMap, ok := capturedPayload["content"].(map[string]interface{})
	require.True(t, ok)
	articleMap, ok := contentMap["article"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, post.CanonicalUrl, articleMap["source"])
	assert.Equal(t, post.Title, articleMap["title"])
	assert.Equal(t, post.Summary, articleMap["description"])

	// Response parsing assertions
	assert.Equal(t, "urn:li:share:7123456789012345678", remoteId)
	assert.Equal(t, "https://www.linkedin.com/feed/update/urn:li:share:7123456789012345678", remoteUrl)
}

func TestDistributionIdempotency_ContentEditDoesNotDuplicate(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Slug:            "idempotency-edit-test",
		Title:           "Original Title Before Routine Edit",
		Summary:         "Original summary text",
		ContentMarkdown: "# Original Markdown Content",
		Status:          model.NewsStatusPublished,
		PublishedAt:     common.GetTimestamp(),
	}
	require.NoError(t, model.CreateNewsPost(post))
	require.NoError(t, QueuePostDistributions(post))

	initialDists, err := model.GetDistributionsByPostId(post.Id)
	require.NoError(t, err)
	require.Len(t, initialDists, 4, "Should have 4 platform distribution rows")

	initialKeys := make(map[string]string)
	for _, d := range initialDists {
		initialKeys[d.Platform] = d.IdempotencyKey
	}

	// 1. Simulate routine editorial edits and SEO remediation updates
	for i := 1; i <= 3; i++ {
		post.Title = fmt.Sprintf("Updated Title Revision %d", i)
		post.Summary = fmt.Sprintf("Updated Summary Revision %d", i)
		post.ContentMarkdown = fmt.Sprintf("# Revision %d\n\nFresh content details...", i)
		post.UpdatedAt = common.GetTimestamp() + int64(i*300)
		require.NoError(t, model.UpdateNewsPost(post))

		// Attempt re-queueing derivatives
		require.NoError(t, QueuePostDistributions(post))

		// Distributions must NOT be duplicated!
		currentDists, err := model.GetDistributionsByPostId(post.Id)
		require.NoError(t, err)
		assert.Len(t, currentDists, 4, "Routine edits must NEVER duplicate distribution rows")

		for _, d := range currentDists {
			assert.Equal(t, initialKeys[d.Platform], d.IdempotencyKey,
				"Idempotency key must remain strictly decoupled from content/timestamp revisions")
		}
	}
}

func TestDistributionIdempotency_AtLeastOnceReconciliation(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Slug:            "reconcile-test-post",
		Title:           "Worker Recovery Reconcile Post",
		Summary:         "Summary for recovery testing",
		ContentMarkdown: "## Content",
		Status:          model.NewsStatusPublished,
		PublishedAt:     common.GetTimestamp(),
	}
	require.NoError(t, model.CreateNewsPost(post))

	// Simulate distribution record where worker got remote ID but crashed before setting status=published
	d := &model.NewsDistribution{
		PostId:              post.Id,
		Platform:            model.DistPlatformDevTo,
		Status:              "pending",
		ContentPayload:      "test payload",
		RemotePostId:        "devto_remote_7788",
		RemoteUrl:           "https://dev.to/tora/reconcile-test",
		DistributionVersion: 1,
	}
	require.NoError(t, model.CreateNewsDistribution(d))

	// Process without DEVTO_API_KEY set: reconciliation MUST succeed without needing remote call
	processed, err := ProcessPendingDistributions(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed, "Reconciliation must succeed without remote network calls")

	reconciled, err := model.GetDistributionByPostPlatformAndVersion(post.Id, model.DistPlatformDevTo, 1)
	require.NoError(t, err)
	require.NotNil(t, reconciled)
	assert.Equal(t, "published", reconciled.Status)
	assert.Equal(t, "devto_remote_7788", reconciled.RemotePostId)
	assert.Equal(t, "devto_remote_7788", reconciled.ExternalPostId)
	assert.Equal(t, "https://dev.to/tora/reconcile-test", reconciled.RemoteUrl)
	assert.NotEmpty(t, reconciled.PublishedAt)
	assert.Empty(t, reconciled.ErrorMessage)
}

func TestDistributionIdempotency_WorkerRaceCondition(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Slug:            "race-condition-test",
		Title:           "Race Condition Test Post",
		Summary:         "Summary for testing concurrency",
		ContentMarkdown: "## Content",
		Status:          model.NewsStatusPublished,
		PublishedAt:     common.GetTimestamp(),
	}
	require.NoError(t, model.CreateNewsPost(post))

	d := &model.NewsDistribution{
		PostId:              post.Id,
		Platform:            model.DistPlatformDevTo,
		Status:              "pending",
		ContentPayload:      "test payload",
		RemotePostId:        "precreated_ext_55",
		RemoteUrl:           "https://dev.to/tora/race-test",
		DistributionVersion: 1,
	}
	require.NoError(t, model.CreateNewsDistribution(d))

	// Launch two concurrent worker passes attempting to claim the same pending row
	var wg sync.WaitGroup
	results := make([]int, 2)
	errs := make([]error, 2)

	for workerID := 0; workerID < 2; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results[id], errs[id] = ProcessPendingDistributions(context.Background(), 10)
		}(workerID)
	}
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	// Total processed between both workers must be exactly 1
	totalProcessed := results[0] + results[1]
	assert.Equal(t, 1, totalProcessed, "Exactly one worker must claim and process the distribution row")
}

func TestNewsSeo_SeedExclusionFromGoogleNewsSitemap(t *testing.T) {
	refTime := time.Now()
	refUnix := refTime.Unix()

	seedPost := &model.NewsPost{
		Id:           101,
		Slug:         "seed-article-fresh",
		ContentType:  model.ContentTypeNews,
		Title:        "Seed Launchpack Article",
		Summary:      "Seed description",
		CanonicalUrl: common.GetCanonicalBaseURL() + "/news/seed-article-fresh",
		Status:       model.NewsStatusPublished,
		IsSeed:       true,
		PublishedAt:  refUnix - 1800, // 30 minutes ago (within 48 hours)
		UpdatedAt:    refUnix - 1800,
	}

	breakingNews := &model.NewsPost{
		Id:           102,
		Slug:         "breaking-ai-event",
		ContentType:  model.ContentTypeNews,
		Title:        "Breaking Technical News Announcement",
		Summary:      "Breaking news description",
		CanonicalUrl: common.GetCanonicalBaseURL() + "/news/breaking-ai-event",
		Status:       model.NewsStatusPublished,
		IsSeed:       false,
		PublishedAt:  refUnix - 1800, // 30 minutes ago
		UpdatedAt:    refUnix - 1800,
	}

	posts := []*model.NewsPost{seedPost, breakingNews}

	// 1. Google News Sitemap: Seed articles MUST be excluded
	newsSitemap := GenerateNewsSitemapXMLWithTime(posts, refTime)
	assert.Contains(t, newsSitemap, breakingNews.CanonicalUrl, "Fresh breaking news MUST be present in Google News sitemap")
	assert.NotContains(t, newsSitemap, seedPost.CanonicalUrl, "Seed post MUST NOT be present in Google News sitemap")

	// 2. Standard Sitemap: Both seed and live news MUST be present for evergreen SEO discovery
	standardSitemap := GenerateSitemapXML(posts)
	assert.Contains(t, standardSitemap, breakingNews.CanonicalUrl, "Breaking news MUST be present in standard sitemap")
	assert.Contains(t, standardSitemap, seedPost.CanonicalUrl, "Seed evergreen article MUST be present in standard sitemap")
}

func TestNewsAttribution_KeyedHMACAndIPPrivacy(t *testing.T) {
	origKey := os.Getenv("ATTRIBUTION_HASH_KEY")
	defer func() { _ = os.Setenv("ATTRIBUTION_HASH_KEY", origKey) }()

	ip := "203.0.113.195"
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"

	_ = os.Setenv("ATTRIBUTION_HASH_KEY", "secret_key_alpha_2026")
	hashAlpha := HashClientIdentity(ip, ua)

	_ = os.Setenv("ATTRIBUTION_HASH_KEY", "secret_key_beta_2026")
	hashBeta := HashClientIdentity(ip, ua)

	assert.NotEmpty(t, hashAlpha)
	assert.NotEmpty(t, hashBeta)
	assert.NotEqual(t, hashAlpha, hashBeta, "Keyed HMAC must produce distinct outputs for different secret keys")
	assert.NotContains(t, hashAlpha, ip, "Hash must never leak raw IP address")
	assert.NotContains(t, hashBeta, ip, "Hash must never leak raw IP address")
	assert.Len(t, hashAlpha, 32, "Pseudonym length should be 32 chars")
}

func TestNewsGSC_CredentialsFileSupport(t *testing.T) {
	origFile := os.Getenv("GSC_CREDENTIALS_FILE")
	origJSON := os.Getenv("GSC_CREDENTIALS_JSON")
	defer func() {
		_ = os.Setenv("GSC_CREDENTIALS_FILE", origFile)
		_ = os.Setenv("GSC_CREDENTIALS_JSON", origJSON)
	}()

	_ = os.Unsetenv("GSC_CREDENTIALS_JSON")

	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "gsc_service_account.json")
	fakeJSON := `{"type":"service_account","project_id":"tora-growth-project","client_email":"sa@tora.iam.gserviceaccount.com"}`
	require.NoError(t, os.WriteFile(credsPath, []byte(fakeJSON), 0600))

	_ = os.Setenv("GSC_CREDENTIALS_FILE", credsPath)

	client := NewDefaultGSCClient()
	assert.Equal(t, credsPath, client.CredentialsFile)
	assert.Equal(t, fakeJSON, client.CredentialsJSON)
	assert.Equal(t, "https://www.googleapis.com/auth/webmasters.readonly", client.ReadOnlyScope)

	// Live query check without operator authorization returns OPERATOR_BLOCKED safely
	_, err := client.QuerySearchAnalytics(context.Background(), common.GetGSCSiteURL(), "2026-10-01", "2026-10-05", []string{"query"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPERATOR_BLOCKED")
	assert.NotContains(t, err.Error(), fakeJSON, "Credentials must never be leaked in error message or logs")
}

func TestExternalAPIConfig_VersionsAndSafety(t *testing.T) {
	// 1. Defaults verification (2026 active versions)
	assert.Equal(t, "v26.0", GetFacebookGraphAPIVersion(), "Facebook Graph API default must be active v26.0")
	assert.Equal(t, "202609", GetLinkedInAPIVersion(), "LinkedIn Posts API default must be active 202609")

	// 2. Safe overrides
	_ = os.Setenv("FACEBOOK_GRAPH_API_VERSION", "v25.0")
	_ = os.Setenv("LINKEDIN_API_VERSION", "202606")
	assert.Equal(t, "v25.0", GetFacebookGraphAPIVersion())
	assert.Equal(t, "202606", GetLinkedInAPIVersion())

	// 3. Invalid version formatting fallback
	_ = os.Setenv("FACEBOOK_GRAPH_API_VERSION", "invalid_ver")
	_ = os.Setenv("LINKEDIN_API_VERSION", "bad")
	assert.Equal(t, "v26.0", GetFacebookGraphAPIVersion(), "Must fallback to default on malformed version")
	assert.Equal(t, "202609", GetLinkedInAPIVersion(), "Must fallback to default on malformed version")

	_ = os.Unsetenv("FACEBOOK_GRAPH_API_VERSION")
	_ = os.Unsetenv("LINKEDIN_API_VERSION")

	// 4. Safe logging verification (zero tokens)
	LogExternalAPIConfigSummary(context.Background())
}

func TestCanonicalHostDecoupling_DefaultsAndOverrides(t *testing.T) {
	// 1. Defaults
	assert.Equal(t, "https://www.toraapi.com", common.GetCanonicalBaseURL())
	assert.Equal(t, "https://www.toraapi.com", common.GetApiBaseURL())
	assert.Equal(t, "sc-domain:toraapi.com", common.GetGSCSiteURL())

	// 2. Overrides
	_ = os.Setenv("CANONICAL_BASE_URL", "https://custom.toraapi.com/")
	_ = os.Setenv("API_BASE_URL", "https://api-custom.toraapi.com/")
	_ = os.Setenv("GSC_SITE_URL", "https://custom.toraapi.com/")

	assert.Equal(t, "https://custom.toraapi.com", common.GetCanonicalBaseURL(), "Trailing slashes must be stripped")
	assert.Equal(t, "https://api-custom.toraapi.com", common.GetApiBaseURL())
	assert.Equal(t, "https://custom.toraapi.com/", common.GetGSCSiteURL())

	_ = os.Unsetenv("CANONICAL_BASE_URL")
	_ = os.Unsetenv("API_BASE_URL")
	_ = os.Unsetenv("GSC_SITE_URL")
}

func TestBrowserSecurity_MarkdownXSSSanitization(t *testing.T) {
	maliciousMarkdown := `# Title
<script>alert("xss")</script>
<img src="x" onerror="stealCookies()">
[Click Me](javascript:stealTokens())
[Safe Link](https://www.toraapi.com/dashboard)
**Bold with <svg onload=exploit()>**
`
	renderedHTML := RenderMarkdownToSafeHTML(maliciousMarkdown)

	// Ensure no active HTML tags can execute (all malicious tags escaped to &lt;tag&gt;)
	assert.NotContains(t, renderedHTML, "<script", "Raw <script> tags must never render unescaped")
	assert.NotContains(t, renderedHTML, "<img", "Raw <img> tags must never render unescaped")
	assert.NotContains(t, renderedHTML, "<svg", "Raw <svg> tags must never render unescaped")
	assert.NotContains(t, renderedHTML, `href="javascript:`, "javascript: URLs must be blocked")

	assert.Contains(t, renderedHTML, "&lt;script&gt;", "Malicious script must be neutralized as text entity")
	assert.Contains(t, renderedHTML, "&lt;img", "Malicious img must be neutralized as text entity")

	// Ensure safe links and tags DO render
	assert.Contains(t, renderedHTML, `href="https://www.toraapi.com/dashboard"`, "Safe HTTPS link must render")
	assert.Contains(t, renderedHTML, "Safe Link")
	assert.Contains(t, renderedHTML, "strong", "Markdown bold must render as strong tag")
}

func TestNewsDistribution_SafetyAndKillSwitch(t *testing.T) {
	setupServiceNewsTestDB(t)

	// 1. URL Safety Invariants (Section 2)
	assert.NoError(t, ValidateDistributionUrl("https://www.toraapi.com/news/sample-slug"))
	assert.Error(t, ValidateDistributionUrl("http://localhost:3000/news/sample-slug"), "Localhost must be rejected")
	assert.Error(t, ValidateDistributionUrl("http://127.0.0.1:3000/news/sample-slug"), "127.0.0.1 must be rejected")
	assert.Error(t, ValidateDistributionUrl("https://staging-api.toraapi.com/news/sample-slug"), "Staging host must be rejected")
	assert.Error(t, ValidateDistributionUrl("https://tora.ai/news/sample-slug"), "Deprecated tora.ai must be rejected")
	assert.Error(t, ValidateDistributionUrl("https://api.tora.ai/news/sample-slug"), "Deprecated api.tora.ai must be rejected")
	assert.Error(t, ValidateDistributionUrl("https://www.toraapi.com/docs"), "Non-news path must be rejected")

	// 2. Global Kill Switch (Section 21)
	origKill := os.Getenv("NEWS_DISTRIBUTION_ENABLED")
	defer func() { _ = os.Setenv("NEWS_DISTRIBUTION_ENABLED", origKill) }()

	_ = os.Setenv("NEWS_DISTRIBUTION_ENABLED", "false")
	assert.False(t, IsGlobalDistributionEnabled())
	processed, err := ProcessPendingDistributions(context.Background(), 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, processed, "Zero distributions must process when kill switch is active")

	_ = os.Setenv("NEWS_DISTRIBUTION_ENABLED", "true")
	assert.True(t, IsGlobalDistributionEnabled())

	// 3. Platform Enablement Switches
	_ = os.Setenv("DEVTO_DISTRIBUTION_ENABLED", "false")
	assert.False(t, IsPlatformDistributionEnabled(model.DistPlatformDevTo))
	assert.True(t, IsPlatformDistributionEnabled(model.DistPlatformFacebook))
	_ = os.Unsetenv("DEVTO_DISTRIBUTION_ENABLED")
}

func TestNewsDistribution_FailureClassification(t *testing.T) {
	// Section 16: Failure classification
	assert.Equal(t, DistFailureAuth, ClassifyDistributionFailure(errors.New("OPERATOR_BLOCKED: token missing"), 0))
	assert.Equal(t, DistFailureAuth, ClassifyDistributionFailure(errors.New("unauthorized"), 401))
	assert.Equal(t, DistFailurePermission, ClassifyDistributionFailure(errors.New("forbidden scope"), 403))
	assert.Equal(t, DistFailureRateLimit, ClassifyDistributionFailure(errors.New("quota exceeded"), 429))
	assert.Equal(t, DistFailureValidation, ClassifyDistributionFailure(errors.New("bad request parameters"), 400))
	assert.Equal(t, DistFailureRemote5xx, ClassifyDistributionFailure(errors.New("internal server error"), 500))
	assert.Equal(t, DistFailureNetworkAmbiguous, ClassifyDistributionFailure(errors.New("context deadline exceeded (timeout)"), 0))
	assert.Equal(t, DistFailurePermanent, ClassifyDistributionFailure(errors.New("unknown permanent error"), 0))
}

func TestNewsDistribution_AllowlistBacklogProtection(t *testing.T) {
	setupServiceNewsTestDB(t)

	origAllow := os.Getenv("NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS")
	origKill := os.Getenv("NEWS_DISTRIBUTION_ENABLED")
	defer func() {
		_ = os.Setenv("NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS", origAllow)
		_ = os.Setenv("NEWS_DISTRIBUTION_ENABLED", origKill)
	}()

	_ = os.Setenv("NEWS_DISTRIBUTION_ENABLED", "true")

	// Create 2 posts: post 1 (backlog) and post 2 (canary target)
	post1 := &model.NewsPost{Id: 101, Slug: "backlog-1", Title: "Backlog 1", Status: model.NewsStatusPublished}
	post2 := &model.NewsPost{Id: 102, Slug: "canary-2", Title: "Canary 2", Status: model.NewsStatusPublished}
	require.NoError(t, model.CreateNewsPost(post1))
	require.NoError(t, model.CreateNewsPost(post2))

	// Queue pending distributions for both
	require.NoError(t, model.CreateNewsDistribution(&model.NewsDistribution{PostId: 101, Platform: model.DistPlatformDevTo, Status: "pending", ContentPayload: "backlog"}))
	require.NoError(t, model.CreateNewsDistribution(&model.NewsDistribution{PostId: 102, Platform: model.DistPlatformDevTo, Status: "pending", ContentPayload: "canary", RemotePostId: "reconciled_99"}))

	// Allowlist ONLY post 102
	_ = os.Setenv("NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS", "102")

	// Process: ONLY post 102 should be processed (reconciled), post 101 MUST remain untouched in pending!
	processed, err := ProcessPendingDistributions(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	d101, err := model.GetDistributionsByPostId(101)
	require.NoError(t, err)
	require.Len(t, d101, 1)
	assert.Equal(t, "pending", d101[0].Status, "Backlog post not in allowlist must remain pending without dispatch surge")

	d102, err := model.GetDistributionsByPostId(102)
	require.NoError(t, err)
	require.Len(t, d102, 1)
	assert.Equal(t, "published", d102[0].Status, "Allowlisted post should process safely")
}

func TestNewsGrowth_PostRecordAndOverview(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Id:           201,
		Slug:         "growth-record-post",
		Title:        "Growth Record Post",
		ContentType:  model.ContentTypeNews,
		Status:       model.NewsStatusPublished,
		CanonicalUrl: common.GetCanonicalBaseURL() + "/news/growth-record-post",
		PublishedAt:  common.GetTimestamp(),
	}
	require.NoError(t, model.CreateNewsPost(post))

	// Record an attribution event
	require.NoError(t, RecordConversion(ConversionSignup, "201", "devto", "referral", "launch", "1.2.3.4", "agent", 1, 0, ""))

	// Fetch growth record
	record, err := GetPostGrowthRecord(201)
	require.NoError(t, err)
	assert.Equal(t, 201, record.PostId)
	assert.Equal(t, "growth-record-post", record.Slug)
	assert.Equal(t, 1, record.SignupsCount)

	// Fetch global overview
	overview, err := GetGlobalGrowthOverview()
	require.NoError(t, err)
	assert.NotNil(t, overview)
	assert.Contains(t, overview.ChannelStatuses, "gsc")
	assert.Contains(t, overview.ChannelStatuses, "devto")
	assert.Contains(t, overview.ChannelStatuses, "facebook")
	assert.True(t, overview.ChannelStatuses["gsc"] == GSCStatusNotConfigured || overview.ChannelStatuses["gsc"] == "OPERATOR_BLOCKED")
}



