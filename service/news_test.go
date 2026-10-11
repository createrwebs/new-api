package service

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupServiceNewsTestDB(t *testing.T) {
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.NewsSource{},
		&model.StoryCluster{},
		&model.NewsPost{},
		&model.NewsDistribution{},
		&model.NewsAnalyticEvent{},
		&model.NewsSeoMetric{},
		&model.NewsSeoOpportunity{},
		&model.NewsConversionEvent{},
		&model.NewsDailyGrowthReview{},
		&model.NewsUrlInspection{},
		&model.NewsSeoExperiment{},
		&model.NewsFeedItem{},
		&model.NewsAiVisibilityObservation{},
		&model.NewsPublicationEvent{},
		&model.NewsAutopilotDailyQuota{},
	))
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_news_pub_events_unique_initial ON news_publication_events (post_id, event_type);")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
		model.InvalidateNewsCache()
	})
}

func TestNewsScout_ParseFeedXML_RSS(t *testing.T) {
	rssData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>OpenAI Official Blog</title>
    <link>https://openai.com/news</link>
    <description>Latest news from OpenAI</description>
    <item>
      <title>Introducing GPT-4.5 for Advanced Reasoning and Chat</title>
      <link>https://openai.com/index/introducing-gpt-4-5/</link>
      <description><![CDATA[Today we are announcing GPT-4.5, our largest and most capable model yet for reasoning, coding, and general knowledge.]]></description>
      <pubDate>Mon, 05 Oct 2026 12:00:00 GMT</pubDate>
      <guid>https://openai.com/index/introducing-gpt-4-5/</guid>
    </item>
  </channel>
</rss>`)

	src := &model.NewsSource{
		Id:        1,
		Name:      "OpenAI",
		TrustTier: "tier_1_official",
	}

	items, err := ParseFeedXML(rssData, src)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, "Introducing GPT-4.5 for Advanced Reasoning and Chat", item.Title)
	assert.Equal(t, "https://openai.com/index/introducing-gpt-4-5/", item.URL)
	assert.Contains(t, item.Summary, "GPT-4.5")
	assert.Equal(t, "OpenAI", item.SourceName)
	assert.Equal(t, 1, item.SourceId)
}

func TestNewsScout_ParseFeedXML_Atom(t *testing.T) {
	atomData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Anthropic Research</title>
  <link href="https://www.anthropic.com/news"/>
  <updated>2026-10-05T12:00:00Z</updated>
  <entry>
    <title>Claude 3.7 Sonnet: Hybrid Reasoning Across Fast and Thoughtful Workflows</title>
    <link href="https://www.anthropic.com/news/claude-3-7-sonnet" rel="alternate"/>
    <id>tag:anthropic.com,2026:news/claude-3-7-sonnet</id>
    <published>2026-10-05T12:00:00Z</published>
    <summary>We introduce Claude 3.7 Sonnet, delivering state of the art hybrid reasoning capabilities for software engineering.</summary>
    <author>
      <name>Anthropic Team</name>
    </author>
  </entry>
</feed>`)

	src := &model.NewsSource{
		Id:        2,
		Name:      "Anthropic",
		TrustTier: "tier_1_official",
	}

	items, err := ParseFeedXML(atomData, src)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, "Claude 3.7 Sonnet: Hybrid Reasoning Across Fast and Thoughtful Workflows", item.Title)
	assert.Equal(t, "https://www.anthropic.com/news/claude-3-7-sonnet", item.URL)
	assert.Equal(t, "Anthropic Team", item.Author)
	assert.Contains(t, item.Summary, "Claude 3.7 Sonnet")
}

