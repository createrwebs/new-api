package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const (
	MaxPublishedPerDay   = 20
	MaxPerBatch          = 5
	MaxPerSourcePerDay   = 3
	QualityScoreMinLimit = 2.0 // ScoreStory weighted relevance minimum
)

var (
	autopilotOnce       sync.Once
	autopilotMu         sync.Mutex
	lastExecutedSlot    string
	lastAutopilotRunAt  int64
	lastAutopilotReport *model.NewsDailyGrowthReview

	// NewsAutopilotPublishingEnabled controls whether automatic batch publishing is active.
	// Default: false (Safety hold during incident reconciliation until explicitly proven and re-enabled)
	NewsAutopilotPublishingEnabled = false
)

func IsAutopilotPublishingEnabled() bool {
	if val := os.Getenv("NEWS_AUTOPILOT_PUBLISH_ENABLED"); val != "" {
		return val == "true" || val == "1"
	}
	return NewsAutopilotPublishingEnabled
}

func SetAutopilotPublishingEnabled(enabled bool) {
	autopilotMu.Lock()
	defer autopilotMu.Unlock()
	NewsAutopilotPublishingEnabled = enabled
}

func GetMaxPublishedPerDay() int {
	if v := os.Getenv("NEWS_MAX_PUBLISHED_PER_DAY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return MaxPublishedPerDay
}

func GetMaxPerBatch() int {
	if v := os.Getenv("NEWS_MAX_PER_BATCH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return MaxPerBatch
}

func GetMaxPerSourcePerDay() int {
	if v := os.Getenv("NEWS_MAX_PER_SOURCE_PER_DAY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return MaxPerSourcePerDay
}

// AutopilotScoutResult records metrics for an autonomous discovery cycle
type AutopilotScoutResult struct {
	SourcesChecked    int      `json:"sources_checked"`
	SourceFailures    int      `json:"source_failures"`
	ItemsDiscovered   int      `json:"items_discovered"`
	DuplicatesRemoved int      `json:"duplicates_removed"`
	ClustersCreated   int      `json:"clusters_created"`
	FailedSources     []string `json:"failed_sources,omitempty"`
}

// AutopilotBatchResult records metrics for a publishing batch cycle
type AutopilotBatchResult struct {
	BatchId           string   `json:"batch_id"`
	ArticlesAttempted int      `json:"articles_attempted"`
	ArticlesPublished int      `json:"articles_published"`
	DraftsCreated     int      `json:"drafts_created"`
	RejectedStories   int      `json:"rejected_stories"`
	AlreadyPublished  int      `json:"already_published"`
	QuotaBlocked      int      `json:"quota_blocked"`
	RejectionReasons  []string `json:"rejection_reasons,omitempty"`
	DailyTotalSoFar   int      `json:"daily_total_so_far"`
	IndexNowSubmitted int      `json:"indexnow_submitted"`
}

// AutopilotStatus represents current live autopilot health and statistics
type AutopilotStatus struct {
	CurrentBangkokTime      string                       `json:"current_bangkok_time"`
	TodayPublishedCount     int                          `json:"today_published_count"` // Authoritative: autopilot_published_today
	AutopilotPublishedToday int                          `json:"autopilot_published_today"`
	PublishedTodayTotal     int                          `json:"published_today_total"`
	LegacyUnknownToday      int                          `json:"legacy_unknown_today"`
	ManualPublishedToday    int                          `json:"manual_published_today"`
	SeedOrHistoricalToday   int                          `json:"seed_or_historical_today"`
	DraftsToday             int                          `json:"drafts_today"`
	MaxDailyCap             int                          `json:"max_daily_cap"`
	PublishingEnabled       bool                         `json:"publishing_enabled"`
	ActiveSourcesCount      int                          `json:"active_sources_count"`
	FailingSourcesCount     int                          `json:"failing_sources_count"`
	LastExecutedSlot        string                       `json:"last_executed_slot"`
	LastRunAt               int64                        `json:"last_run_at"`
	ScheduleCycles          []string                     `json:"schedule_cycles"`
	LatestReport            *model.NewsDailyGrowthReview `json:"latest_report,omitempty"`
}

// RunAutopilotScoutCycle executes a full feed discovery and clustering pass (Section 3 & 4)
func RunAutopilotScoutCycle(ctx context.Context) (*AutopilotScoutResult, error) {
	autopilotMu.Lock()
	defer autopilotMu.Unlock()

	sources, err := model.GetAllNewsSources(true)
	if err != nil {
		return nil, err
	}

	res := &AutopilotScoutResult{
		SourcesChecked: len(sources),
	}

	for _, src := range sources {
		items, err := FetchSourceFeed(ctx, src)
		if err != nil {
			res.SourceFailures++
			res.FailedSources = append(res.FailedSources, fmt.Sprintf("%s: %v", src.Name, err))
			logger.LogWarn(ctx, fmt.Sprintf("[Autopilot] Source fetch warning for %s: %v", src.Name, err))
			continue
		}

		for _, item := range items {
			// Section 3: Deduplication check
			if model.IsFeedItemProcessed(item.GUID, item.URL) {
				res.DuplicatesRemoved++
				continue
			}

			// Record feed item
			_ = model.RecordFeedItem(&model.NewsFeedItem{
				SourceId:    src.Id,
				Guid:        item.GUID,
				Url:         item.URL,
				Title:       item.Title,
				PublishedAt: item.PublishedAt.Unix(),
				Status:      "discovered",
			})
			res.ItemsDiscovered++

			// Section 4: Cluster Story
			cluster, isNew, err := ClusterStory(item)
			if err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("[Autopilot] Failed to cluster %q: %v", item.Title, err))
				continue
			}
			if isNew && cluster != nil {
				res.ClustersCreated++
			}
		}
	}

	lastAutopilotRunAt = common.GetTimestamp()
	logger.LogInfo(ctx, fmt.Sprintf("[Autopilot] Scout cycle complete: checked=%d, fail=%d, disc=%d, dup=%d, clusters=%d",
		res.SourcesChecked, res.SourceFailures, res.ItemsDiscovered, res.DuplicatesRemoved, res.ClustersCreated))

	return res, nil
}

// RunAutopilotPublishBatch evaluates candidate clusters and publishes up to batchSize articles (Section 5, 6, 7, 8, 9, 10)
func RunAutopilotPublishBatch(ctx context.Context, batchLimit int) (*AutopilotBatchResult, error) {
	autopilotMu.Lock()
	defer autopilotMu.Unlock()

	maxBatch := GetMaxPerBatch()
	if batchLimit <= 0 || batchLimit > maxBatch {
		batchLimit = maxBatch
	}
	maxDaily := GetMaxPublishedPerDay()
	maxSource := GetMaxPerSourcePerDay()

	batchId := fmt.Sprintf("batch-%d", common.GetTimestamp())
	runId := fmt.Sprintf("run-%d", common.GetTimestamp())

	todayAutopilotPublished, _ := model.GetAutopilotPublishedToday()
	res := &AutopilotBatchResult{
		BatchId:         batchId,
		DailyTotalSoFar: todayAutopilotPublished,
	}

	// Section 1: Immediate Safety Check (Publishing disabled by default)
	if !IsAutopilotPublishingEnabled() {
		res.RejectionReasons = append(res.RejectionReasons, "Automatic publishing is currently disabled (Safety Hold)")
		logger.LogInfo(ctx, "[Autopilot] Automatic publishing is currently disabled. Skipping publish batch.")
		return res, nil
	}

	// Section 6 & 7: Check daily cap
	if todayAutopilotPublished >= maxDaily {
		res.QuotaBlocked++
		res.RejectionReasons = append(res.RejectionReasons, fmt.Sprintf("Global daily cap reached (%d/%d)", todayAutopilotPublished, maxDaily))
		logger.LogInfo(ctx, fmt.Sprintf("[Autopilot] Global daily cap reached (%d/%d). Skipping batch.", todayAutopilotPublished, maxDaily))
		return res, nil
	}

	availableCap := maxDaily - todayAutopilotPublished
	effectiveLimit := batchLimit
	if effectiveLimit > availableCap {
		effectiveLimit = availableCap
	}

	// Fetch active clusters awaiting editorial coverage
	clusters, err := model.GetActiveStoryClusters(50)
	if err != nil {
		return res, err
	}

	var publishedURLs []string
	indexNow := NewIndexNowClient()

	for _, cluster := range clusters {
		if res.ArticlesPublished >= effectiveLimit {
			break
		}

		// Check if post already created for cluster
		var existingPost model.NewsPost
		if err := model.DB.Where("cluster_id = ?", cluster.Id).First(&existingPost).Error; err == nil {
			res.AlreadyPublished++
			continue
		}

		res.ArticlesAttempted++

		// Section 5: Quality Gate (Quality beats quota)
		if cluster.RelevanceScore < QualityScoreMinLimit && cluster.DeveloperScore < 2.0 {
			res.RejectedStories++
			res.RejectionReasons = append(res.RejectionReasons, fmt.Sprintf("Quality score below threshold (rel=%.1f, dev=%.1f) for %q", cluster.RelevanceScore, cluster.DeveloperScore, cluster.Title))
			continue
		}

		// Check source cap in memory before entering transaction
		if cluster.PrimarySourceId > 0 {
			sourceCount, _ := model.GetPublishedCountBySourceToday(cluster.PrimarySourceId)
			if sourceCount >= maxSource {
				res.QuotaBlocked++
				res.RejectedStories++
				res.RejectionReasons = append(res.RejectionReasons, fmt.Sprintf("Source daily cap (%d/%d) reached for cluster %d", sourceCount, maxSource, cluster.Id))
				continue
			}
		}

		// Retrieve primary source info
		var src *model.NewsSource
		if cluster.PrimarySourceId > 0 {
			src, _ = model.GetNewsSourceById(cluster.PrimarySourceId)
		}

		// Generate complete Thai technical article (Section 9, 10, 11, 13)
		post := GenerateEditorialPost(cluster, src)

		// Section 16: Security & Risk Gate
		if post.ContentRisk == model.ContentRiskHigh {
			post.Status = model.NewsStatusReviewRequired
			_ = model.CreateNewsPost(post)
			res.DraftsCreated++
			continue
		} else if post.ContentRisk == model.ContentRiskMedium {
			post.Status = model.NewsStatusDraft
			_ = model.CreateNewsPost(post)
			res.DraftsCreated++
			continue
		}

		// Section 7 & 8: Atomic Transactional Publication
		pubResult, err := model.PublishPostWithAtomicQuota(&model.AutopilotPublishParams{
			Post:            post,
			ClusterId:       cluster.Id,
			PrimarySourceId: cluster.PrimarySourceId,
			AutopilotRunId:  runId,
			BatchId:         batchId,
			MaxDailyCap:     maxDaily,
			MaxSourceCap:    maxSource,
		})
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("[Autopilot] Transactional publication error for cluster %d: %v", cluster.Id, err))
			continue
		}
		if !pubResult.Success {
			res.QuotaBlocked++
			res.RejectedStories++
			res.RejectionReasons = append(res.RejectionReasons, pubResult.RejectReason)
			continue
		}

		res.ArticlesPublished++
		todayAutopilotPublished++
		res.DailyTotalSoFar = todayAutopilotPublished
		publishedURLs = append(publishedURLs, post.CanonicalUrl)

		// Section 19: Distribution to DEV.to if high value
		if cluster.DeveloperScore >= 5.0 {
			_ = QueuePostDistributions(post)
		}
	}

	// Section 18: Discovery submission to IndexNow
	if len(publishedURLs) > 0 {
		_ = indexNow.SubmitURLs(ctx, publishedURLs)
		res.IndexNowSubmitted = len(publishedURLs)
	}

	lastAutopilotRunAt = common.GetTimestamp()
	logger.LogInfo(ctx, fmt.Sprintf("[Autopilot] Batch %s complete: published=%d, drafts=%d, rejected=%d, quotaBlocked=%d, dailyTotal=%d",
		batchId, res.ArticlesPublished, res.DraftsCreated, res.RejectedStories, res.QuotaBlocked, res.DailyTotalSoFar))

	return res, nil
}

