package service

import (
	"fmt"
	"os"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// GenerateDailyGrowthReview compiles a comprehensive 24-hour summary of the news and growth pipeline
func GenerateDailyGrowthReview(targetDate string) (*model.NewsDailyGrowthReview, error) {
	if targetDate == "" {
		targetDate = time.Now().Format("2006-01-02")
	}

	// 1. Content Inventory Breakdown
	var newsCount, guideCount, analysisCount, changelogCount int64
	model.DB.Model(&model.NewsPost{}).Where("status = ? AND content_type = ?", model.NewsStatusPublished, model.ContentTypeNews).Count(&newsCount)
	model.DB.Model(&model.NewsPost{}).Where("status = ? AND content_type = ?", model.NewsStatusPublished, model.ContentTypeGuide).Count(&guideCount)
	model.DB.Model(&model.NewsPost{}).Where("status = ? AND content_type = ?", model.NewsStatusPublished, model.ContentTypeAnalysis).Count(&analysisCount)
	model.DB.Model(&model.NewsPost{}).Where("status = ? AND content_type = ?", model.NewsStatusPublished, model.ContentTypeChangelog).Count(&changelogCount)

	totalPublishedToday := int(newsCount + guideCount + analysisCount + changelogCount)

	// 2. SEO Performance Totals
	type SeoSum struct {
		TotalImpressions int
		TotalClicks      int
	}
	var seoSummary SeoSum
	_ = model.DB.Model(&model.NewsSeoMetric{}).Select("COALESCE(SUM(impressions), 0) as total_impressions, COALESCE(SUM(clicks), 0) as total_clicks").Scan(&seoSummary)

	avgCTR := 0.0
	if seoSummary.TotalImpressions > 0 {
		avgCTR = (float64(seoSummary.TotalClicks) / float64(seoSummary.TotalImpressions)) * 100.0
	}

	// 3. High-Priority SEO Opportunities Pending
	var oppsCount int64
	model.DB.Model(&model.NewsSeoOpportunity{}).Where("status = ?", "detected").Count(&oppsCount)

	// 4. Distribution Pipeline Health
	var distPub, distFail, distBlock int64
	model.DB.Model(&model.NewsDistribution{}).Where("status = ?", "published").Count(&distPub)
	model.DB.Model(&model.NewsDistribution{}).Where("status = ?", "failed").Count(&distFail)
	model.DB.Model(&model.NewsDistribution{}).Where("status = ?", "operator_blocked").Count(&distBlock)

	// 5. Attributed Conversions
	var signupsCount, conversionsCount int64
	model.DB.Model(&model.NewsConversionEvent{}).Where("event_type = ?", ConversionSignup).Count(&signupsCount)
	model.DB.Model(&model.NewsConversionEvent{}).Where("event_type IN ?", []string{ConversionTopup, ConversionSubscription}).Count(&conversionsCount)

	// 6. Overall Pipeline Health Determination
	healthStatus := "healthy"
	if distBlock > 0 {
		healthStatus = "operator_blocked"
	} else if distFail > 0 {
		healthStatus = "degraded"
	}

	notes := fmt.Sprintf(
		"News growth loop status for %s:\n"+
			"- Content types: %d news, %d evergreen guides, %d technical analysis, %d changelogs\n"+
			"- SEO metrics: %d impressions, %d clicks (CTR: %.2f%%)\n"+
			"- Pending SEO opportunities: %d\n"+
			"- Distribution statuses: %d published, %d operator-blocked, %d failed\n"+
			"- Funnel conversions: %d signups, %d commercial transactions",
		targetDate, newsCount, guideCount, analysisCount, changelogCount,
		seoSummary.TotalImpressions, seoSummary.TotalClicks, avgCTR,
		oppsCount,
		distPub, distBlock, distFail,
		signupsCount, conversionsCount,
	)

	review := &model.NewsDailyGrowthReview{
		ReviewDate:             targetDate,
		PostsPublishedToday:    totalPublishedToday,
		NewsCount:              int(newsCount),
		GuideCount:             int(guideCount),
		AnalysisCount:          int(analysisCount),
		ChangelogCount:         int(changelogCount),
		TotalImpressions:       seoSummary.TotalImpressions,
		TotalClicks:            seoSummary.TotalClicks,
		AverageCTR:             avgCTR,
		HighPriorityOppsCount:  int(oppsCount),
		DistributionsPublished: int(distPub),
		DistributionsFailed:    int(distFail),
		DistributionsBlocked:   int(distBlock),
		SignupsAttributed:      int(signupsCount),
		ConversionsAttributed:  int(conversionsCount),
		PipelineHealthStatus:   healthStatus,
		ReviewNotes:            notes,
		CreatedAt:              common.GetTimestamp(),
	}

	if err := model.CreateOrUpdateDailyGrowthReview(review); err != nil {
		return nil, err
	}

	return review, nil
}

// PostGrowthRecord provides an unified growth and distribution record for a post (Section 19)
type PostGrowthRecord struct {
	PostId           int                         `json:"post_id"`
	Slug             string                      `json:"slug"`
	Title            string                      `json:"title"`
	ContentType      string                      `json:"content_type"`
	Status           string                      `json:"status"`
	PublishedAt      int64                       `json:"published_at"`
	CanonicalUrl     string                      `json:"canonical_url"`
	Distributions    []*model.NewsDistribution   `json:"distributions"`
	SeoMetrics       []*model.NewsSeoMetric      `json:"seo_metrics"`
	TotalImpressions int                         `json:"total_impressions"`
	TotalClicks      int                         `json:"total_clicks"`
	AverageCTR       float64                     `json:"average_ctr"`
	Opportunities    []*model.NewsSeoOpportunity `json:"opportunities"`
	SignupsCount     int                         `json:"signups_count"`
	ConversionsCount int                         `json:"conversions_count"`
}

// GetPostGrowthRecord compiles all growth telemetry associated with a single news post
func GetPostGrowthRecord(postId int) (*PostGrowthRecord, error) {
	post, err := model.GetNewsPostById(postId)
	if err != nil || post == nil {
		return nil, fmt.Errorf("post %d not found", postId)
	}

	dists, _ := model.GetDistributionsByPostId(postId)
	metrics, _ := model.GetNewsSeoMetricsByPostId(postId, 50)
	opps, _ := model.GetSeoOpportunitiesByPost(postId)

	totalImp := 0
	totalClicks := 0
	for _, m := range metrics {
		totalImp += m.Impressions
		totalClicks += m.Clicks
	}
	avgCTR := 0.0
	if totalImp > 0 {
		avgCTR = (float64(totalClicks) / float64(totalImp)) * 100.0
	}

	contentIdStr := fmt.Sprintf("%d", postId)
	var signups int64
	var conversions int64
	model.DB.Model(&model.NewsConversionEvent{}).Where("content_id = ? AND event_type = ?", contentIdStr, ConversionSignup).Count(&signups)
	model.DB.Model(&model.NewsConversionEvent{}).Where("content_id = ? AND event_type IN ?", contentIdStr, []string{ConversionTopup, ConversionSubscription}).Count(&conversions)

	return &PostGrowthRecord{
		PostId:           post.Id,
		Slug:             post.Slug,
		Title:            post.Title,
		ContentType:      post.ContentType,
		Status:           post.Status,
		PublishedAt:      post.PublishedAt,
		CanonicalUrl:     post.CanonicalUrl,
		Distributions:    dists,
		SeoMetrics:       metrics,
		TotalImpressions: totalImp,
		TotalClicks:      totalClicks,
		AverageCTR:       avgCTR,
		Opportunities:    opps,
		SignupsCount:     int(signups),
		ConversionsCount: int(conversions),
	}, nil
}

// GlobalGrowthOverview provides an operational snapshot of the growth engine (Section 19 & 21)
type GlobalGrowthOverview struct {
	GlobalKillSwitchActive bool                         `json:"global_kill_switch_active"`
	DistributionEnabled    bool                         `json:"distribution_enabled"`
	ChannelStatuses        map[string]string            `json:"channel_statuses"`
	AllowlistPostIDs       []int                        `json:"allowlist_post_ids"`
	PendingQueueDepth      map[string]int64             `json:"pending_queue_depth"`
	LatestDailyReview      *model.NewsDailyGrowthReview `json:"latest_daily_review"`
}

// GetGlobalGrowthOverview evaluates runtime growth state across all channels
func GetGlobalGrowthOverview() (*GlobalGrowthOverview, error) {
	distEnabled := IsGlobalDistributionEnabled()
	killSwitchActive := !distEnabled

	channelStatuses := make(map[string]string)

	// GSC
	if os.Getenv("GSC_CREDENTIALS_FILE") != "" || os.Getenv("GSC_CREDENTIALS_JSON") != "" || os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		channelStatuses["gsc"] = "CONFIGURED"
	} else {
		channelStatuses["gsc"] = "OPERATOR_BLOCKED"
	}

	// DEV.to
	if os.Getenv("DEVTO_API_KEY") != "" {
		if !IsPlatformDistributionEnabled(model.DistPlatformDevTo) {
			channelStatuses["devto"] = "DISABLED_BY_CONFIG"
		} else {
			channelStatuses["devto"] = "ACTIVE"
		}
	} else {
		channelStatuses["devto"] = "OPERATOR_BLOCKED"
	}

	// Facebook
	if os.Getenv("FACEBOOK_PAGE_ACCESS_TOKEN") != "" && os.Getenv("FACEBOOK_PAGE_ID") != "" {
		if !IsPlatformDistributionEnabled(model.DistPlatformFacebook) {
			channelStatuses["facebook"] = "DISABLED_BY_CONFIG"
		} else {
			channelStatuses["facebook"] = "ACTIVE"
		}
	} else {
		channelStatuses["facebook"] = "OPERATOR_BLOCKED"
	}

	// LinkedIn
	if os.Getenv("LINKEDIN_ACCESS_TOKEN") != "" && os.Getenv("LINKEDIN_ORG_ID") != "" {
		if !IsPlatformDistributionEnabled(model.DistPlatformLinkedIn) {
			channelStatuses["linkedin"] = "DISABLED_BY_CONFIG"
		} else {
			channelStatuses["linkedin"] = "ACTIVE"
		}
	} else {
		channelStatuses["linkedin"] = "OPERATOR_BLOCKED"
	}

	// Allowlist post IDs
	allowlistMap := GetDistributionAllowlistPostIDs()
	var allowlistIDs []int
	for id := range allowlistMap {
		allowlistIDs = append(allowlistIDs, id)
	}

	// Queue depth
	queueDepth := make(map[string]int64)
	type QueueRow struct {
		Platform string
		Count    int64
	}
	var rows []QueueRow
	_ = model.DB.Model(&model.NewsDistribution{}).
		Select("platform, count(*) as count").
		Where("status = ?", "pending").
		Group("platform").
		Scan(&rows)
	for _, r := range rows {
		queueDepth[r.Platform] = r.Count
	}

	recentReviews, _ := model.GetRecentDailyGrowthReviews(1)
	var latestReview *model.NewsDailyGrowthReview
	if len(recentReviews) > 0 {
		latestReview = recentReviews[0]
	}

	return &GlobalGrowthOverview{
		GlobalKillSwitchActive: killSwitchActive,
		DistributionEnabled:    distEnabled,
		ChannelStatuses:        channelStatuses,
		AllowlistPostIDs:       allowlistIDs,
		PendingQueueDepth:      queueDepth,
		LatestDailyReview:      latestReview,
	}, nil
}