func TestNewsCluster_SimilarityAndGrouping(t *testing.T) {
	setupServiceNewsTestDB(t)

	// High similarity: same story from different feeds
	title1 := "OpenAI Announces GPT-4.5 With Expanded Context Window"
	title2 := "OpenAI Launches GPT-4.5 With 128k Expanded Context"
	sim := CalculateSimilarity(title1, title2)
	assert.Greater(t, sim, 0.40)

	// Low similarity: completely different topics
	title3 := "DeepSeek Releases MLA Paper on Memory Optimization"
	simDiff := CalculateSimilarity(title1, title3)
	assert.Less(t, simDiff, 0.20)

	// Test ClusterStory creation and corroboration
	item1 := &RawStoryItem{
		Title:      title1,
		URL:        "https://example.com/story1",
		Summary:    "OpenAI releases GPT-4.5 with large context window for API developers.",
		SourceId:   1,
		SourceName: "Source1",
		TrustTier:  "tier_1_official",
		Category:   "model_release",
		Tags:       []string{"openai", "gpt-4.5"},
	}

	cluster1, isNew1, err := ClusterStory(item1)
	require.NoError(t, err)
	assert.True(t, isNew1)
	require.NotNil(t, cluster1)
	assert.Positive(t, cluster1.Id)

	// Second story should corroborate into the same cluster
	item2 := &RawStoryItem{
		Title:      title2,
		URL:        "https://example.com/story2",
		Summary:    "GPT-4.5 context expansion details announced.",
		SourceId:   2,
		SourceName: "Source2",
		TrustTier:  "tier_2_syndicate",
		Category:   "model_release",
		Tags:       []string{"openai", "gpt-4.5"},
	}

	cluster2, isNew2, err := ClusterStory(item2)
	require.NoError(t, err)
	assert.False(t, isNew2)
	assert.Equal(t, cluster1.Id, cluster2.Id)
}

func TestNewsEditorial_RiskAndContent(t *testing.T) {
	// 1. Content risk assessment
	riskLow, statusLow := AssessContentRisk("Claude 3.7 Sonnet Released", "Anthropic announces new model version")
	assert.Equal(t, model.ContentRiskLow, riskLow)
	assert.Equal(t, model.NewsStatusPublished, statusLow)

	riskHigh, statusHigh := AssessContentRisk("Major Security Vulnerability Discovered in Cloud API", "Security breach leaked keys")
	assert.Equal(t, model.ContentRiskHigh, riskHigh)
	assert.Equal(t, model.NewsStatusReviewRequired, statusHigh)

	// 2. Slug generation
	slug := GenerateSlug("OpenAI GPT-4.5: The New Era of AI & APIs!")
	assert.Equal(t, "openai-gpt-4-5-the-new-era-of-ai-apis", slug)

	// 3. Markdown to safe HTML parsing
	md := "## สาระสำคัญ\n\n- ข้อที่ 1: **เร็วมาก**\n- ข้อที่ 2: ใช้ `token`\n\n```python\nprint('hello')\n```\n\nลิงก์: [Tora AI](https://www.toraapi.com)"
	html := RenderMarkdownToSafeHTML(md)
	assert.Contains(t, html, "<h2")
	assert.Contains(t, html, "สาระสำคัญ")
	assert.Contains(t, html, "<strong")
	assert.Contains(t, html, "<code class=")
	assert.Contains(t, html, "language-python")
	assert.Contains(t, html, "https://www.toraapi.com")
}