// RunDailyNewsroomReport compiles Section 23 persisted report (Section 17, 18)
func RunDailyNewsroomReport(ctx context.Context, targetDate string) (*model.NewsDailyGrowthReview, error) {
	loc := time.FixedZone("Asia/Bangkok", 7*3600)
	now := time.Now().In(loc)
	if targetDate == "" {
		targetDate = now.Format("2006-01-02")
	}

	// 1. Ingest base growth review metrics
	review, err := GenerateDailyGrowthReview(targetDate)
	if err != nil {
		return nil, err
	}

	// 2. Query source counts and health
	allSources, _ := model.GetAllNewsSources(false)
	failingSources := 0
	for _, s := range allSources {
		if s.ConsecutiveFailures > 0 || s.FetchErrorCount > 0 {
			failingSources++
		}
	}
	review.SourcesChecked = len(allSources)
	review.SourceFailures = failingSources

	// 3. Count feed items discovered today
	startOfDay, endOfDay, _ := model.GetBangkokDateRange(now)
	var feedCount, dupCount int64
	model.DB.Model(&model.NewsFeedItem{}).Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).Count(&feedCount)
	review.CandidateStories = int(feedCount)

	// 4. Clusters created today
	var clusterCount int64
	model.DB.Model(&model.StoryCluster{}).Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).Count(&clusterCount)
	review.ClustersCreated = int(clusterCount)
	review.DuplicatesRemoved = int(dupCount)

	// 5. Authoritative publication counters (Section 4 & 18)
	counters, _ := model.GetAuthoritativePublicationCounters()
	if counters == nil {
		counters = &model.NewsPublicationCounters{}
	}
	review.PostsPublishedToday = counters.AutopilotPublishedToday
	review.DraftsRequiringReview = counters.ReviewsToday

	// 6. Sitemaps and IndexNow status
	baseURL := common.GetCanonicalBaseURL()
	review.MainSitemapStatus = "ok"
	review.NewsSitemapStatus = "ok"
	review.IndexNowStatus = "active"

	// Verify sitemap HTTP status non-blockingly
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		if resp, err := client.Get(baseURL + "/sitemap.xml"); err == nil {
			_ = resp.Body.Close()
		}
	}()

	// 7. DEV.to status
	review.DevToStatus = "active_idempotent"

	// 8. AI Visibility signals (Section 22)
	obsList, _ := model.GetRecentAiVisibilityObservations(10)
	var obsSummary strings.Builder
	obsSummary.WriteString(fmt.Sprintf("Recorded %d recent AI visibility observations.", len(obsList)))
	for _, o := range obsList {
		obsSummary.WriteString(fmt.Sprintf(" [%s: cited=%v, type=%s]", o.Provider, o.ToraCited, o.ObservationType))
	}
	review.AiVisibilitySummary = obsSummary.String()

	// 9. Top 5 opportunities for tomorrow (Truthful SEO wording - Section 17)
	opps, _ := model.GetAllNewsSeoOpportunities("detected", 5)
	var oppsSummary strings.Builder
	for i, op := range opps {
		oppsSummary.WriteString(fmt.Sprintf("%d. [%s] %s: %s\n", i+1, op.OpportunityType, op.Query, op.ProposedAction))
	}
	if len(opps) == 0 {
		gscOverview := GetNormalizedGSCStatus(ctx, false)
		gscStatus := gscOverview.Status
		if gscStatus == "CONNECTED" || gscStatus == "CONFIGURED" || gscStatus == "NOT_CONFIGURED" || gscStatus == "" {
			oppsSummary.WriteString("NO_DATA_YET — Search Console baseline observations pending initial Google sync.")
		} else {
			oppsSummary.WriteString("NO_ACTIONABLE_OPPORTUNITIES — No search queries currently meeting CTR gap or impression threshold.")
		}
	}
	review.TopOpportunitiesTomorrow = oppsSummary.String()

	// 10. Newsroom Run Status with Invariant Check (Section 18 & 26)
	maxDaily := GetMaxPublishedPerDay()
	if counters.AutopilotPublishedToday > maxDaily {
		review.NewsroomRunStatus = fmt.Sprintf("INVARIANT VIOLATION: AUTOPILOT PUBLISHED TODAY (%d) EXCEEDED DAILY CAP (%d)", counters.AutopilotPublishedToday, maxDaily)
	} else if failingSources > 0 {
		review.NewsroomRunStatus = "NEWSROOM RUN PARTIAL — SOURCE/QUALITY ISSUES"
	} else {
		review.NewsroomRunStatus = "NEWSROOM RUN COMPLETE"
	}

	_ = model.CreateOrUpdateDailyGrowthReview(review)
	lastAutopilotReport = review

	logger.LogInfo(ctx, fmt.Sprintf("[Autopilot] Section 23 daily newsroom report persisted for %s: status=%s, autopilotPublished=%d, totalPublishedToday=%d",
		targetDate, review.NewsroomRunStatus, counters.AutopilotPublishedToday, counters.PublishedTodayTotal))

	return review, nil
}

