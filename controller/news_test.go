package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupControllerNewsTestDB(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.NewsSource{},
		&model.StoryCluster{},
		&model.NewsPost{},
		&model.NewsDistribution{},
		&model.NewsAnalyticEvent{},
		&model.NewsSeoMetric{},
		&model.NewsSeoOpportunity{},
		&model.NewsConversionEvent{},
		&model.NewsDailyGrowthReview{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// Seed initial launchpack
	require.NoError(t, model.InitDefaultNewsPosts())
	require.NoError(t, model.InitDefaultNewsSources())

	t.Cleanup(func() {
		time.Sleep(50 * time.Millisecond)
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
		model.InvalidateNewsCache()
	})

	r := gin.New()

	// Public crawlable SEO & web routes (GET & HEAD)
	registerRoute := func(path string, handler gin.HandlerFunc) {
		r.GET(path, handler)
		r.HEAD(path, handler)
	}
	registerRoute("/news", RenderNewsIndexPage)
	registerRoute("/news/:slug", RenderNewsPostPage)
	registerRoute("/news/:slug/og.svg", RenderNewsOGCard)
	registerRoute("/news/:slug/og.png", RenderNewsOGPNGCard)
	registerRoute("/sitemap.xml", RenderSitemap)
	registerRoute("/news-sitemap.xml", RenderNewsSitemap)
	registerRoute("/robots.txt", RenderRobots)

	// Public API
	api := r.Group("/api")
	{
		api.GET("/news", GetNewsPostsAPI)
		api.GET("/news/:slug", GetNewsPostDetailAPI)
		api.POST("/news/conversion", RecordNewsConversionAPI)

		// Admin API (mock auth context for testing)
		admin := api.Group("/admin/news")
		{
			admin.GET("/posts", AdminGetNewsPosts)
			admin.POST("/posts", AdminCreateNewsPost)
			admin.PUT("/posts/:id", AdminUpdateNewsPost)
			admin.DELETE("/posts/:id", AdminDeleteNewsPost)
			admin.GET("/sources", AdminGetNewsSources)
			admin.POST("/sources/:id/sync", AdminSyncNewsSource)
			admin.GET("/seo/opportunities", AdminGetSeoOpportunities)
			admin.POST("/seo/remediate/:id", AdminRemediateSeoOpportunity)
			admin.GET("/reviews/daily", AdminGetDailyGrowthReviews)
			admin.POST("/scout/trigger", AdminTriggerNewsScout)
		}
	}

	return r
}

func TestController_RenderNewsIndexPage(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/news", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "text/html")

	body := w.Body.String()
	assert.Contains(t, body, "<title>")
	assert.Contains(t, body, "Tora AI Tech News")
	assert.Contains(t, body, common.GetCanonicalBaseURL()+"/news")
	assert.Contains(t, body, "OpenAI")
	assert.Contains(t, body, "Claude 3.7")
	assert.Contains(t, body, "DeepSeek-V3")
}

func TestController_RenderNewsPostPage_Success(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/news/openai-gpt-4-5-release-analysis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Verify SEO & Crawlability Elements
	assert.Contains(t, body, "GPT-4.5")
	assert.Contains(t, body, "canonical")
	assert.Contains(t, body, common.GetCanonicalBaseURL()+"/news/openai-gpt-4-5-release-analysis")
	assert.Contains(t, body, `schema.org`)
	assert.Contains(t, body, `"@type":"AnalysisNewsArticle"`)
	assert.Contains(t, body, `"@type":"BreadcrumbList"`)
	assert.Contains(t, body, `og:image`)
	assert.Contains(t, body, `/news/openai-gpt-4-5-release-analysis/og.png`)
	assert.Contains(t, body, `og:image:type`)
	assert.Contains(t, body, `image/png`)
	assert.Contains(t, body, "ตรวจสอบข้อเท็จจริงแล้ว")
	assert.Contains(t, body, "สรุปสาระสำคัญ (TL;DR)")
}

func TestController_RenderNewsPostPage_NotFound(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/news/non-existent-article-slug", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "404")
	assert.Contains(t, body, "ไม่พบบทความที่ร้องขอ")
}

func TestController_RenderNewsOGCard(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/news/openai-gpt-4-5-release-analysis/og.svg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "image/svg+xml")
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=86400")

	svg := w.Body.String()
	assert.Contains(t, svg, "<svg width=\"1200\" height=\"630\"")
	assert.Contains(t, svg, "Tora AI")
	assert.Contains(t, svg, "NEWSROOM")
}