func TestNewsSEO_JSONLDAndSitemaps(t *testing.T) {
	post := &model.NewsPost{
		Id:           10,
		Slug:         "test-post-slug",
		Title:        "ทดสอบบทความ SEO",
		Summary:      "สรุปข้อมูลบทความเพื่อการทดสอบ",
		AuthorName:   "Tora Editorial",
		CanonicalUrl: common.GetCanonicalBaseURL() + "/news/test-post-slug",
		OgImageUrl:   common.GetCanonicalBaseURL() + "/news/test-post-slug/og.svg",
		Status:       model.NewsStatusPublished,
		PublishedAt:  time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	// 1. NewsArticle JSON-LD
	articleLD := GenerateNewsArticleJSONLD(post)
	assert.Contains(t, articleLD, `"@type":"NewsArticle"`)
	assert.Contains(t, articleLD, `"headline":"ทดสอบบทความ SEO"`)
	canonicalBase := common.GetCanonicalBaseURL()
	assert.Contains(t, articleLD, `"`+canonicalBase+`/news/test-post-slug"`)

	// 2. BreadcrumbList JSON-LD
	breadLD := GenerateBreadcrumbJSONLD(post)
	assert.Contains(t, breadLD, `"@type":"BreadcrumbList"`)
	assert.Contains(t, breadLD, canonicalBase+"/news")

	// 3. XML Sitemap
	sitemap := GenerateSitemapXML([]*model.NewsPost{post})
	assert.Contains(t, sitemap, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, sitemap, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	assert.Contains(t, sitemap, `<loc>`+canonicalBase+`/news/test-post-slug</loc>`)

	// 4. Google News Sitemap
	newsSitemap := GenerateNewsSitemapXML([]*model.NewsPost{post})
	assert.Contains(t, newsSitemap, `xmlns:news="http://www.google.com/schemas/sitemap-news/0.9"`)
	assert.Contains(t, newsSitemap, `<news:title><![CDATA[ทดสอบบทความ SEO]]></news:title>`)

	// 5. Robots.txt
	robots := GenerateRobotsTXT()
	assert.Contains(t, robots, "Allow: /news")
	assert.Contains(t, robots, "Sitemap: "+canonicalBase+"/sitemap.xml")
}

func TestNewsCreative_OGCardSVG(t *testing.T) {
	post := &model.NewsPost{
		Title:       "OpenAI GPT-4.5 Released: Benchmark Analysis and API Guide",
		AuthorName:  "Tora Architecture",
		SeoKeywords: "model_release,openai",
		PublishedAt: time.Now().Unix(),
	}

	svg := GenerateOGCardSVG(post)
	assert.Contains(t, svg, "<svg width=\"1200\" height=\"630\"")
	assert.Contains(t, svg, "Tora AI")
	assert.Contains(t, svg, "NEWSROOM")
	assert.Contains(t, svg, "MODEL_RELEASE")
	assert.Contains(t, svg, "OpenAI GPT-4.5")
}

func TestNewsDistribution_Derivatives(t *testing.T) {
	post := &model.NewsPost{
		Id:              25,
		Slug:            "claude-3-7-hybrid",
		Title:           "Claude 3.7 Sonnet Hybrid Reasoning",
		Summary:         "Anthropic releases hybrid reasoning architecture.",
		CanonicalUrl:    "https://www.toraapi.com/news/claude-3-7-hybrid",
		ContentMarkdown: "## รายละเอียด...",
	}

	dists := GenerateSocialDerivatives(post)
	require.Len(t, dists, 4)

	platforms := make(map[string]*model.NewsDistribution)
	for _, d := range dists {
		platforms[d.Platform] = d
		assert.Equal(t, 25, d.PostId)
		assert.Equal(t, "pending", d.Status)
	}

	assert.Contains(t, platforms, model.DistPlatformFacebook)
	assert.Contains(t, platforms, model.DistPlatformLinkedIn)
	assert.Contains(t, platforms, model.DistPlatformTwitter)
	assert.Contains(t, platforms, model.DistPlatformDevTo)

	assert.Contains(t, platforms[model.DistPlatformFacebook].ContentPayload, "#ToraAI")
	assert.Contains(t, platforms[model.DistPlatformDevTo].ContentPayload, "canonical_url: https://www.toraapi.com/news/claude-3-7-hybrid")
}

func TestNewsTask_SyncSingleNewsSource(t *testing.T) {
	setupServiceNewsTestDB(t)

	// Spin up mock RSS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Mock Tech Blog</title>
    <link>https://example.com</link>
    <item>
      <title>Gemini 2.5 Flash High Speed API Inference Announced</title>
      <link>https://example.com/gemini-2-5-flash</link>
      <description>Google announces sub-second latency for Gemini 2.5 Flash API with token efficiency.</description>
      <pubDate>Mon, 05 Oct 2026 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	src := &model.NewsSource{
		Name:                   "Mock Source",
		Slug:                   "mock-source",
		FeedUrl:                server.URL,
		SiteUrl:                "https://example.com",
		SourceType:             "rss",
		TrustTier:              "tier_1_official",
		Enabled:                true,
		PollingIntervalMinutes: 30,
	}
	require.NoError(t, model.CreateNewsSource(src))

	count, err := SyncSingleNewsSource(context.Background(), src)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	SetAutopilotPublishingEnabled(true)
	defer SetAutopilotPublishingEnabled(false)

	batchRes, err := RunAutopilotPublishBatch(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1, batchRes.ArticlesPublished)

	// Verify post created in database
	posts, total, err := model.GetPublishedNewsPosts(1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Gemini 2.5 Flash High Speed API Inference Announced", posts[0].Title)

	// Verify distributions created
	dists, err := model.GetDistributionsByPostId(posts[0].Id)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(dists), 4)
}

func TestNewsSEO_GoogleNewsSitemapFreshnessBoundary(t *testing.T) {
	refTime := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	refUnix := refTime.Unix()

	posts := []*model.NewsPost{
		{
			Id:           1,
			Slug:         "news-47h-ago",
			Title:        "Breaking News 47 Hours Ago",
			CanonicalUrl: "https://www.toraapi.com/news/news-47h-ago",
			ContentType:  model.ContentTypeNews,
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 47*3600, // 47 hours ago: eligible
		},
		{
			Id:           2,
			Slug:         "news-48h-ago",
			Title:        "Breaking News 48 Hours Ago Exact",
			CanonicalUrl: "https://www.toraapi.com/news/news-48h-ago",
			ContentType:  model.ContentTypeNews,
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 48*3600, // 48 hours ago: eligible (boundary)
		},
		{
			Id:           3,
			Slug:         "news-49h-ago",
			Title:        "Older News 49 Hours Ago",
			CanonicalUrl: "https://www.toraapi.com/news/news-49h-ago",
			ContentType:  model.ContentTypeNews,
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 49*3600, // 49 hours ago: aged out!
		},
		{
			Id:           4,
			Slug:         "evergreen-guide-1h-ago",
			Title:        "LLM Prompt Caching Complete Guide",
			CanonicalUrl: "https://www.toraapi.com/news/evergreen-guide-1h-ago",
			ContentType:  model.ContentTypeGuide, // Evergreen guide: MUST be excluded from Google News
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 3600,
		},
		{
			Id:           5,
			Slug:         "technical-analysis-1h-ago",
			Title:        "Deep Architecture Analysis",
			CanonicalUrl: "https://www.toraapi.com/news/technical-analysis-1h-ago",
			ContentType:  model.ContentTypeAnalysis, // Technical analysis: excluded from Google News
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 3600,
		},
		{
			Id:           6,
			Slug:         "changelog-1h-ago",
			Title:        "Tora Route Engine v2.0 Released",
			CanonicalUrl: "https://www.toraapi.com/news/changelog-1h-ago",
			ContentType:  model.ContentTypeChangelog, // Changelog: excluded from Google News
			Status:       model.NewsStatusPublished,
			PublishedAt:  refUnix - 3600,
		},
	}

	xmlOutput := GenerateNewsSitemapXMLWithTime(posts, refTime)
	canonicalBase := common.GetCanonicalBaseURL()

	// Inclusions
	assert.Contains(t, xmlOutput, canonicalBase+"/news/news-47h-ago", "47h fresh news MUST be included")
	assert.Contains(t, xmlOutput, canonicalBase+"/news/news-48h-ago", "48h boundary news MUST be included")
	assert.Contains(t, xmlOutput, "<news:publication>", "XML must contain Google News schema namespace")

	// Exclusions
	assert.NotContains(t, xmlOutput, canonicalBase+"/news/news-49h-ago", "49h news MUST be excluded from Google News sitemap")
	assert.NotContains(t, xmlOutput, canonicalBase+"/news/evergreen-guide-1h-ago", "Evergreen guide MUST be excluded from Google News sitemap")
	assert.NotContains(t, xmlOutput, canonicalBase+"/news/technical-analysis-1h-ago", "Analysis article MUST be excluded from Google News sitemap")
	assert.NotContains(t, xmlOutput, canonicalBase+"/news/changelog-1h-ago", "Changelog MUST be excluded from Google News sitemap")
}

func TestNewsCreative_GenerateOGCardPNG(t *testing.T) {
	post := &model.NewsPost{
		Title:       "OpenAI Launches GPT-4.5 with Reduced Hallucination & Sub-Second Latency",
		Summary:     "Deep technical breakdown for engineers building agentic workflows in Thailand.",
		AuthorName:  "Tora Engineering",
		SeoKeywords: "openai,gpt-4.5,llm,gateway",
		PublishedAt: common.GetTimestamp(),
	}

	pngBytes, err := GenerateOGCardPNG(post)
	require.NoError(t, err)
	require.NotEmpty(t, pngBytes)

	// Check PNG magic signature: \x89PNG\r\n\x1a\n
	assert.Equal(t, byte(0x89), pngBytes[0])
	assert.Equal(t, byte('P'), pngBytes[1])
	assert.Equal(t, byte('N'), pngBytes[2])
	assert.Equal(t, byte('G'), pngBytes[3])

	// Decode dimensions
	cfg, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.Equal(t, 1200, cfg.Width, "OG card width must be 1200px")
	assert.Equal(t, 630, cfg.Height, "OG card height must be 630px")
}

func TestNewsGSC_OpportunityDetectionAndCooldown(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Slug:            "test-gsc-article",
		Title:           "Claude 3.7 Hybrid Reasoning Guide",
		Summary:         "Comprehensive review of hybrid reasoning",
		ContentMarkdown: "## Hybrid Reasoning Overview\n\nClaude 3.7 allows switching between fast output and deep thinking.",
		Status:          model.NewsStatusPublished,
		PublishedAt:     common.GetTimestamp() - 70*86400, // 70 days ago
	}
	require.NoError(t, model.CreateNewsPost(post))

	metrics := []GSCMetricRow{
		{
			Page:        post.CanonicalUrl,
			Query:       "claude 3.7 reasoning api",
			Impressions: 150,
			Clicks:      2,
			CTR:         1.33,
			Position:    12.4,
			Date:        "2026-10-05",
		},
		{
			Page:        post.CanonicalUrl,
			Query:       "prompt caching discount", // Not in content!
			Impressions: 35,
			Clicks:      0,
			CTR:         0.0,
			Position:    4.1,
			Date:        "2026-10-05",
		},
	}

	opps, err := DetectSeoOpportunities([]*model.NewsPost{post}, metrics)
	require.NoError(t, err)
	require.NotEmpty(t, opps)

	oppTypes := make(map[string]bool)
	for _, opp := range opps {
		oppTypes[opp.OpportunityType] = true
	}

	assert.True(t, oppTypes[model.OpportunityHighImpLowCTR] || oppTypes[model.OpportunityHighImpLowCTRAlt], "High impressions (185) with low CTR (1.08%) should trigger OpportunityHighImpLowCTR")
	assert.True(t, oppTypes[model.OpportunityPosition8To20] || oppTypes[model.OpportunityPosition5To20], "Position 12.4 with 150 impressions should trigger striking distance opportunity")
	assert.True(t, oppTypes[model.OpportunityRisingQuery] || oppTypes[model.OpportunityNewQuery], "Query 'prompt caching discount' not in text should trigger new query opportunity")

	// Test 7-Day Cooldown Enforcement
	canRemediate, reason, err := CanRemediatePost(post.Id)
	require.NoError(t, err)
	assert.True(t, canRemediate, "Initially eligible before remediation applied")
	assert.Empty(t, reason)

	// Apply remediation
	oppToApply := opps[0]
	require.NoError(t, model.CreateNewsSeoOpportunity(oppToApply))
	err = ApplySeoRemediation(context.Background(), oppToApply)
	require.NoError(t, err)

	// Second check immediately after: MUST be blocked by 7-day cooldown
	canRemediateAfter, cooldownReason, err := CanRemediatePost(post.Id)
	require.NoError(t, err)
	assert.False(t, canRemediateAfter, "Must be blocked by active 7-day cooldown")
	assert.Contains(t, cooldownReason, "in 7-day cooldown window")
}

func TestNewsDistribution_ConnectorsAndOperatorBlocked(t *testing.T) {
	setupServiceNewsTestDB(t)

	post := &model.NewsPost{
		Slug:            "dist-safety-test",
		Title:           "Safety Verification for Multi-Channel Connectors",
		Summary:         "Summary for testing external distribution adapters",
		ContentMarkdown: "## Content",
		Status:          model.NewsStatusPublished,
		PublishedAt:     common.GetTimestamp(),
	}
	require.NoError(t, model.CreateNewsPost(post))
	require.NoError(t, QueuePostDistributions(post))

	// Process without external tokens configured -> MUST transition to operator_blocked safely
	processed, err := ProcessPendingDistributions(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "No posts published without credentials")

	dists, err := model.GetDistributionsByPostId(post.Id)
	require.NoError(t, err)
	require.NotEmpty(t, dists)

	for _, d := range dists {
		assert.Equal(t, "operator_blocked", d.Status)
		assert.Contains(t, d.ErrorMessage, "OPERATOR_BLOCKED")
		assert.NotEmpty(t, d.IdempotencyKey, "Idempotency key must be generated")
		assert.Equal(t, 1, d.AttemptCount)
	}
}

func TestNewsAttribution_TrackingAndDailyReview(t *testing.T) {
	setupServiceNewsTestDB(t)

	// 1. Attribution URL Builder
	taggedUrl := BuildAttributionURL("https://www.toraapi.com/news/gpt-4-5", "facebook", "social", "growth_autopilot", "gpt-4-5")
	assert.Contains(t, taggedUrl, "utm_source=facebook")
	assert.Contains(t, taggedUrl, "utm_medium=social")
	assert.Contains(t, taggedUrl, "utm_campaign=growth_autopilot")
	assert.Contains(t, taggedUrl, "utm_content=gpt-4-5")

	// 2. Client Identity Pseudonymization
	hash1 := HashClientIdentity("203.0.113.195", "Mozilla/5.0 Mac")
	hash2 := HashClientIdentity("203.0.113.195", "Mozilla/5.0 Mac")
	hash3 := HashClientIdentity("198.51.100.44", "Mozilla/5.0 Mac")
	assert.Equal(t, hash1, hash2, "Identical client must yield deterministic hash")
	assert.NotEqual(t, hash1, hash3, "Different IP must yield different hash")
	assert.NotContains(t, hash1, "203.0.113.195", "Hash must not leak raw IP")

	// 3. Record Conversion Funnel Events
	require.NoError(t, RecordConversion(ConversionOrganicLanding, "gpt-4-5", "google", "organic", "news", "203.0.113.1", "curl", 0, 0, ""))
	require.NoError(t, RecordConversion(ConversionSignup, "gpt-4-5", "google", "organic", "news", "203.0.113.1", "curl", 42, 0, ""))
	require.NoError(t, RecordConversion(ConversionTopup, "gpt-4-5", "google", "organic", "news", "203.0.113.1", "curl", 42, 500.0, ""))

	// 4. Generate Daily Growth Review
	todayStr := time.Now().Format("2006-01-02")
	review, err := GenerateDailyGrowthReview(todayStr)
	require.NoError(t, err)
	require.NotNil(t, review)

	assert.Equal(t, todayStr, review.ReviewDate)
	assert.Equal(t, 1, review.SignupsAttributed)
	assert.Equal(t, 1, review.ConversionsAttributed)
	assert.Contains(t, review.ReviewNotes, "News growth loop status")
}

