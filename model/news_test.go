package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

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
		&NewsPublicationEvent{},
		&NewsAutopilotDailyQuota{},
	))
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_news_pub_events_unique_initial ON news_publication_events (post_id, event_type);")
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

func TestBangkokDateBoundaries(t *testing.T) {
	// Verify Bangkok UTC+7 boundaries
	// 2026-10-06 16:59:59 UTC -> 2026-10-06 23:59:59 Bangkok
	utcJustBeforeMidnight := time.Date(2026, 10, 6, 16, 59, 59, 0, time.UTC)
	startUnix, endUnix, dateStr := GetBangkokDateRange(utcJustBeforeMidnight)
	assert.Equal(t, "2026-10-06", dateStr)

	loc := time.FixedZone("Asia/Bangkok", 7*3600)
	expectedStart := time.Date(2026, 10, 6, 0, 0, 0, 0, loc).Unix()
	expectedEnd := time.Date(2026, 10, 7, 0, 0, 0, 0, loc).Unix()
	assert.Equal(t, expectedStart, startUnix)
	assert.Equal(t, expectedEnd, endUnix)

	// 2026-10-06 17:00:00 UTC -> 2026-10-07 00:00:00 Bangkok
	utcExactlyMidnight := time.Date(2026, 10, 6, 17, 0, 0, 0, time.UTC)
	_, _, dateStrMidnight := GetBangkokDateRange(utcExactlyMidnight)
	assert.Equal(t, "2026-10-07", dateStrMidnight)
}

func TestPublishPostWithAtomicQuota(t *testing.T) {
	setupNewsTestDB(t)

	// 1. Create a source
	src := &NewsSource{
		Name:    "Test Source",
		Slug:    "test-source",
		FeedUrl: "https://example.com/rss",
	}
	require.NoError(t, DB.Create(src).Error)

	// 2. Test daily cap enforcement with maxDaily = 2, maxPerSource = 3
	post1 := &NewsPost{
		Slug:    "post-1",
		Title:   "Story 1",
		Summary: "Summary 1",
		Status:  NewsStatusDraft,
	}
	require.NoError(t, DB.Create(post1).Error)

	params1 := &AutopilotPublishParams{
		Post:            post1,
		ClusterId:       1,
		PrimarySourceId: src.Id,
		BatchId:         "batch-1",
		MaxDailyCap:     2,
		MaxSourceCap:    3,
	}
	res1, err := PublishPostWithAtomicQuota(params1)
	require.NoError(t, err)
	assert.True(t, res1.Success)

	todayCount, err := GetAutopilotPublishedToday()
	require.NoError(t, err)
	assert.Equal(t, 1, todayCount)

	srcCount, err := GetPublishedCountBySourceToday(src.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, srcCount)

	// Second post should succeed
	post2 := &NewsPost{
		Slug:    "post-2",
		Title:   "Story 2",
		Summary: "Summary 2",
		Status:  NewsStatusDraft,
	}
	require.NoError(t, DB.Create(post2).Error)

	params2 := &AutopilotPublishParams{
		Post:            post2,
		ClusterId:       2,
		PrimarySourceId: src.Id,
		BatchId:         "batch-1",
		MaxDailyCap:     2,
		MaxSourceCap:    3,
	}
	res2, err := PublishPostWithAtomicQuota(params2)
	require.NoError(t, err)
	assert.True(t, res2.Success)

	todayCount2, err := GetAutopilotPublishedToday()
	require.NoError(t, err)
	assert.Equal(t, 2, todayCount2)

	// Third post should be rejected due to daily cap (2)
	post3 := &NewsPost{
		Slug:    "post-3",
		Title:   "Story 3",
		Summary: "Summary 3",
		Status:  NewsStatusDraft,
	}
	require.NoError(t, DB.Create(post3).Error)

	params3 := &AutopilotPublishParams{
		Post:            post3,
		ClusterId:       3,
		PrimarySourceId: src.Id,
		BatchId:         "batch-1",
		MaxDailyCap:     2,
		MaxSourceCap:    3,
	}
	res3, err := PublishPostWithAtomicQuota(params3)
	require.NoError(t, err)
	assert.False(t, res3.Success)
	assert.Contains(t, res3.RejectReason, "Daily autopilot quota reached")

	// Post 3 should remain a draft
	refetchedPost3, err := GetNewsPostById(post3.Id)
	require.NoError(t, err)
	assert.Equal(t, NewsStatusDraft, refetchedPost3.Status)

	// 3. Test per-source limit enforcement
	src2 := &NewsSource{
		Name:    "Source 2",
		Slug:    "source-2",
		FeedUrl: "https://example2.com/rss",
	}
	require.NoError(t, DB.Create(src2).Error)

	// Post 4 from source 2 with maxDaily = 10, maxPerSource = 1
	post4 := &NewsPost{
		Slug:    "post-4",
		Title:   "Story 4",
		Summary: "Summary 4",
		Status:  NewsStatusDraft,
	}
	require.NoError(t, DB.Create(post4).Error)

	params4 := &AutopilotPublishParams{
		Post:            post4,
		ClusterId:       4,
		PrimarySourceId: src2.Id,
		BatchId:         "batch-2",
		MaxDailyCap:     10,
		MaxSourceCap:    1,
	}
	res4, err := PublishPostWithAtomicQuota(params4)
	require.NoError(t, err)
	assert.True(t, res4.Success)

	// Post 5 from same source 2 should be rejected by source cap
	post5 := &NewsPost{
		Slug:    "post-5",
		Title:   "Story 5",
		Summary: "Summary 5",
		Status:  NewsStatusDraft,
	}
	require.NoError(t, DB.Create(post5).Error)

	params5 := &AutopilotPublishParams{
		Post:            post5,
		ClusterId:       5,
		PrimarySourceId: src2.Id,
		BatchId:         "batch-2",
		MaxDailyCap:     10,
		MaxSourceCap:    1,
	}
	res5, err := PublishPostWithAtomicQuota(params5)
	require.NoError(t, err)
	assert.False(t, res5.Success)
	assert.Contains(t, res5.RejectReason, "Source daily cap reached")
}