func TestController_RenderSitemaps(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	// 1. Standard Sitemap
	req1, _ := http.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Header().Get("Content-Type"), "application/xml")
	xml1 := w1.Body.String()
	assert.Contains(t, xml1, "<urlset")
	assert.Contains(t, xml1, common.GetCanonicalBaseURL()+"/news/openai-gpt-4-5-release-analysis")

	// 2. Google News Sitemap
	req2, _ := http.NewRequest(http.MethodGet, "/news-sitemap.xml", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Header().Get("Content-Type"), "application/xml")
	xml2 := w2.Body.String()
	assert.Contains(t, xml2, "<urlset")
	assert.Contains(t, xml2, "xmlns:news=")
}

func TestController_RenderRobots(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/robots.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	txt := w.Body.String()
	assert.Contains(t, txt, "Allow: /news")
	assert.Contains(t, txt, "Disallow: /news-admin")
	assert.Contains(t, txt, "Sitemap: "+common.GetCanonicalBaseURL()+"/sitemap.xml")
}

func TestController_HeadRequests(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	headEndpoints := []string{"/news", "/sitemap.xml", "/news-sitemap.xml", "/robots.txt"}
	for _, ep := range headEndpoints {
		req, _ := http.NewRequest(http.MethodHead, ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "HEAD %s must return 200", ep)
	}
}

func TestController_PublicNewsAPI(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	// List
	reqList, _ := http.NewRequest(http.MethodGet, "/api/news", nil)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	assert.Equal(t, http.StatusOK, wList.Code)
	var listResp struct {
		Success bool              `json:"success"`
		Data    []*model.NewsPost `json:"data"`
		Total   int64             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &listResp))
	assert.True(t, listResp.Success)
	assert.Equal(t, int64(6), listResp.Total)
	assert.Len(t, listResp.Data, 6)

	// Detail
	reqDetail, _ := http.NewRequest(http.MethodGet, "/api/news/openai-gpt-4-5-release-analysis", nil)
	wDetail := httptest.NewRecorder()
	router.ServeHTTP(wDetail, reqDetail)

	assert.Equal(t, http.StatusOK, wDetail.Code)
	var detailResp struct {
		Success bool            `json:"success"`
		Data    *model.NewsPost `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wDetail.Body.Bytes(), &detailResp))
	assert.True(t, detailResp.Success)
	require.NotNil(t, detailResp.Data)
	assert.Equal(t, "openai-gpt-4-5-release-analysis", detailResp.Data.Slug)
}

func TestController_AdminNewsCRUD(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	// 1. Create Post
	newPost := model.NewsPost{
		Title:           "Admin Test Story",
		Summary:         "Summary for testing admin creation",
		ContentMarkdown: "## Section\n\nContent for admin creation",
		Status:          model.NewsStatusDraft,
		ContentRisk:     model.ContentRiskLow,
		FactCheckStatus: model.FactCheckVerified,
	}
	bodyBytes, _ := json.Marshal(newPost)
	reqCreate, _ := http.NewRequest(http.MethodGet, "/api/admin/news/posts", bytes.NewBuffer(bodyBytes))
	reqCreate.Method = http.MethodPost
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, reqCreate)

	assert.Equal(t, http.StatusOK, wCreate.Code)
	var createResp struct {
		Success bool            `json:"success"`
		Data    *model.NewsPost `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &createResp))
	assert.True(t, createResp.Success)
	require.NotNil(t, createResp.Data)
	createdId := createResp.Data.Id
	assert.Positive(t, createdId)
	assert.Equal(t, "admin-test-story", createResp.Data.Slug)
	assert.Contains(t, createResp.Data.ContentHTML, "<h2")

	// 2. Update Post
	updateReq := model.NewsPost{
		Title:   "Admin Test Story Updated",
		Summary: "Updated summary",
		Status:  model.NewsStatusPublished,
	}
	upBytes, _ := json.Marshal(updateReq)
	reqUp, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/news/posts/%d", createdId), bytes.NewBuffer(upBytes))
	reqUp.Header.Set("Content-Type", "application/json")
	wUp := httptest.NewRecorder()
	router.ServeHTTP(wUp, reqUp)

	assert.Equal(t, http.StatusOK, wUp.Code)
	var upResp struct {
		Success bool            `json:"success"`
		Data    *model.NewsPost `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wUp.Body.Bytes(), &upResp))
	assert.Equal(t, "Admin Test Story Updated", upResp.Data.Title)
	assert.Equal(t, model.NewsStatusPublished, upResp.Data.Status)

	// 3. Delete Post
	reqDel, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/news/posts/%d", createdId), nil)
	wDel := httptest.NewRecorder()
	router.ServeHTTP(wDel, reqDel)

	assert.Equal(t, http.StatusOK, wDel.Code)

	// Verify post is deleted
	_, err := model.GetNewsPostById(createdId)
	assert.Error(t, err)

	// 4. Get Sources
	reqSources, _ := http.NewRequest(http.MethodGet, "/api/admin/news/sources", nil)
	wSources := httptest.NewRecorder()
	router.ServeHTTP(wSources, reqSources)

	assert.Equal(t, http.StatusOK, wSources.Code)
	var srcResp struct {
		Success bool                `json:"success"`
		Data    []*model.NewsSource `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wSources.Body.Bytes(), &srcResp))
	assert.True(t, srcResp.Success)
	assert.GreaterOrEqual(t, len(srcResp.Data), 6)
}