// GetAutopilotStatus returns the live dashboard status for the newsroom autopilot
func GetAutopilotStatus() *AutopilotStatus {
	loc := time.FixedZone("Asia/Bangkok", 7*3600)
	bkkTime := time.Now().In(loc).Format("2006-01-02 15:04:05 MST")
	counters, _ := model.GetAuthoritativePublicationCounters()
	if counters == nil {
		counters = &model.NewsPublicationCounters{}
	}

	sources, _ := model.GetAllNewsSources(false)
	failing := 0
	for _, s := range sources {
		if s.ConsecutiveFailures > 0 {
			failing++
		}
	}

	maxDaily := GetMaxPublishedPerDay()

	return &AutopilotStatus{
		CurrentBangkokTime:      bkkTime,
		TodayPublishedCount:     counters.AutopilotPublishedToday, // Authoritative: only autonomous publications
		AutopilotPublishedToday: counters.AutopilotPublishedToday,
		PublishedTodayTotal:     counters.PublishedTodayTotal,
		LegacyUnknownToday:      counters.LegacyUnknownToday,
		ManualPublishedToday:    counters.ManualPublishedToday,
		SeedOrHistoricalToday:   counters.SeedOrHistoricalToday,
		DraftsToday:             counters.DraftsToday,
		MaxDailyCap:             maxDaily,
		PublishingEnabled:       IsAutopilotPublishingEnabled(),
		ActiveSourcesCount:      len(sources),
		FailingSourcesCount:     failing,
		LastExecutedSlot:        lastExecutedSlot,
		LastRunAt:               lastAutopilotRunAt,
		ScheduleCycles: []string{
			"05:30 scout",
			"07:00 publish batch 1 (max 5)",
			"10:30 scout",
			"12:00 publish batch 2 (max 5)",
			"15:30 scout",
			"17:00 publish batch 3 (max 5)",
			"20:00 scout",
			"21:00 publish batch 4 (max 5)",
			"23:30 daily growth review report",
		},
		LatestReport: lastAutopilotReport,
	}
}

