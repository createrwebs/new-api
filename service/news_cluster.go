package service

import (
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// CalculateSimilarity returns the Jaccard similarity [0.0, 1.0] between two titles
func CalculateSimilarity(title1, title2 string) float64 {
	tokens1 := tokenizeTitle(title1)
	tokens2 := tokenizeTitle(title2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := make(map[string]struct{}, len(tokens1))
	for _, t := range tokens1 {
		set1[t] = struct{}{}
	}

	intersection := 0
	set2 := make(map[string]struct{}, len(tokens2))
	for _, t := range tokens2 {
		set2[t] = struct{}{}
		if _, exists := set1[t]; exists {
			intersection++
		}
	}

	union := len(set1)
	for t := range set2 {
		if _, exists := set1[t]; !exists {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// ClusterStory evaluates an incoming RawStoryItem against recent active clusters.
// It returns an existing cluster if matched, or creates and returns a new cluster.
func ClusterStory(item *RawStoryItem) (*model.StoryCluster, bool, error) {
	// Look up active clusters within recent time window (last 5 days)
	recentClusters, err := model.GetActiveStoryClusters(100)
	if err != nil {
		return nil, false, err
	}

	similarityThreshold := 0.38

	var bestCluster *model.StoryCluster
	bestScore := 0.0

	for _, c := range recentClusters {
		score := CalculateSimilarity(item.Title, c.Title)
		if score > bestScore && score >= similarityThreshold {
			bestScore = score
			bestCluster = c
		}
	}

	// Found matching cluster: Corroborate and update
	if bestCluster != nil {
		bestCluster.LastEventAt = common.GetTimestamp()
		// Update primary source if this new source has higher trust tier
		if isHigherTrust(item.TrustTier, bestCluster.Category) {
			bestCluster.PrimarySourceId = item.SourceId
			bestCluster.PrimaryUrl = item.URL
		}
		_ = model.DB.Save(bestCluster)
		return bestCluster, false, nil
	}

	// New story cluster required
	relScore, devScore, thaiScore := ScoreStory(item)
	newCluster := &model.StoryCluster{
		Title:           item.Title,
		Summary:         item.Summary,
		PrimarySourceId: item.SourceId,
		PrimaryUrl:      item.URL,
		Category:        item.Category,
		Tags:            strings.Join(item.Tags, ","),
		RelevanceScore:  relScore,
		DeveloperScore:  devScore,
		ThailandScore:   thaiScore,
		Status:          "discovered",
		FirstSeenAt:     common.GetTimestamp(),
		LastEventAt:     common.GetTimestamp(),
	}

	if err := model.CreateStoryCluster(newCluster); err != nil {
		return nil, false, err
	}

	return newCluster, true, nil
}

func tokenizeTitle(text string) []string {
	var words []string
	clean := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return ' '
		}
		return unicode.ToLower(r)
	}, text)

	stopWords := map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "in": {}, "on": {},
		"at": {}, "to": {}, "for": {}, "of": {}, "with": {}, "by": {}, "from": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "new": {}, "now": {},
		"ประกาศ": {}, "เปิดตัว": {}, "พร้อม": {}, "และ": {}, "ใน": {},
	}

	for _, w := range strings.Fields(clean) {
		if len(w) <= 1 {
			continue
		}
		if _, isStop := stopWords[w]; !isStop {
			words = append(words, w)
		}
	}
	return words
}

func isHigherTrust(newTier, currentTier string) bool {
	tiers := map[string]int{
		"tier_1_official":      3,
		"tier_2_authoritative": 2,
		"tier_3_community":     1,
	}
	return tiers[newTier] > tiers[currentTier]
}
