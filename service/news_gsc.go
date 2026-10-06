package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"golang.org/x/oauth2/jwt"
)

// Normalized GSC status definitions (Section 2)
const (
	GSCStatusNotConfigured = "NOT_CONFIGURED"
	GSCStatusConfigured    = "CONFIGURED"
	GSCStatusConnected     = "CONNECTED"
	GSCStatusDataAvailable = "DATA_AVAILABLE"
	GSCStatusError         = "ERROR"
)

type GSCStatusOverview struct {
	Status          string `json:"status"` // NOT_CONFIGURED, CONFIGURED, CONNECTED, DATA_AVAILABLE, ERROR
	SiteURL         string `json:"site_url"`
	PermissionLevel string `json:"permission_level"`
	RowCount        int64  `json:"row_count"`
	DataAvailable   bool   `json:"data_available"`
	LastSyncAt      int64  `json:"last_sync_at"`
	LastError       string `json:"last_error,omitempty"`
}

var (
	cachedGSCStatus GSCStatusOverview
	cachedGSCMu     sync.RWMutex
	lastGSCCheckAt  int64
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

// GSCSitemap represents sitemap state in Search Console
type GSCSitemap struct {
	Path           string    `json:"path"`
	LastSubmitted  time.Time `json:"last_submitted"`
	IsPending      bool      `json:"is_pending"`
	IsSitemapsList bool      `json:"is_sitemaps_list"`
	LastDownloaded time.Time `json:"last_downloaded"`
	Warnings       int64     `json:"warnings"`
	Errors         int64     `json:"errors"`
}

// GSCClient defines the search console interface for live queries or mock testing
type GSCClient interface {
	VerifySiteAccess(ctx context.Context, siteUrl string) (string, error)
	QuerySearchAnalytics(ctx context.Context, siteUrl string, startDate, endDate string, dimensions []string) ([]GSCMetricRow, error)
	InspectURL(ctx context.Context, siteUrl string, inspectionUrl string) (isIndexed bool, verdict string, err error)
	InspectURLDetails(ctx context.Context, siteUrl string, inspectionUrl string) (*model.NewsUrlInspection, error)
	GetGSCSitemaps(ctx context.Context, siteUrl string) ([]GSCSitemap, error)
}

// DefaultGSCClient connects to Google Search Console via official API
type DefaultGSCClient struct {
	CredentialsJSON string
	CredentialsFile string
	ReadOnlyScope   string
}

func NewDefaultGSCClient() *DefaultGSCClient {
	credsFile := common.GetGSCCredentialsFile()
	credsJSON := common.GetGSCCredentialsJSON()
	gac := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// If file path is specified, safely load file contents without logging secrets
	if credsFile != "" {
		if content, err := os.ReadFile(credsFile); err == nil {
			credsJSON = string(content)
		}
	} else if credsJSON == "" && gac != "" {
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

func (c *DefaultGSCClient) getAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	if c.CredentialsJSON == "" {
		return nil, errors.New("GSC_CREDENTIALS_FILE, GSC_CREDENTIALS_JSON, or GOOGLE_APPLICATION_CREDENTIALS not configured (OPERATOR_BLOCKED)")
	}
	var key struct {
		PrivateKeyID string `json:"private_key_id"`
		PrivateKey   string `json:"private_key"`
		ClientEmail  string `json:"client_email"`
		TokenURI     string `json:"token_uri"`
	}
	if err := json.Unmarshal([]byte(c.CredentialsJSON), &key); err != nil {
		return nil, fmt.Errorf("failed to parse google credentials json: %w", err)
	}
	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, errors.New("invalid google service account json: missing client_email or private_key (OPERATOR_BLOCKED)")
	}
	if key.TokenURI == "" {
		key.TokenURI = "https://oauth2.googleapis.com/token"
	}
	jwtConf := &jwt.Config{
		Email:        key.ClientEmail,
		PrivateKey:   []byte(key.PrivateKey),
		PrivateKeyID: key.PrivateKeyID,
		Scopes:       []string{c.ReadOnlyScope},
		TokenURL:     key.TokenURI,
	}
	return jwtConf.Client(ctx), nil
}

// VerifySiteAccess checks property verification and permissions on GSC
func (c *DefaultGSCClient) VerifySiteAccess(ctx context.Context, siteUrl string) (string, error) {
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return "", err
	}
	if siteUrl == "" {
		siteUrl = common.GetGSCSiteURL()
	}
	escapedSite := url.QueryEscape(siteUrl)
	apiURL := fmt.Sprintf("https://www.googleapis.com/webmasters/v3/sites/%s", escapedSite)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GSC site verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GSC site verification failed HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		SiteUrl         string `json:"siteUrl"`
		PermissionLevel string `json:"permissionLevel"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.PermissionLevel, nil
}

func (c *DefaultGSCClient) QuerySearchAnalytics(ctx context.Context, siteUrl string, startDate, endDate string, dimensions []string) ([]GSCMetricRow, error) {
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if siteUrl == "" {
		siteUrl = common.GetGSCSiteURL()
	}
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	}
	if len(dimensions) == 0 {
		dimensions = []string{"page", "query"}
	}

	escapedSite := url.QueryEscape(siteUrl)
	apiURL := fmt.Sprintf("https://www.googleapis.com/webmasters/v3/sites/%s/searchAnalytics/query", escapedSite)

	reqPayload := map[string]interface{}{
		"startDate":  startDate,
		"endDate":    endDate,
		"dimensions": dimensions,
		"rowLimit":   5000,
	}
	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GSC API query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GSC API error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Rows []struct {
			Keys        []string `json:"keys"`
			Clicks      int      `json:"clicks"`
			Impressions int      `json:"impressions"`
			CTR         float64  `json:"ctr"`
			Position    float64  `json:"position"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode GSC response: %w", err)
	}

	var results []GSCMetricRow
	for _, row := range res.Rows {
		m := GSCMetricRow{
			Clicks:      row.Clicks,
			Impressions: row.Impressions,
			CTR:         row.CTR,
			Position:    row.Position,
			Date:        endDate,
		}
		if len(row.Keys) > 0 {
			m.Page = row.Keys[0]
		}
		if len(row.Keys) > 1 {
			m.Query = row.Keys[1]
		}
		results = append(results, m)
	}
	return results, nil
}

