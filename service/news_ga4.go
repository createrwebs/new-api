package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"golang.org/x/oauth2/jwt"
)

// GA4 Status definitions (Section 2)
const (
	GA4StatusNotConfigured   = "NOT_CONFIGURED"
	GA4StatusOperatorBlocked = "OPERATOR_BLOCKED"
	GA4StatusConnected       = "CONNECTED"
	GA4StatusDataAvailable   = "DATA_AVAILABLE"
	GA4StatusError           = "ERROR"
)

// GA4StatusOverview provides truthful operational state of GA4 analytics
type GA4StatusOverview struct {
	Status           string `json:"status"` // OPERATOR_BLOCKED, CONNECTED, DATA_AVAILABLE, ERROR
	PropertyID       string `json:"property_id"`
	MeasurementID    string `json:"measurement_id"`
	DataAvailable    bool   `json:"data_available"`
	TotalUsers       int64  `json:"total_users"`
	TotalSessions    int64  `json:"total_sessions"`
	TotalConversions int64  `json:"total_conversions"`
	LastSyncAt       int64  `json:"last_sync_at"`
	LastError        string `json:"last_error,omitempty"`
}

var (
	cachedGA4Status GA4StatusOverview
	cachedGA4Mu     sync.RWMutex
	lastGA4CheckAt  int64
)

// GA4LandingPageMetric represents traffic & engagement for a specific landing page
type GA4LandingPageMetric struct {
	LandingPage    string  `json:"landing_page"`
	Sessions       int     `json:"sessions"`
	ActiveUsers    int     `json:"active_users"`
	EngagementRate float64 `json:"engagement_rate"`
	Conversions    int     `json:"conversions"`
	EventCount     int     `json:"event_count"`
	Date           string  `json:"date"`
}

// GA4Client defines read-only GA4 Data API operations
type GA4Client interface {
	GetNormalizedGA4Status(ctx context.Context, forceRefresh bool) GA4StatusOverview
	QueryLandingPageMetrics(ctx context.Context, startDate, endDate string) ([]GA4LandingPageMetric, error)
	QueryConversionEvents(ctx context.Context, startDate, endDate string) (map[string]int, error)
}

// DefaultGA4Client connects to Google Analytics Data API v1beta
type DefaultGA4Client struct {
	PropertyID      string
	MeasurementID   string
	CredentialsJSON string
	CredentialsFile string
	ReadOnlyScope   string
	HTTPClient      *http.Client
}

func NewDefaultGA4Client() *DefaultGA4Client {
	propID := common.GetGA4PropertyID()
	measID := common.GetGA4MeasurementID()
	credsFile := strings.TrimSpace(os.Getenv("GA4_CREDENTIALS_FILE"))
	credsJSON := strings.TrimSpace(os.Getenv("GA4_CREDENTIALS_JSON"))

	// Fallback to GSC service account credentials if GA4-specific file is not explicitly set
	if credsFile == "" && credsJSON == "" {
		credsFile = common.GetGSCCredentialsFile()
		credsJSON = common.GetGSCCredentialsJSON()
	}

	if credsFile != "" && credsJSON == "" {
		if content, err := os.ReadFile(credsFile); err == nil {
			credsJSON = string(content)
		}
	}

	return &DefaultGA4Client{
		PropertyID:      propID,
		MeasurementID:   measID,
		CredentialsJSON: credsJSON,
		CredentialsFile: credsFile,
		ReadOnlyScope:   "https://www.googleapis.com/auth/analytics.readonly",
		HTTPClient:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *DefaultGA4Client) getAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	if c.CredentialsJSON == "" {
		return nil, errors.New("GA4 credentials not configured (OPERATOR_BLOCKED)")
	}
	var key struct {
		PrivateKeyID string `json:"private_key_id"`
		PrivateKey   string `json:"private_key"`
		ClientEmail  string `json:"client_email"`
		TokenURI     string `json:"token_uri"`
	}
	if err := json.Unmarshal([]byte(c.CredentialsJSON), &key); err != nil {
		return nil, fmt.Errorf("failed to parse GA4 service account JSON: %w", err)
	}
	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, errors.New("invalid service account key: client_email or private_key is empty")
	}

	conf := &jwt.Config{
		Email:        key.ClientEmail,
		PrivateKey:   []byte(key.PrivateKey),
		PrivateKeyID: key.PrivateKeyID,
		Scopes:       []string{c.ReadOnlyScope},
		TokenURL:     key.TokenURI,
	}
	if conf.TokenURL == "" {
		conf.TokenURL = "https://oauth2.googleapis.com/token"
	}
	return conf.Client(ctx), nil
}