func TestPublicationCounters_Isolation(t *testing.T) {
	setupNewsTestDB(t)

	now := time.Now().Unix()

	// 1 Autopilot post
	p1 := &NewsPost{
		Slug:              "p-autopilot",
		Title:             "Autopilot post",
		Status:            NewsStatusPublished,
		PublicationOrigin: PublicationOriginAutopilot,
		PublishedAt:       now,
	}
	require.NoError(t, DB.Create(p1).Error)

	// Record corresponding publication event so Autopilot counter picks it up
	_, _, dateStr := GetBangkokDateRange(time.Now())
	require.NoError(t, DB.Create(&NewsPublicationEvent{
		PostId:                 p1.Id,
		EventType:              EventTypeAutopilotInitialPublish,
		PublicationOrigin:      PublicationOriginAutopilot,
		BangkokPublicationDate: dateStr,
		PublishedAt:            now,
	}).Error)

	// 1 Seed post
	p2 := &NewsPost{
		Slug:              "p-seed",
		Title:             "Seed post",
		Status:            NewsStatusPublished,
		IsSeed:            true,
		PublicationOrigin: PublicationOriginSeed,
		PublishedAt:       now,
	}
	require.NoError(t, DB.Create(p2).Error)

	// 1 Manual post
	p3 := &NewsPost{
		Slug:              "p-manual",
		Title:             "Manual post",
		Status:            NewsStatusPublished,
		PublicationOrigin: PublicationOriginManualAdmin,
		PublishedAt:       now,
	}
	require.NoError(t, DB.Create(p3).Error)

	// 1 Legacy post
	p4 := &NewsPost{
		Slug:              "p-legacy",
		Title:             "Legacy post",
		Status:            NewsStatusPublished,
		PublicationOrigin: PublicationOriginUnknownLegacy,
		PublishedAt:       now,
	}
	require.NoError(t, DB.Create(p4).Error)

	// 1 Draft
	p5 := &NewsPost{
		Slug:   "p-draft",
		Title:  "Draft post",
		Status: NewsStatusDraft,
	}
	require.NoError(t, DB.Create(p5).Error)

	counters, err := GetAuthoritativePublicationCounters()
	require.NoError(t, err)

	assert.Equal(t, 1, counters.AutopilotPublishedToday, "Autopilot counter must be strictly 1")
	assert.Equal(t, 1, counters.SeedOrHistoricalToday, "Seed counter must be 1")
	assert.Equal(t, 1, counters.ManualPublishedToday, "Manual counter must be 1")
	assert.Equal(t, 1, counters.LegacyUnknownToday, "Legacy counter must be 1")
	assert.Equal(t, 1, counters.DraftsToday, "Drafts counter must be 1")
	assert.Equal(t, 4, counters.PublishedTodayTotal, "Total published must be 4")
}

func TestNewsPublicationEvent_Idempotency(t *testing.T) {
	setupNewsTestDB(t)

	// Create event
	evt1 := &NewsPublicationEvent{
		PostId:                 42,
		EventType:              EventTypeAutopilotInitialPublish,
		PublicationOrigin:      PublicationOriginAutopilot,
		BangkokPublicationDate: "2026-10-06",
		PublishedAt:            time.Now().Unix(),
	}
	require.NoError(t, DB.Create(evt1).Error)

	// Attempt duplicate event with identical PostId and EventType
	evt2 := &NewsPublicationEvent{
		PostId:                 42,
		EventType:              EventTypeAutopilotInitialPublish,
		PublicationOrigin:      PublicationOriginAutopilot,
		BangkokPublicationDate: "2026-10-06",
		PublishedAt:            time.Now().Unix(),
	}
	err := DB.Create(evt2).Error
	require.Error(t, err, "Duplicate publication event should violate unique constraint")
}