// InspectURLDetails queries Google's URL Inspection API and returns a structured model
func (c *DefaultGSCClient) InspectURLDetails(ctx context.Context, siteUrl string, inspectionUrl string) (*model.NewsUrlInspection, error) {
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if siteUrl == "" {
		siteUrl = common.GetGSCSiteURL()
	}
	apiURL := "https://searchconsole.googleapis.com/v1/urlInspection/index:inspect"
	reqPayload := map[string]interface{}{
		"siteUrl":       siteUrl,
		"inspectionUrl": inspectionUrl,
	}
	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GSC inspection HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		InspectionResult struct {
			IndexStatusResult struct {
				Verdict         string `json:"verdict"`
				CoverageState   string `json:"coverageState"`
				RobotsTxtState  string `json:"robotsTxtState"`
				IndexingState   string `json:"indexingState"`
				LastCrawlTime   string `json:"lastCrawlTime"`
				PageFetchState  string `json:"pageFetchState"`
				GoogleCanonical string `json:"googleCanonical"`
				UserCanonical   string `json:"userCanonical"`
			} `json:"indexStatusResult"`
		} `json:"inspectionResult"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode GSC inspection response: %w", err)
	}

	idx := res.InspectionResult.IndexStatusResult
	now := common.GetTimestamp()
	insp := &model.NewsUrlInspection{
		InspectionUrl:   inspectionUrl,
		Verdict:         idx.Verdict,
		CoverageState:   idx.CoverageState,
		RobotsTxtState:  idx.RobotsTxtState,
		IndexingState:   idx.IndexingState,
		LastCrawlTime:   idx.LastCrawlTime,
		PageFetchState:  idx.PageFetchState,
		GoogleCanonical: idx.GoogleCanonical,
		UserCanonical:   idx.UserCanonical,
		InspectionTime:  now,
		CreatedAt:       now,
	}
	return insp, nil
}

// InspectURL satisfies the legacy simple interface with strict indexing truth
func (c *DefaultGSCClient) InspectURL(ctx context.Context, siteUrl string, inspectionUrl string) (bool, string, error) {
	insp, err := c.InspectURLDetails(ctx, siteUrl, inspectionUrl)
	if err != nil {
		return false, "ERROR", err
	}
	// "Do not claim indexing merely because inspection API responded HTTP 200"
	isIndexed := insp.Verdict == "PASS" && strings.Contains(strings.ToLower(insp.CoverageState), "indexed")
	return isIndexed, insp.Verdict, nil
}

// GetGSCSitemaps queries GSC for registered sitemaps
func (c *DefaultGSCClient) GetGSCSitemaps(ctx context.Context, siteUrl string) ([]GSCSitemap, error) {
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if siteUrl == "" {
		siteUrl = common.GetGSCSiteURL()
	}
	escapedSite := url.QueryEscape(siteUrl)
	apiURL := fmt.Sprintf("https://www.googleapis.com/webmasters/v3/sites/%s/sitemaps", escapedSite)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GSC sitemaps query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GSC sitemaps query failed HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Sitemap []struct {
			Path           string `json:"path"`
			LastSubmitted  string `json:"lastSubmitted"`
			IsPending      bool   `json:"isPending"`
			IsSitemapsList bool   `json:"isSitemapsList"`
			LastDownloaded string `json:"lastDownloaded"`
		} `json:"sitemap"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var list []GSCSitemap
	for _, s := range res.Sitemap {
		item := GSCSitemap{
			Path:           s.Path,
			IsPending:      s.IsPending,
			IsSitemapsList: s.IsSitemapsList,
		}
		if t, err := time.Parse(time.RFC3339, s.LastSubmitted); err == nil {
			item.LastSubmitted = t
		}
		if t, err := time.Parse(time.RFC3339, s.LastDownloaded); err == nil {
			item.LastDownloaded = t
		}
		list = append(list, item)
	}
	return list, nil
}