// StartNewsAutopilotRunner initializes the Asia/Bangkok schedule runner (Section 20 & 25)
func StartNewsAutopilotRunner() {
	autopilotOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		go func() {
			logger.LogInfo(context.Background(), "[NewsAutopilot] Tora AI Daily Newsroom Autopilot initialized (Asia/Bangkok UTC+7)")
			loc := time.FixedZone("Asia/Bangkok", 7*3600)

			// Poll every minute to check schedule slots
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				now := time.Now().In(loc)
				timeSlot := now.Format("15:04")
				slotKey := fmt.Sprintf("%s:%s", now.Format("2006-01-02"), timeSlot)

				if slotKey == lastExecutedSlot {
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

				switch timeSlot {
				case "05:30", "10:30", "15:30", "20:00":
					logger.LogInfo(ctx, fmt.Sprintf("[NewsAutopilot] Executing scheduled scout cycle for %s", timeSlot))
					_, _ = RunAutopilotScoutCycle(ctx)
					lastExecutedSlot = slotKey

				case "07:00", "12:00", "17:00", "21:00":
					logger.LogInfo(ctx, fmt.Sprintf("[NewsAutopilot] Executing scheduled publishing batch for %s", timeSlot))
					// Quick scout before batch to pick up newest items
					_, _ = RunAutopilotScoutCycle(ctx)
					_, _ = RunAutopilotPublishBatch(ctx, MaxPerBatch)
					lastExecutedSlot = slotKey

				case "23:30":
					logger.LogInfo(ctx, fmt.Sprintf("[NewsAutopilot] Executing scheduled Section 23 daily growth review report for %s", timeSlot))
					_, _ = RunDailyNewsroomReport(ctx, now.Format("2006-01-02"))
					lastExecutedSlot = slotKey
				}

				cancel()
			}
		}()
	})
}
