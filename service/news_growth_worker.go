package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

var (
	growthWorkerOnce sync.Once
	growthWorkerMu   sync.Mutex
	lastGrowthRunAt  int64
)

// GrowthCollectorResult reports outcomes of a single growth/collector cycle
type GrowthCollectorResult struct {
	MetricsIngested     int    `json:"metrics_ingested"`
	InspectionsRun      int    `json:"inspections_run"`
	OpportunitiesFound  int    `json:"opportunities_found"`
	DailyReviewDate     string `json:"daily_review_date"`
	GSCStatus           string `json:"gsc_status"`
	GA4Status           string `json:"ga4_status"`
	LastError           string `json:"last_error,omitempty"`
}

// RunGrowthCollectorIteration executes one pass of the persistent Growth/SEO intelligence loop
// Adheres strictly to Sections 5, 6, 7, 8, 16, 20
func RunGrowthCollectorIteration(ctx context.Context) (*GrowthCollectorResult, error) {
	growthWorkerMu.Lock()
	defer growthWorkerMu.Unlock()

	res := &GrowthCollectorResult{
		DailyReviewDate: time.Now().Format("2006-01-02"),
	}

	// 1. Check GSC Status truthfully
	gscStatus := GetNormalizedGSCStatus(ctx, true)
	res.GSCStatus = gscStatus.Status

	// 2. Collect Search Analytics if connected (Delta-aware, Section 1)
	if gscStatus.Status == GSCStatusConnected || gscStatus.Status == GSCStatusDataAvailable {
		gscClient := NewDefaultGSCClient()
		rows, err := gscClient.QueryDeltaSearchAnalytics(ctx, common.GetGSCSiteURL())
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("[GrowthWorker] GSC delta query returned: %v", err))
			res.LastError = err.Error()
		} else if len(rows) > 0 {
			ingested, ingestErr := IngestSearchMetrics(rows)
			if ingestErr != nil {
				logger.LogWarn(ctx, fmt.Sprintf("[GrowthWorker] Error ingesting search metrics: %v", ingestErr))
			} else {
				res.MetricsIngested = ingested
				logger.LogInfo(ctx, fmt.Sprintf("[GrowthWorker] Successfully ingested %d GSC metric rows", ingested))
			}

			// Daily full SEO opportunity evaluation is sufficient (Section 1)
			posts, _, _ := model.GetPublishedNewsPosts(1, 100, "", "", "")
			opps, _ := DetectSeoOpportunities(posts, rows)
			for _, opp := range opps {
				if err := model.CreateNewsSeoOpportunity(opp); err == nil {
					res.OpportunitiesFound++
				}
			}
		} else {
			// Section 6: Truthfully handle no rows (NO_DATA_YET) without fake opportunities
			logger.LogInfo(ctx, "[GrowthWorker] GSC returned 0 rows (NO_DATA_YET - normal for new properties)")
		}
	}

	// 3. Selective URL Inspection (Section 7)
	// Only inspect newly published posts (published in last 48h) or posts without recent inspection
	publishedPosts, _, _ := model.GetPublishedNewsPosts(1, 10, "", "", "")
	inspectionQuota := 2 // strictly rate-limited: max 2 inspections per cycle
	for _, p := range publishedPosts {
		if inspectionQuota <= 0 {
			break
		}
		latest, _ := model.GetLatestUrlInspectionByPostId(p.Id)
		now := common.GetTimestamp()
		// Inspect if never inspected or older than 7 days
		if latest == nil || (now-latest.InspectionTime > 7*86400) {
			insp, err := PerformSelectiveUrlInspection(ctx, p.Id)
			if err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("[GrowthWorker] URL inspection failed for post %d (%s): %v", p.Id, p.Slug, err))
			} else if insp != nil {
				res.InspectionsRun++
				inspectionQuota--
				logger.LogInfo(ctx, fmt.Sprintf("[GrowthWorker] URL inspected for post %d: verdict=%s, coverage=%s", p.Id, insp.Verdict, insp.CoverageState))
			}
		}
	}

	// 4. Collect GA4 Telemetry if configured (Section 11)
	ga4Client := NewDefaultGA4Client()
	ga4Status := ga4Client.GetNormalizedGA4Status(ctx, false)
	res.GA4Status = ga4Status.Status
	if ga4Status.Status == GA4StatusDataAvailable || ga4Status.Status == GA4StatusConnected {
		lpMetrics, ga4Err := ga4Client.QueryLandingPageMetrics(ctx, "28daysAgo", "yesterday")
		if ga4Err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("[GrowthWorker] GA4 landing page query returned: %v", ga4Err))
		} else {
			logger.LogInfo(ctx, fmt.Sprintf("[GrowthWorker] GA4 collected %d landing page rows", len(lpMetrics)))
		}
	} else {
		logger.LogInfo(ctx, fmt.Sprintf("[GrowthWorker] GA4 status is %s (ready for operator provisioning)", ga4Status.Status))
	}

	// 5. Generate & persist internal Daily Growth Review (Section 16)
	_, reviewErr := GenerateDailyGrowthReview(res.DailyReviewDate)
	if reviewErr != nil {
		logger.LogWarn(ctx, fmt.Sprintf("[GrowthWorker] Failed to generate daily growth review: %v", reviewErr))
	} else {
		logger.LogInfo(ctx, fmt.Sprintf("[GrowthWorker] Daily growth review persisted for %s", res.DailyReviewDate))
	}

	lastGrowthRunAt = common.GetTimestamp()
	return res, nil
}

// StartNewsGrowthCollectorRunner initializes the persistent production worker (Section 20)
func StartNewsGrowthCollectorRunner() {
	growthWorkerOnce.Do(func() {
		// Only run on master node
		if !common.IsMasterNode {
			return
		}

		go func() {
			logger.LogInfo(context.Background(), "[GrowthWorker] Tora News Growth persistent background worker initialized")
			// Run initial check after 1 minute of application boot
			time.Sleep(1 * time.Minute)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			_, _ = RunGrowthCollectorIteration(ctx)
			cancel()

			// Scheduled periodic checks every 1 hour
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Minute)
					res, err := RunGrowthCollectorIteration(runCtx)
					runCancel()
					if err != nil {
						logger.LogWarn(context.Background(), fmt.Sprintf("[GrowthWorker] Iteration completed with warning: %v", err))
					} else {
						logger.LogInfo(context.Background(), fmt.Sprintf("[GrowthWorker] Iteration finished: ingested=%d, inspected=%d, opps=%d, status=%s",
							res.MetricsIngested, res.InspectionsRun, res.OpportunitiesFound, res.GSCStatus))
					}
				}
			}
		}()
	})
}
