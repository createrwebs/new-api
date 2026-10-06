package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupNewsTestDB(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&NewsSource{},
		&StoryCluster{},
		&NewsPost{},
		&NewsDistribution{},
		&NewsAnalyticEvent{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
		InvalidateNewsCache()
	})
}

func TestNewsSource_CRUDAndSeeding(t *testing.T) {
	setupNewsTestDB(t)

	// 1. Initial seeding
	err := InitDefaultNewsSources()
	require.NoError(t, err)

	sources, err := GetAllNewsSources(true)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sources), 6)

	// 2. Check specific authoritative source
	var openaiSrc *NewsSource
	for _, s := range sources {
		if s.Slug == "openai-official" {
			openaiSrc = s
			break
		}
	}
	require.NotNil(t, openaiSrc)
	assert.Equal(t, "OpenAI Newsroom", openaiSrc.Name)
	assert.Equal(t, "tier_1_official", openaiSrc.TrustTier)

	// 3. Idempotent re-run
	err = InitDefaultNewsSources()
	require.NoError(t, err)
	sourcesAfter, err := GetAllNewsSources(true)
	require.NoError(t, err)
	assert.Equal(t, len(sources), len(sourcesAfter))

	// 4. Update source
	openaiSrc.PollingIntervalMinutes = 15
	err = UpdateNewsSource(openaiSrc)
	require.NoError(t, err)

	refetched, err := GetNewsSourceById(openaiSrc.Id)
	require.NoError(t, err)
	assert.Equal(t, 15, refetched.PollingIntervalMinutes)
}

func TestNewsPost_LifecycleAndSlugs(t *testing.T) {
	setupNewsTestDB(t)

	// 1. Create a published post
	post := &NewsPost{
		Slug:            "gpt-4-5-released",
		Title:           "OpenAI เปิดตัว GPT-4.5 พร้อม Context 128k",
		Summary:         "OpenAI ประกาศเปิดตัวโมเดล GPT-4.5 อย่างเป็นทางการ พร้อมความสามารถใหม่",
		ContentMarkdown: "# OpenAI เปิดตัว GPT-4.5\n\nรายละเอียดทางเทคนิค...",
		ContentHTML:     "<h1>OpenAI เปิดตัว GPT-4.5</h1><p>รายละเอียดทางเทคนิค...</p>",
		Status:          NewsStatusPublished,
		ContentRisk:     ContentRiskLow,
		FactCheckStatus: FactCheckVerified,
		SeoTitle:        "OpenAI เปิดตัว GPT-4.5: รายละเอียดและราคาต่อ 1M Tokens",
		SeoDescription:  "สรุปสเปกและราคา OpenAI GPT-4.5 ละเอียดครบถ้วนสำหรับนักพัฒนาไทย",
		SeoKeywords:     "openai,gpt-4.5,llm,api",
	}
	err := CreateNewsPost(post)
	require.NoError(t, err)
	require.Positive(t, post.Id)
	assert.Equal(t, common.GetCanonicalBaseURL()+"/news/gpt-4-5-released", post.CanonicalUrl)
	assert.Positive(t, post.PublishedAt)

	// 2. Fetch by slug
	fetched, err := GetNewsPostBySlug("gpt-4-5-released")
	require.NoError(t, err)
	assert.Equal(t, post.Title, fetched.Title)

	// 3. Verify view counter
	err = IncrementNewsPostView(post.Id)
	require.NoError(t, err)
	refetched, err := GetNewsPostById(post.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, refetched.ViewCount)

	// 4. Draft privacy: Draft post must NOT be returned by public GetNewsPostBySlug
	draftPost := &NewsPost{
		Slug:    "unreleased-draft-story",
		Title:   "Draft Story",
		Status:  NewsStatusDraft,
		Summary: "Not yet ready",
	}
	require.NoError(t, CreateNewsPost(draftPost))

	_, err = GetNewsPostBySlug("unreleased-draft-story")
	assert.Error(t, err, "draft post must not be queryable by public slug getter")

	// 5. Query published posts with pagination
	published, total, err := GetPublishedNewsPosts(1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, published, 1)

	// 6. Delete post
	err = DeleteNewsPost(post.Id)
	require.NoError(t, err)
	_, err = GetNewsPostById(post.Id)
	assert.Error(t, err)
}

func TestStoryCluster_Deduplication(t *testing.T) {
	setupNewsTestDB(t)

	cluster := &StoryCluster{
		Title:           "DeepSeek V3 Open Weights Released",
		Summary:         "DeepSeek releases 671B MoE model with exceptional code benchmarks.",
		PrimaryUrl:      "https://github.com/deepseek-ai/DeepSeek-V3",
		Category:        "model_release",
		Tags:            "deepseek,v3,open-source",
		RelevanceScore:  9.5,
		DeveloperScore:  9.8,
		ThailandScore:   8.0,
		Status:          "discovered",
	}
	err := CreateStoryCluster(cluster)
	require.NoError(t, err)
	require.Positive(t, cluster.Id)
	assert.Positive(t, cluster.FirstSeenAt)

	clusters, err := GetActiveStoryClusters(10)
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.Equal(t, "DeepSeek V3 Open Weights Released", clusters[0].Title)
}

func TestNewsDistributionAndAnalytics(t *testing.T) {
	setupNewsTestDB(t)

	// 1. Distribution record
	dist := &NewsDistribution{
		PostId:         100,
		Platform:       DistPlatformFacebook,
		Status:         "pending",
		ContentPayload: "สรุปข่าว AI วันนี้...",
	}
	err := CreateNewsDistribution(dist)
	require.NoError(t, err)

	dists, err := GetDistributionsByPostId(100)
	require.NoError(t, err)
	require.Len(t, dists, 1)
	assert.Equal(t, DistPlatformFacebook, dists[0].Platform)

	// 2. Analytic events
	require.NoError(t, RecordNewsAnalyticEvent(&NewsAnalyticEvent{
		PostId:    100,
		EventType: "read",
		UtmSource: "facebook",
	}))
	require.NoError(t, RecordNewsAnalyticEvent(&NewsAnalyticEvent{
		PostId:    100,
		EventType: "read",
		UtmSource: "google",
	}))
	require.NoError(t, RecordNewsAnalyticEvent(&NewsAnalyticEvent{
		PostId:    100,
		EventType: "signup",
		UtmSource: "google",
	}))

	summary, err := GetNewsAnalyticsSummary(100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), summary["read"])
	assert.Equal(t, int64(1), summary["signup"])
}

func TestNewsPosts_LaunchpackSeeding(t *testing.T) {
	setupNewsTestDB(t)

	// 1. Initial seeding
	err := InitDefaultNewsPosts()
	require.NoError(t, err)

	posts, total, err := GetPublishedNewsPosts(1, 20, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, posts, 6)

	// Verify specific seeded post content & canonical URL
	gptPost, err := GetNewsPostBySlug("openai-gpt-4-5-release-analysis")
	require.NoError(t, err)
	require.NotNil(t, gptPost)
	assert.Equal(t, common.GetCanonicalBaseURL()+"/news/openai-gpt-4-5-release-analysis", gptPost.CanonicalUrl)
	assert.Contains(t, gptPost.ContentHTML, "GPT-4.5")
	assert.Equal(t, NewsStatusPublished, gptPost.Status)
	assert.Equal(t, ContentRiskLow, gptPost.ContentRisk)

	// 2. Idempotency
	err = InitDefaultNewsPosts()
	require.NoError(t, err)
	_, totalAfter, err := GetPublishedNewsPosts(1, 20, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, total, totalAfter)
}