// IngestSearchMetrics processes incoming Search Console metrics with deduplication
func IngestSearchMetrics(rows []GSCMetricRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	ingested := 0
	for _, r := range rows {
		slug := ExtractSlugFromURL(r.Page)
		var postId int
		if slug != "" {
			post, _ := model.GetNewsPostBySlug(slug)
			if post != nil {
				postId = post.Id
			}
		}

		// Avoid duplicate metric snapshots
		var existing model.NewsSeoMetric
		err := model.DB.Where("snapshot_date = ? AND page_url = ? AND query = ?", r.Date, r.Page, r.Query).First(&existing).Error
		if err == nil {
			// Update existing record
			existing.Clicks = r.Clicks
			existing.Impressions = r.Impressions
			existing.Ctr = r.CTR
			existing.Position = r.Position
			_ = model.DB.Save(&existing)
			continue
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

		if err := model.CreateNewsSeoMetric(metric); err == nil {
			ingested++
		}
	}
	return ingested, nil
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

// PerformSelectiveUrlInspection inspects a specific post's URL and persists the observation
func PerformSelectiveUrlInspection(ctx context.Context, postId int) (*model.NewsUrlInspection, error) {
	post, err := model.GetNewsPostById(postId)
	if err != nil || post == nil {
		return nil, fmt.Errorf("post %d not found", postId)
	}
	targetUrl := post.CanonicalUrl
	if targetUrl == "" {
		targetUrl = fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), post.Slug)
	}

	client := NewDefaultGSCClient()
	insp, err := client.InspectURLDetails(ctx, common.GetGSCSiteURL(), targetUrl)
	if err != nil {
		return nil, err
	}
	insp.PostId = postId

	// Save to DB
	_ = model.CreateNewsUrlInspection(insp)
	logger.LogInfo(ctx, fmt.Sprintf("[GSC] Selective inspection for post %d (%s): verdict=%s, coverage=%s", postId, targetUrl, insp.Verdict, insp.CoverageState))

	now := common.GetTimestamp()
	// Real observation: CANONICAL_MISMATCH
	if insp.GoogleCanonical != "" && insp.UserCanonical != "" && insp.GoogleCanonical != insp.UserCanonical {
		_ = model.CreateNewsSeoOpportunity(&model.NewsSeoOpportunity{
			PostId:            postId,
			OpportunityType:   model.OpportunityCanonicalMismatch,
			Observation:       fmt.Sprintf("Google selected canonical %q differs from user declared canonical %q", insp.GoogleCanonical, insp.UserCanonical),
			EvidenceJSON:      fmt.Sprintf(`{"google_canonical":%q,"user_canonical":%q}`, insp.GoogleCanonical, insp.UserCanonical),
			Hypothesis:        "Aligning rel=canonical will prevent link equity dilution across duplicate URLs",
			ProposedAction:    fmt.Sprintf("Enforce declared canonical %q in header and sitemaps", insp.UserCanonical),
			RiskClass:         "low",
			Status:            "detected",
			CooldownExpiresAt: now + 7*86400,
		})
	}

	// Real observation: INDEXING_ANOMALY
	if insp.Verdict == "FAIL" || strings.Contains(strings.ToLower(insp.CoverageState), "error") {
		_ = model.CreateNewsSeoOpportunity(&model.NewsSeoOpportunity{
			PostId:            postId,
			OpportunityType:   model.OpportunityIndexingAnomaly,
			Observation:       fmt.Sprintf("URL inspection reported verdict %q with coverage %q", insp.Verdict, insp.CoverageState),
			EvidenceJSON:      fmt.Sprintf(`{"verdict":%q,"coverage_state":%q,"robots":%q}`, insp.Verdict, insp.CoverageState, insp.RobotsTxtState),
			Hypothesis:        "Remediating robots or server fetch issues will restore Googlebot crawler access",
			ProposedAction:    "Verify HTTP response and robots.txt rules for the article URL",
			RiskClass:         "medium",
			Status:            "detected",
			CooldownExpiresAt: now + 7*86400,
		})
	}

	return insp, nil
}