// GetNormalizedGA4Status inspects GA4 readiness with thread-safe cached TTL (Section 2)
func (c *DefaultGA4Client) GetNormalizedGA4Status(ctx context.Context, forceRefresh bool) GA4StatusOverview {
	cachedGA4Mu.RLock()
	now := common.GetTimestamp()
	if !forceRefresh && lastGA4CheckAt > 0 && (now-lastGA4CheckAt) < 300 {
		status := cachedGA4Status
		cachedGA4Mu.RUnlock()
		return status
	}
	cachedGA4Mu.RUnlock()

	cachedGA4Mu.Lock()
	defer cachedGA4Mu.Unlock()

	overview := GA4StatusOverview{
		PropertyID:    c.PropertyID,
		MeasurementID: c.MeasurementID,
		LastSyncAt:    now,
	}

	// Boundary check: GA4 Property ID and credentials
	if c.PropertyID == "" {
		overview.Status = GA4StatusOperatorBlocked
		overview.LastError = "GA4_PROPERTY_ID not configured (OPERATOR_BLOCKED)"
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}

	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		overview.Status = GA4StatusOperatorBlocked
		overview.LastError = err.Error()
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}

	// Verify property query access via runReport test
	cleanProp := strings.TrimPrefix(c.PropertyID, "properties/")
	apiURL := fmt.Sprintf("https://analyticsdata.googleapis.com/v1beta/properties/%s:runReport", cleanProp)

	reqPayload := map[string]interface{}{
		"dateRanges": []map[string]string{
			{"startDate": "7daysAgo", "endDate": "yesterday"},
		},
		"metrics": []map[string]string{
			{"name": "activeUsers"},
			{"name": "sessions"},
			{"name": "conversions"},
		},
		"limit": 1,
	}
	reqBytes, _ := json.Marshal(reqPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		overview.Status = GA4StatusError
		overview.LastError = err.Error()
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		overview.Status = GA4StatusError
		overview.LastError = err.Error()
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		overview.Status = GA4StatusOperatorBlocked
		overview.LastError = fmt.Sprintf("GA4 API HTTP %d: %s", resp.StatusCode, string(body))
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}

	var res struct {
		RowCount int `json:"rowCount"`
		Rows     []struct {
			MetricValues []struct {
				Value string `json:"value"`
			} `json:"metricValues"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		overview.Status = GA4StatusError
		overview.LastError = err.Error()
		cachedGA4Status = overview
		lastGA4CheckAt = now
		return overview
	}

	overview.Status = GA4StatusConnected
	if res.RowCount > 0 {
		overview.Status = GA4StatusDataAvailable
		overview.DataAvailable = true
	}

	cachedGA4Status = overview
	lastGA4CheckAt = now
	return overview
}

// QueryLandingPageMetrics fetches landing page traffic (read-only)
func (c *DefaultGA4Client) QueryLandingPageMetrics(ctx context.Context, startDate, endDate string) ([]GA4LandingPageMetric, error) {
	if c.PropertyID == "" {
		return nil, errors.New("GA4_PROPERTY_ID not configured (OPERATOR_BLOCKED)")
	}
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if startDate == "" {
		startDate = "28daysAgo"
	}
	if endDate == "" {
		endDate = "yesterday"
	}

	cleanProp := strings.TrimPrefix(c.PropertyID, "properties/")
	apiURL := fmt.Sprintf("https://analyticsdata.googleapis.com/v1beta/properties/%s:runReport", cleanProp)

	reqPayload := map[string]interface{}{
		"dateRanges": []map[string]string{
			{"startDate": startDate, "endDate": endDate},
		},
		"dimensions": []map[string]string{
			{"name": "landingPagePlusQueryString"},
		},
		"metrics": []map[string]string{
			{"name": "sessions"},
			{"name": "activeUsers"},
			{"name": "engagementRate"},
			{"name": "conversions"},
		},
		"limit": 1000,
	}
	reqBytes, _ := json.Marshal(reqPayload)
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
		return nil, fmt.Errorf("GA4 API error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Rows []struct {
			DimensionValues []struct {
				Value string `json:"value"`
			} `json:"dimensionValues"`
			MetricValues []struct {
				Value string `json:"value"`
			} `json:"metricValues"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var metrics []GA4LandingPageMetric
	for _, r := range res.Rows {
		if len(r.DimensionValues) == 0 || len(r.MetricValues) < 4 {
			continue
		}
		var sessions, users, convs int
		var engRate float64
		fmt.Sscanf(r.MetricValues[0].Value, "%d", &sessions)
		fmt.Sscanf(r.MetricValues[1].Value, "%d", &users)
		fmt.Sscanf(r.MetricValues[2].Value, "%f", &engRate)
		fmt.Sscanf(r.MetricValues[3].Value, "%d", &convs)

		metrics = append(metrics, GA4LandingPageMetric{
			LandingPage:    r.DimensionValues[0].Value,
			Sessions:       sessions,
			ActiveUsers:    users,
			EngagementRate: engRate,
			Conversions:    convs,
			Date:           endDate,
		})
	}

	return metrics, nil
}

// QueryConversionEvents aggregates named conversion events (Section 2)
func (c *DefaultGA4Client) QueryConversionEvents(ctx context.Context, startDate, endDate string) (map[string]int, error) {
	if c.PropertyID == "" {
		return nil, errors.New("GA4_PROPERTY_ID not configured (OPERATOR_BLOCKED)")
	}
	client, err := c.getAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if startDate == "" {
		startDate = "28daysAgo"
	}
	if endDate == "" {
		endDate = "yesterday"
	}

	cleanProp := strings.TrimPrefix(c.PropertyID, "properties/")
	apiURL := fmt.Sprintf("https://analyticsdata.googleapis.com/v1beta/properties/%s:runReport", cleanProp)

	reqPayload := map[string]interface{}{
		"dateRanges": []map[string]string{
			{"startDate": startDate, "endDate": endDate},
		},
		"dimensions": []map[string]string{
			{"name": "eventName"},
		},
		"metrics": []map[string]string{
			{"name": "eventCount"},
		},
		"limit": 100,
	}
	reqBytes, _ := json.Marshal(reqPayload)
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
		return nil, fmt.Errorf("GA4 API error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Rows []struct {
			DimensionValues []struct {
				Value string `json:"value"`
			} `json:"dimensionValues"`
			MetricValues []struct {
				Value string `json:"value"`
			} `json:"metricValues"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	events := make(map[string]int)
	for _, r := range res.Rows {
		if len(r.DimensionValues) > 0 && len(r.MetricValues) > 0 {
			var count int
			fmt.Sscanf(r.MetricValues[0].Value, "%d", &count)
			events[r.DimensionValues[0].Value] = count
		}
	}
	return events, nil
}
