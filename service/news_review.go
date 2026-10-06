package service

import (
	"fmt"
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