// DetectSeoOpportunities generates SEO opportunities ONLY from real observations (Section 8)
// Zero opportunities synthesized if metrics are missing or delayed (Section 6)
func DetectSeoOpportunities(posts []*model.NewsPost, metrics []GSCMetricRow) ([]*model.NewsSeoOpportunity, error) {
	if len(metrics) == 0 {
		// Strict truthfulness: NO_DATA_YET. Never synthesize opportunities from empty data!
		return nil, nil
	}

	var opportunities []*model.NewsSeoOpportunity
	now := common.GetTimestamp()

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
		if len(pageMetrics) == 0 {
			continue
		}

		totalImp := 0
		totalClicks := 0
		for _, m := range pageMetrics {
			totalImp += m.Impressions
			totalClicks += m.Clicks
		}

		// 1. HIGH_IMPRESSION_LOW_CTR: Impressions >= 100, CTR < 2.0%
		if totalImp >= 100 {
			overallCTR := (float64(totalClicks) / float64(totalImp)) * 100.0
			if overallCTR < 2.0 {
				opportunities = append(opportunities, &model.NewsSeoOpportunity{
					PostId:              post.Id,
					OpportunityType:     model.OpportunityHighImpLowCTR,
					Query:               "aggregate",
					CurrentImpressions:  totalImp,
					CurrentClicks:       totalClicks,
					CurrentCTR:          overallCTR,
					Observation:         fmt.Sprintf("Page received %d impressions with CTR %.2f%% over 28-day window", totalImp, overallCTR),
					EvidenceJSON:        fmt.Sprintf(`{"impressions":%d,"clicks":%d,"ctr":%.2f,"window_days":28}`, totalImp, totalClicks, overallCTR),
					Hypothesis:          "Improving headline click appeal and meta description clarity will raise CTR toward 4.0%",
					ProposedAction:      "Rewrite title and meta description with developer value hook and model benchmarks",
					RiskClass:           "low",
					Status:              "detected",
					CooldownExpiresAt:   now + 7*86400,
					CreatedAt:           now,
				})
			}
		}

		// 2. POSITION_8_TO_20: Ranking in striking distance with latent impressions
		for _, m := range pageMetrics {
			if m.Position >= 8.0 && m.Position <= 20.0 && m.Impressions >= 20 {
				opportunities = append(opportunities, &model.NewsSeoOpportunity{
					PostId:              post.Id,
					OpportunityType:     model.OpportunityPosition8To20,
					Query:               m.Query,
					CurrentImpressions:  m.Impressions,
					CurrentClicks:       m.Clicks,
					CurrentCTR:          m.CTR,
					CurrentPosition:     m.Position,
					Observation:         fmt.Sprintf("Striking distance ranking (pos %.1f) with %d impressions for query %q", m.Position, m.Impressions, m.Query),
					EvidenceJSON:        fmt.Sprintf(`{"query":%q,"position":%.1f,"impressions":%d,"clicks":%d}`, m.Query, m.Position, m.Impressions, m.Clicks),
					Hypothesis:          fmt.Sprintf("Enriching code examples and benchmarks for query %q will push ranking from pos %.1f into top 5", m.Query, m.Position),
					ProposedAction:      fmt.Sprintf("Add dedicated H3 section and Python/cURL example covering %q", m.Query),
					RiskClass:           "medium",
					Status:              "detected",
					CooldownExpiresAt:   now + 7*86400,
					CreatedAt:           now,
				})
			}
		}

		// 3. RISING_QUERY / NEW_QUERY_OPPORTUNITY: Queries not yet covered in body
		for _, m := range pageMetrics {
			if m.Impressions >= 25 && m.Query != "" && m.Query != "aggregate" {
				queryLower := strings.ToLower(m.Query)
				contentLower := strings.ToLower(post.ContentMarkdown)
				if !strings.Contains(contentLower, queryLower) {
					opportunities = append(opportunities, &model.NewsSeoOpportunity{
						PostId:              post.Id,
						OpportunityType:     model.OpportunityRisingQuery,
						Query:               m.Query,
						CurrentImpressions:  m.Impressions,
						CurrentClicks:       m.Clicks,
						CurrentCTR:          m.CTR,
						CurrentPosition:     m.Position,
						Observation:         fmt.Sprintf("Search query %q drove %d impressions but is not mentioned in article body", m.Query, m.Impressions),
						EvidenceJSON:        fmt.Sprintf(`{"query":%q,"impressions":%d,"clicks":%d,"position":%.1f}`, m.Query, m.Impressions, m.Clicks, m.Position),
						Hypothesis:          fmt.Sprintf("Adding explanation for query %q will capture high-intent developer searches", m.Query),
						ProposedAction:      fmt.Sprintf("Add technical FAQ / explanation for %q", m.Query),
						RiskClass:           "low",
						Status:              "detected",
						CooldownExpiresAt:   now + 7*86400,
						CreatedAt:           now,
					})
				}
			}
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

// ApplyOptimizationExperiment applies an SEO optimization and records it as an experiment (Section 9 & 10)
func ApplyOptimizationExperiment(ctx context.Context, opp *model.NewsSeoOpportunity) error {
	if opp == nil {
		return errors.New("nil opportunity")
	}

	eligible, reason, err := CanRemediatePost(opp.PostId)
	if err != nil {
		return err
	}
	if !eligible {
		return fmt.Errorf("optimization blocked by safety guard: %s", reason)
	}

	post, err := model.GetNewsPostById(opp.PostId)
	if err != nil || post == nil {
		return fmt.Errorf("post %d not found", opp.PostId)
	}

	now := common.GetTimestamp()
	changeType := "generic_refresh"
	beforeVal := ""
	afterVal := ""

	switch opp.OpportunityType {
	case model.OpportunityHighImpLowCTR, model.OpportunityHighImpLowCTRAlt:
		changeType = "title_and_meta"
		beforeVal = fmt.Sprintf("Title: %s | Desc: %s", post.SeoTitle, post.SeoDescription)
		if !strings.Contains(post.SeoTitle, "⚡") {
			post.SeoTitle = fmt.Sprintf("⚡ %s | Tora AI News", post.Title)
		}
		if post.SeoDescription == "" || len(post.SeoDescription) < 50 {
			post.SeoDescription = post.Summary
		}
		afterVal = fmt.Sprintf("Title: %s | Desc: %s", post.SeoTitle, post.SeoDescription)
		if err := model.UpdateNewsPost(post); err != nil {
			return err
		}

	case model.OpportunityRisingQuery, model.OpportunityNewQuery:
		changeType = "faq_section"
		beforeVal = "(none)"
		if opp.Query != "" && !strings.Contains(strings.ToLower(post.ContentMarkdown), strings.ToLower(opp.Query)) {
			addition := fmt.Sprintf("\n\n### ข้อมูลเพิ่มเติมเกี่ยวกับ %s\n\nสำหรับนักพัฒนาที่สนใจในประเด็น **%s** โมเดลนี้รองรับการทำงานร่วมกับ Tora API Gateway และรองรับการสลับ Fallback อัตโนมัติ\n", opp.Query, opp.Query)
			post.ContentMarkdown += addition
			post.ContentHTML = RenderMarkdownToSafeHTML(post.ContentMarkdown)
			afterVal = addition
			if err := model.UpdateNewsPost(post); err != nil {
				return err
			}
		}

	default:
		changeType = "content_freshness"
		beforeVal = fmt.Sprintf("UpdatedAt: %d", post.UpdatedAt)
		post.UpdatedAt = now
		afterVal = fmt.Sprintf("UpdatedAt: %d", post.UpdatedAt)
		_ = model.UpdateNewsPost(post)
	}

	// Persist experiment record
	exp := &model.NewsSeoExperiment{
		OpportunityId:       opp.Id,
		PostId:              opp.PostId,
		ChangeType:          changeType,
		BeforeValue:         beforeVal,
		AfterValue:          afterVal,
		AppliedAt:           now,
		BaselineImpressions: opp.CurrentImpressions,
		BaselineClicks:      opp.CurrentClicks,
		BaselineCTR:         opp.CurrentCTR,
		BaselinePosition:    opp.CurrentPosition,
		ResultVerdict:       "in_progress",
		CreatedAt:           now,
	}
	_ = model.CreateNewsSeoExperiment(exp)

	// Update opportunity state and lock in 7-day cooldown
	opp.Status = "applied"
	opp.AppliedAt = now
	opp.CooldownExpiresAt = now + 7*86400
	return model.UpdateNewsSeoOpportunity(opp)
}

// ApplySeoRemediation redirects to ApplyOptimizationExperiment
func ApplySeoRemediation(ctx context.Context, opp *model.NewsSeoOpportunity) error {
	return ApplyOptimizationExperiment(ctx, opp)
}

// GetNormalizedGSCStatus retrieves the truthful GSC status without fake claims (Section 2)
func GetNormalizedGSCStatus(ctx context.Context, forceRefresh bool) GSCStatusOverview {
	cachedGSCMu.RLock()
	if !forceRefresh && time.Now().Unix()-lastGSCCheckAt < 300 && cachedGSCStatus.Status != "" {
		defer cachedGSCMu.RUnlock()
		return cachedGSCStatus
	}
	cachedGSCMu.RUnlock()

	cachedGSCMu.Lock()
	defer cachedGSCMu.Unlock()

	siteUrl := common.GetGSCSiteURL()
	credFile := common.GetGSCCredentialsFile()
	credJSON := common.GetGSCCredentialsJSON()
	gac := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	res := GSCStatusOverview{
		SiteURL: siteUrl,
	}

	if credFile == "" && credJSON == "" && gac == "" {
		res.Status = GSCStatusNotConfigured
		cachedGSCStatus = res
		lastGSCCheckAt = time.Now().Unix()
		return res
	}

	res.Status = GSCStatusConfigured

	var rowCount int64
	if model.DB != nil {
		_ = model.DB.Model(&model.NewsSeoMetric{}).Count(&rowCount).Error
	}
	res.RowCount = rowCount
	res.DataAvailable = rowCount > 0

	client := NewDefaultGSCClient()
	perm, err := client.VerifySiteAccess(ctx, siteUrl)
	if err != nil {
		res.Status = GSCStatusError
		res.LastError = err.Error()
		cachedGSCStatus = res
		lastGSCCheckAt = time.Now().Unix()
		return res
	}

	res.PermissionLevel = perm
	if rowCount > 0 {
		res.Status = GSCStatusDataAvailable
	} else {
		res.Status = GSCStatusConnected
	}

	cachedGSCStatus = res
	lastGSCCheckAt = time.Now().Unix()
	return res
}
