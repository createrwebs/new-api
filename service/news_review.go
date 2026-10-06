package service

import (
	"context"
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

// GrowthTimelineEvent models a chronological milestone in an article's growth lifecycle (Section 15)
type GrowthTimelineEvent struct {
	Timestamp int64  `json:"timestamp"`
	EventType string `json:"event_type"` // "published", "distributed", "inspected", "metrics_updated", "opportunity_detected", "experiment_applied"
	Channel   string `json:"channel,omitempty"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
}

// PostGrowthRecord provides an unified growth and distribution record for a post (Section 15)
type PostGrowthRecord struct {
	PostId           int                          `json:"post_id"`
	Slug             string                       `json:"slug"`
	Title            string                       `json:"title"`
	ContentType      string                       `json:"content_type"`
	Status           string                       `json:"status"`
	PublishedAt      int64                        `json:"published_at"`
	CanonicalUrl     string                       `json:"canonical_url"`
	Distributions    []*model.NewsDistribution    `json:"distributions"`
	SeoMetrics       []*model.NewsSeoMetric       `json:"seo_metrics"`
	TotalImpressions int                          `json:"total_impressions"`
	TotalClicks      int                          `json:"total_clicks"`
	AverageCTR       float64                      `json:"average_ctr"`
	AveragePosition  float64                      `json:"average_position"`
	Opportunities    []*model.NewsSeoOpportunity  `json:"opportunities"`
	UrlInspection    *model.NewsUrlInspection     `json:"url_inspection,omitempty"`
	Experiments      []*model.NewsSeoExperiment   `json:"experiments,omitempty"`
	DevToAnalytics   *DevToAnalytics              `json:"devto_analytics,omitempty"`
	InternalLinks    []InternalLinkRecommendation `json:"internal_links,omitempty"`
	Timeline         []GrowthTimelineEvent        `json:"timeline,omitempty"`
	SignupsCount     int                          `json:"signups_count"`
	ConversionsCount int                          `json:"conversions_count"`
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
	insp, _ := model.GetLatestUrlInspectionByPostId(postId)
	exps, _ := model.GetSeoExperimentsByPostId(postId)
	links, _ := RecommendInternalLinks(postId)

	totalImp := 0
	totalClicks := 0
	var weightedPosSum float64
	for _, m := range metrics {
		totalImp += m.Impressions
		totalClicks += m.Clicks
		if m.Impressions > 0 {
			weightedPosSum += m.Position * float64(m.Impressions)
		}
	}
	avgCTR := 0.0
	avgPos := 0.0
	if totalImp > 0 {
		avgCTR = (float64(totalClicks) / float64(totalImp)) * 100.0
		avgPos = weightedPosSum / float64(totalImp)
	}

	contentIdStr := fmt.Sprintf("%d", postId)
	var signups int64
	var conversions int64
	model.DB.Model(&model.NewsConversionEvent{}).Where("content_id = ? AND event_type = ?", contentIdStr, ConversionSignup).Count(&signups)
	model.DB.Model(&model.NewsConversionEvent{}).Where("content_id = ? AND event_type IN ?", contentIdStr, []string{ConversionTopup, ConversionSubscription}).Count(&conversions)

	// Fetch real DEV.to analytics if remote article exists (Section 14)
	var devToAnalytics *DevToAnalytics
	for _, d := range dists {
		if d.Platform == model.DistPlatformDevTo && (d.RemotePostId != "" || d.ExternalPostId != "") {
			remoteId := d.RemotePostId
			if remoteId == "" {
				remoteId = d.ExternalPostId
			}
			pub := NewDevToPublisher()
			analytics, err := pub.GetArticleAnalytics(context.Background(), remoteId)
			if err == nil {
				analytics.PublishedAt = d.PublishedAt
				devToAnalytics = analytics
			}
			break
		}
	}

	// Build unified chronological timeline (Section 15)
	var timeline []GrowthTimelineEvent
	if post.PublishedAt > 0 {
		timeline = append(timeline, GrowthTimelineEvent{
			Timestamp: post.PublishedAt,
			EventType: "published",
			Channel:   "tora_web",
			Summary:   fmt.Sprintf("Published on Tora News at %s", post.CanonicalUrl),
			Status:    "active",
		})
	}
	for _, d := range dists {
		if d.PublishedAt > 0 {
			timeline = append(timeline, GrowthTimelineEvent{
				Timestamp: d.PublishedAt,
				EventType: "distributed",
				Channel:   d.Platform,
				Summary:   fmt.Sprintf("Cross-posted to %s (Remote ID: %s)", d.Platform, d.RemotePostId),
				Status:    d.Status,
			})
		}
	}
	if insp != nil && insp.InspectionTime > 0 {
		timeline = append(timeline, GrowthTimelineEvent{
			Timestamp: insp.InspectionTime,
			EventType: "inspected",
			Channel:   "google_search_console",
			Summary:   fmt.Sprintf("Google Search Console URL inspection: %s (Verdict: %s)", insp.CoverageState, insp.Verdict),
			Status:    insp.Verdict,
		})
	}
	for _, exp := range exps {
		timeline = append(timeline, GrowthTimelineEvent{
			Timestamp: exp.AppliedAt,
			EventType: "experiment_applied",
			Channel:   "seo_autopilot",
			Summary:   fmt.Sprintf("Applied SEO experiment (%s): %s", exp.ChangeType, exp.ResultVerdict),
			Status:    exp.ResultVerdict,
		})
	}

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
		AveragePosition:  avgPos,
		Opportunities:    opps,
		UrlInspection:    insp,
		Experiments:      exps,
		DevToAnalytics:   devToAnalytics,
		InternalLinks:    links,
		Timeline:         timeline,
		SignupsCount:     int(signups),
		ConversionsCount: int(conversions),
	}, nil
}

// GlobalGrowthOverview provides an operational snapshot of the growth engine (Section 2, 18, 21)
type GlobalGrowthOverview struct {
	GlobalKillSwitchActive bool                         `json:"global_kill_switch_active"`
	DistributionEnabled    bool                         `json:"distribution_enabled"`
	ChannelStatuses        map[string]string            `json:"channel_statuses"`
	AllowlistPostIDs       []int                        `json:"allowlist_post_ids"`
	PendingQueueDepth      map[string]int64             `json:"pending_queue_depth"`
	LatestDailyReview      *model.NewsDailyGrowthReview `json:"latest_daily_review"`
	GSCStatus              string                       `json:"gsc_status"`
	GSCDataAvailable       bool                         `json:"gsc_data_available"`
	GSCRowCount            int64                        `json:"gsc_row_count"`
	GSCSiteURL             string                       `json:"gsc_site_url"`
	MassAutopublish        bool                         `json:"mass_autopublish"`
	DevToUpdatePolicy      string                       `json:"devto_update_policy"`
}

// GetGlobalGrowthOverview evaluates runtime growth state across all channels
func GetGlobalGrowthOverview() (*GlobalGrowthOverview, error) {
	distEnabled := IsGlobalDistributionEnabled()
	killSwitchActive := !distEnabled

	channelStatuses := make(map[string]string)

	// Normalized GSC state (Section 2)
	gscStatus := GetNormalizedGSCStatus(context.Background(), false)
	channelStatuses["gsc"] = gscStatus.Status

	// DEV.to
	devPub := NewDevToPublisher()
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

	// Allowlist post IDs (Section 18: empty fails closed in canary mode)
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
		GSCStatus:              gscStatus.Status,
		GSCDataAvailable:       gscStatus.DataAvailable,
		GSCRowCount:            gscStatus.RowCount,
		GSCSiteURL:             gscStatus.SiteURL,
		MassAutopublish:        false, // Strictly disabled (Section 17)
		DevToUpdatePolicy:      devPub.UpdatePolicy,
	}, nil
}