func TestController_RenderNewsOGPNGCard(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/news/openai-gpt-4-5-release-analysis/og.png", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=86400")

	body := w.Body.Bytes()
	// Standard PNG magic signature: \x89PNG\r\n\x1a\n
	require.Greater(t, len(body), 8)
	assert.Equal(t, byte(0x89), body[0])
	assert.Equal(t, byte('P'), body[1])
	assert.Equal(t, byte('N'), body[2])
	assert.Equal(t, byte('G'), body[3])
}

func TestController_RecordNewsConversion(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	convReq := map[string]interface{}{
		"event_type":   "signup",
		"content_id":   "openai-gpt-4-5-release-analysis",
		"utm_source":   "google",
		"utm_medium":   "organic",
		"utm_campaign": "growth_autopilot",
		"revenue_thb":  0.0,
	}
	reqBytes, _ := json.Marshal(convReq)

	req, _ := http.NewRequest(http.MethodPost, "/api/news/conversion", bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestController_AdminSeoOpportunitiesAndDailyReview(t *testing.T) {
	router := setupControllerNewsTestDB(t)

	// 1. Get daily growth reviews
	reqRev, _ := http.NewRequest(http.MethodGet, "/api/admin/news/reviews/daily", nil)
	wRev := httptest.NewRecorder()
	router.ServeHTTP(wRev, reqRev)

	assert.Equal(t, http.StatusOK, wRev.Code)
	var revResp struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(wRev.Body.Bytes(), &revResp))
	assert.True(t, revResp.Success)

	// 2. Get SEO opportunities
	reqOpp, _ := http.NewRequest(http.MethodGet, "/api/admin/news/seo/opportunities", nil)
	wOpp := httptest.NewRecorder()
	router.ServeHTTP(wOpp, reqOpp)

	assert.Equal(t, http.StatusOK, wOpp.Code)
	var oppResp struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(wOpp.Body.Bytes(), &oppResp))
	assert.True(t, oppResp.Success)
}

