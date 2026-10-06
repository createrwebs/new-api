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
	newsScoutRunnerOnce sync.Once
	newsSyncMutex       sync.Mutex
)

// SyncResult summarizes the outcome of a news ingestion pass
type SyncResult struct {
	SourcesProcessed int `json:"sources_processed"`
	ItemsDiscovered  int `json:"items_discovered"`
	ClustersCreated  int `json:"clusters_created"`
	PostsDrafted     int `json:"posts_drafted"`
	Errors           int `json:"errors"`
}

// SyncSingleNewsSource ingests, clusters, and creates editorial drafts for a single source
func SyncSingleNewsSource(ctx context.Context, src *model.NewsSource) (int, error) {
	if src == nil {
		return 0, fmt.Errorf("nil news source")
	}

	items, err := FetchSourceFeed(ctx, src)
	if err != nil {
		return 0, err
	}

	newPostCount := 0
	for _, item := range items {
		cluster, isNew, err := ClusterStory(item)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("failed to cluster story %q: %v", item.Title, err))
			continue
		}

		// Only generate editorial content for new clusters with sufficient developer relevance
		if isNew && (cluster.DeveloperScore >= 1.0 || cluster.RelevanceScore >= 1.0) {
			post := GenerateEditorialPost(cluster, src)
			if err := model.CreateNewsPost(post); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("failed to save news post for cluster %d: %v", cluster.Id, err))
				continue
			}

			// Generate and queue social distributions
			_ = QueuePostDistributions(post)
			newPostCount++
		}
	}

	return newPostCount, nil
}

// RunNewsScoutSync runs a full ingestion pass across all enabled sources
func RunNewsScoutSync(ctx context.Context) (*SyncResult, error) {
	newsSyncMutex.Lock()
	defer newsSyncMutex.Unlock()

	sources, err := model.GetAllNewsSources(true)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{
		SourcesProcessed: len(sources),
	}

	now := common.GetTimestamp()
	for _, src := range sources {
		// Respect source polling interval
		intervalSec := int64(src.PollingIntervalMinutes * 60)
		if intervalSec <= 0 {
			intervalSec = 1800 // default 30 min
		}
		if src.LastFetchedAt > 0 && (now-src.LastFetchedAt) < intervalSec {
			continue
		}

		count, err := SyncSingleNewsSource(ctx, src)
		if err != nil {
			result.Errors++
			logger.LogWarn(ctx, fmt.Sprintf("news scout sync failed for source %s: %v", src.Name, err))
		} else {
			result.PostsDrafted += count
		}
	}

	return result, nil
}

// StartNewsScoutRunner starts a background worker that polls feeds periodically
func StartNewsScoutRunner() {
	newsScoutRunnerOnce.Do(func() {
		// Only run automatic background scraping on master node
		if !common.IsMasterNode {
			return
		}

		go func() {
			logger.LogInfo(context.Background(), "Tora News Scout background runner initialized")
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					res, err := RunNewsScoutSync(ctx)
					cancel()
					if err != nil {
						logger.LogWarn(context.Background(), fmt.Sprintf("periodic news sync error: %v", err))
					} else if res != nil && res.PostsDrafted > 0 {
						logger.LogInfo(context.Background(), fmt.Sprintf("news scout synced: %d new posts drafted", res.PostsDrafted))
					}
				}
			}
		}()
	})
}
