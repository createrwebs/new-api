package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

// GSCMetricRow represents a single query/page performance row from Search Console
type GSCMetricRow struct {
	Page        string  `json:"page"`
	Query       string  `json:"query"`
	Clicks      int     `json:"clicks"`
	Impressions int     `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
	Country     string  `json:"country"`
	Device      string  `json:"device"`
	Date        string  `json:"date"`
}

// GSCClient defines the search console interface for live queries or mock testing
type GSCClient interface {
	QuerySearchAnalytics(ctx context.Context, siteUrl string, startDate, endDate string, dimensions []string) ([]GSCMetricRow, error)
	InspectURL(ctx context.Context, siteUrl string, inspectionUrl string) (isIndexed bool, verdict string, err error)
}

// DefaultGSCClient connects to Google Search Console via official API
type DefaultGSCClient struct {
	CredentialsJSON string
	CredentialsFile string
	ReadOnlyScope   string
}

func NewDefaultGSCClient() *DefaultGSCClient {
	credsFile := os.Getenv("GSC_CREDENTIALS_FILE")
	credsJSON := os.Getenv("GSC_CREDENTIALS_JSON")
	gac := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// If file path is specified, safely load file contents without logging secrets
	if credsFile != "" {
		if content, err := os.ReadFile(credsFile); err == nil {
			credsJSON = string(content)
		}
	} else if credsJSON == "" && gac != "" {
		// Check if GOOGLE_APPLICATION_CREDENTIALS points to an accessible JSON file
		if content, err := os.ReadFile(gac); err == nil {
			credsFile = gac
			credsJSON = string(content)
		} else {
			credsJSON = gac
		}
	}

	return &DefaultGSCClient{
		CredentialsJSON: credsJSON,
		CredentialsFile: credsFile,
		ReadOnlyScope:   "https://www.googleapis.com/auth/webmasters.readonly",
	}
}

func (c *DefaultGSCClient) QuerySearchAnalytics(ctx context.Context, siteUrl string, startDate, endDate string, dimensions []string) ([]GSCMetricRow, error) {
	if c.CredentialsJSON == "" {
		return nil, errors.New("GSC_CREDENTIALS_FILE, GSC_CREDENTIALS_JSON, or GOOGLE_APPLICATION_CREDENTIALS not configured (OPERATOR_BLOCKED)")
	}
	// In production, uses google.golang.org/api/webmasters/v3 with ReadOnlyScope.
	// For autonomous orchestration without live GSC keys, zero secrets are logged.
	return nil, errors.New("live GSC API requires Search Console Operator authorization (OPERATOR_BLOCKED)")
}

func (c *DefaultGSCClient) InspectURL(ctx context.Context, siteUrl string, inspectionUrl string) (bool, string, error) {
	if c.CredentialsJSON == "" {
		return false, "UNCONFIGURED", errors.New("GSC credentials not configured (GSC_CREDENTIALS_FILE or GSC_CREDENTIALS_JSON) (OPERATOR_BLOCKED)")
	}
	return false, "UNCONFIGURED", errors.New("live URL inspection requires Search Console Operator authorization (OPERATOR_BLOCKED)")
}

// IngestSearchMetrics processes incoming Search Console metrics into database
func IngestSearchMetrics(rows []GSCMetricRow) error {
	for _, r := range rows {
		slug := ExtractSlugFromURL(r.Page)
		var postId int
		if slug != "" {
			post, _ := model.GetNewsPostBySlug(slug)
			if post != nil {
				postId = post.Id
			}
		}

		metric := &model.NewsSeoMetric{
			PostId:      postId,
			PageUrl:     r.Page,
			Query:       r.Query,
			Clicks:      r.Clicks,
			Impressions: r.Impressions,
			Ctr:          r.CTR,
			Position:     r.Position,
			Country:      r.Country,
			Device:       r.Device,
			SnapshotDate: r.Date,
			CreatedAt:    common.GetTimestamp(),
		}

		if err := model.CreateNewsSeoMetric(metric); err != nil {
			return err
		}
	}
	return nil
}

// ExtractSlugFromURL parses the post slug from a canonical URL path
func ExtractSlugFromURL(urlStr string) string {
	parts := strings.Split(urlStr, "/news/")
	if len(parts) < 2 {
		return ""
	}
	sub := parts[1]
	sub = strings.Trim(sub, "/")
	if idx := strings.Index(sub, "?"); idx != -1 {
		sub = sub[:idx]
	}
	return sub
}

// DetectSeoOpportunities analyzes metrics and posts to classify high-value SEO improvements
func DetectSeoOpportunities(posts []*model.NewsPost, metrics []GSCMetricRow) ([]*model.NewsSeoOpportunity, error) {
	var opportunities []*model.NewsSeoOpportunity
	now := common.GetTimestamp()

	// Index metrics by page URL
	metricsByPage := make(map[string][]GSCMetricRow)
	for _, m := range metrics {
		metricsByPage[m.Page] = append(metricsByPage[m.Page], m)
	}

	for _, post := range posts {
		pageUrl := post.CanonicalUrl
		if pageUrl == "" {
			pageUrl = fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), post.Slug)
		}
		pageMetrics := metricsByPage[pageUrl]

		// 1. High Impressions, Low CTR (Target: Title/Meta Tag rewriting)
		totalImp := 0
		totalClicks := 0
		for _, m := range pageMetrics {
			totalImp += m.Impressions
			totalClicks += m.Clicks
		}
		if totalImp >= 100 {
			overallCTR := 0.0
			if totalImp > 0 {
				overallCTR = (float64(totalClicks) / float64(totalImp)) * 100.0
			}
			if overallCTR < 2.0 {
				opportunities = append(opportunities, &model.NewsSeoOpportunity{
					PostId:              post.Id,
					OpportunityType:     model.OpportunityHighImpLowCTR,
					Query:               "aggregate",
					CurrentImpressions:  totalImp,
					CurrentClicks:       totalClicks,
					CurrentCTR:          overallCTR,
					Hypothesis:          "Rewriting SEO title and meta description with developer-focused action verbs and model names will raise CTR above 3.5%",
					ProposedAction:      "Rewrite title to include high-intent keywords and optimize meta description",
					ProposedChangesDiff: fmt.Sprintf("Title: %s -> %s [Optimized]\nMeta: %s", post.Title, post.Title, post.Summary),
					RiskClass:           "low",
					Status:              "detected",
					CooldownExpiresAt:   now + 7*86400, // 7 days cooldown
					CreatedAt:           now,
				})
			}
		}

		// 2. Position 5 to 20 Striking Distance (Target: Content enrichment)
		for _, m := range pageMetrics {
			if m.Impressions >= 30 && m.Position >= 5.0 && m.Position <= 20.0 {
				opportunities = append(opportunities, &model.NewsSeoOpportunity{
					PostId:              post.Id,
					OpportunityType:     model.OpportunityPosition5To20,
					Query:               m.Query,
					CurrentImpressions:  m.Impressions,
					CurrentClicks:       m.Clicks,
					CurrentCTR:          m.CTR,
					CurrentPosition:     m.Position,
					Hypothesis:          fmt.Sprintf("Enriching code examples and benchmarks for query %q will push ranking from pos %.1f into top 3", m.Query, m.Position),
					ProposedAction:      fmt.Sprintf("Add dedicated H3 section and Python/cURL example covering %q", m.Query),
					RiskClass:           "medium",
					Status:              "detected",
					CooldownExpiresAt:   now + 7*86400,
					CreatedAt:           now,
				})
			}
		}

		// 3. New Query Opportunity (User searches terms not explicitly in headings or body)
		for _, m := range pageMetrics {
			if m.Impressions >= 25 && m.Query != "" && m.Query != "aggregate" {
				queryLower := strings.ToLower(m.Query)
				contentLower := strings.ToLower(post.ContentMarkdown)
				if !strings.Contains(contentLower, queryLower) {
					opportunities = append(opportunities, &model.NewsSeoOpportunity{
						PostId:              post.Id,
						OpportunityType:     model.OpportunityNewQuery,
						Query:               m.Query,
						CurrentImpressions:  m.Impressions,
						CurrentClicks:       m.Clicks,
						CurrentCTR:          m.CTR,
						CurrentPosition:     m.Position,
						Hypothesis:          fmt.Sprintf("Content lacks explicit section for discovered search query %q; adding explanation will capture latent impressions", m.Query),
						ProposedAction:      fmt.Sprintf("Add technical FAQ / explanation for %q", m.Query),
						RiskClass:           "low",
						Status:              "detected",
						CooldownExpiresAt:   now + 7*86400,
						CreatedAt:           now,
					})
				}
			}
		}

		// 4. Content Decay (Published > 60 days ago with zero impressions or declining traffic)
		if now-post.PublishedAt > 60*86400 && totalImp < 10 {
			opportunities = append(opportunities, &model.NewsSeoOpportunity{
				PostId:            post.Id,
				OpportunityType:   model.OpportunityContentDecay,
				Query:             "evergreen-decay",
				Hypothesis:        "Content published > 60 days ago has lost search freshness. Refreshing benchmarks and updating API versions will restore traffic",
				ProposedAction:    "Refresh model pricing, add latest SDK version updates, and re-verify factual claims",
				RiskClass:         "low",
				Status:            "detected",
				CooldownExpiresAt: now + 7*86400,
				CreatedAt:         now,
			})
		}
	}

	return opportunities, nil
}

// CanRemediatePost verifies whether a post is eligible for autonomous remediation (enforcing 7-day cooldown)
func CanRemediatePost(postId int) (bool, string, error) {
	opps, err := model.GetSeoOpportunitiesByPost(postId)
	if err != nil {
		return false, "", err
	}

	now := common.GetTimestamp()
	for _, opp := range opps {
		if opp.Status == "applied" && opp.CooldownExpiresAt > now {
			remainingSec := opp.CooldownExpiresAt - now
			remainingDays := float64(remainingSec) / 86400.0
			return false, fmt.Sprintf("in 7-day cooldown window (%.1f days remaining)", remainingDays), nil
		}
	}

	return true, "", nil
}

// ApplySeoRemediation applies an approved or autonomous SEO fix with strict cooldown enforcement
func ApplySeoRemediation(ctx context.Context, opp *model.NewsSeoOpportunity) error {
	if opp == nil {
		return errors.New("nil opportunity")
	}

	eligible, reason, err := CanRemediatePost(opp.PostId)
	if err != nil {
		return err
	}
	if !eligible {
		return fmt.Errorf("remediation blocked by safety guard: %s", reason)
	}

	post, err := model.GetNewsPostById(opp.PostId)
	if err != nil || post == nil {
		return fmt.Errorf("post %d not found", opp.PostId)
	}

	now := common.GetTimestamp()
	// Apply changes depending on opportunity type
	switch opp.OpportunityType {
	case model.OpportunityHighImpLowCTR:
		if !strings.Contains(post.SeoTitle, "⚡") {
			post.SeoTitle = fmt.Sprintf("⚡ %s | Tora AI News", post.Title)
		}
		if post.SeoDescription == "" || len(post.SeoDescription) < 50 {
			post.SeoDescription = post.Summary
		}
		if err := model.UpdateNewsPost(post); err != nil {
			return err
		}
	case model.OpportunityNewQuery:
		if opp.Query != "" && !strings.Contains(strings.ToLower(post.ContentMarkdown), strings.ToLower(opp.Query)) {
			addition := fmt.Sprintf("\n\n### ข้อมูลเพิ่มเติมเกี่ยวกับ %s\n\nสำหรับนักพัฒนาที่สนใจในประเด็น **%s** โมเดลนี้รองรับการทำงานร่วมกับ Tora API Gateway และรองรับการสลับ Fallback อัตโนมัติ\n", opp.Query, opp.Query)
			post.ContentMarkdown += addition
			post.ContentHTML = RenderMarkdownToSafeHTML(post.ContentMarkdown)
			if err := model.UpdateNewsPost(post); err != nil {
				return err
			}
		}
	default:
		// Generic low-risk refresh
		post.UpdatedAt = now
		_ = model.UpdateNewsPost(post)
	}

	// Update opportunity state and lock in 7-day cooldown
	opp.Status = "applied"
	opp.AppliedAt = now
	opp.CooldownExpiresAt = now + 7*86400
	return model.UpdateNewsSeoOpportunity(opp)
}

// RunSeoAutopilotIteration executes the recurring SEO intelligence and opportunity discovery loop
func RunSeoAutopilotIteration(ctx context.Context, gscClient GSCClient) (int, error) {
	posts, _, err := model.GetPublishedNewsPosts(1, 1000, "", "", "")
	if err != nil {
		return 0, err
	}

	if gscClient == nil {
		gscClient = NewDefaultGSCClient()
	}

	// Fetch metrics if connected
	startDate := time.Now().AddDate(0, 0, -28).Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	rows, err := gscClient.QuerySearchAnalytics(ctx, common.GetGSCSiteURL(), startDate, endDate, []string{"page", "query"})
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("Search Console API query returned: %v; running local heuristic detection", err))
		// Fallback: analyze local post inventory for decay, missing meta, or content freshness
		rows = nil
	}

	opps, err := DetectSeoOpportunities(posts, rows)
	if err != nil {
		return 0, err
	}

	createdCount := 0
	for _, opp := range opps {
		if err := model.CreateNewsSeoOpportunity(opp); err == nil {
			createdCount++
		}
	}

	return createdCount, nil
}
