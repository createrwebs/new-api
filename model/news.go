package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// News publication statuses
const (
	NewsStatusDraft          = "draft"
	NewsStatusReviewRequired = "review_required"
	NewsStatusScheduled      = "scheduled"
	NewsStatusPublished      = "published"
	NewsStatusArchived       = "archived"
)

// Publication origins (Section 5)
const (
	PublicationOriginAutopilot     = "AUTOPILOT"
	PublicationOriginManualAdmin   = "MANUAL_ADMIN"
	PublicationOriginSeed          = "SEED"
	PublicationOriginImport        = "IMPORT"
	PublicationOriginApi           = "API"
	PublicationOriginUnknownLegacy = "UNKNOWN_LEGACY"
)

// Publication event types (Section 6)
const (
	EventTypeAutopilotInitialPublish = "AUTOPILOT_INITIAL_PUBLISH"
	EventTypeManualInitialPublish    = "MANUAL_INITIAL_PUBLISH"
)

// Content types
const (
	ContentTypeNews      = "news"
	ContentTypeGuide     = "guide"
	ContentTypeAnalysis  = "analysis"
	ContentTypeChangelog = "changelog"
)

// Content risk levels
const (
	ContentRiskLow    = "low"
	ContentRiskMedium = "medium"
	ContentRiskHigh   = "high"
)

// Fact check statuses
const (
	FactCheckPending  = "pending"
	FactCheckVerified = "verified"
	FactCheckDisputed = "disputed"
)

// Distribution platforms
const (
	DistPlatformFacebook   = "facebook"
	DistPlatformLinkedIn   = "linkedin"
	DistPlatformTwitter    = "twitter"
	DistPlatformDevTo      = "devto"
	DistPlatformVideoShort = "video_short"
)

// NewsSource represents an authoritative external technology news feed or API.
type NewsSource struct {
	Id                     int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                   string `json:"name" gorm:"type:varchar(128);not null"`
	Slug                   string `json:"slug" gorm:"type:varchar(128);uniqueIndex;not null"`
	FeedUrl                string `json:"feed_url" gorm:"type:varchar(512);not null"`
	SiteUrl                string `json:"site_url" gorm:"type:varchar(512)"`
	SourceType             string `json:"source_type" gorm:"type:varchar(32);default:'rss'"` // "rss", "atom", "github_release", "api"
	TrustTier              string `json:"trust_tier" gorm:"type:varchar(32);default:'tier_1_official'"`
	Enabled                bool   `json:"enabled" gorm:"default:true;index"`
	PollingIntervalMinutes int    `json:"polling_interval_minutes" gorm:"default:30"`
	LastFetchedAt          int64  `json:"last_fetched_at" gorm:"bigint;default:0"`
	LastSuccessAt          int64  `json:"last_success_at" gorm:"bigint;default:0"`
	LastHttpStatus         int    `json:"last_http_status" gorm:"default:0"`
	ParseStatus            string `json:"parse_status" gorm:"type:varchar(32);default:'ok'"`
	LatestItemDate         int64  `json:"latest_item_date" gorm:"bigint;default:0"`
	ConsecutiveFailures    int    `json:"consecutive_failures" gorm:"default:0"`
	FetchErrorCount        int    `json:"fetch_error_count" gorm:"default:0"`
	LastError              string `json:"last_error" gorm:"type:varchar(512);default:''"`
	Tags                   string `json:"tags" gorm:"type:varchar(255);default:''"` // Comma-separated
	CreatedAt              int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64  `json:"updated_at" gorm:"bigint"`
}

func (s *NewsSource) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *NewsSource) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

// NewsFeedItem tracks discovered source items for strict deduplication (Section 3).
type NewsFeedItem struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceId    int    `json:"source_id" gorm:"index;not null"`
	Guid        string `json:"guid" gorm:"type:varchar(512);index;not null"`
	Url         string `json:"url" gorm:"type:varchar(512);index;not null"`
	Title       string `json:"title" gorm:"type:varchar(512);not null"`
	PublishedAt int64  `json:"published_at" gorm:"bigint;index"`
	ClusterId   int    `json:"cluster_id" gorm:"index;default:0"`
	Status      string `json:"status" gorm:"type:varchar(32);default:'discovered';index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
}

func (f *NewsFeedItem) BeforeCreate(tx *gorm.DB) error {
	if f.CreatedAt == 0 {
		f.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func IsFeedItemProcessed(guid, url string) bool {
	if DB == nil {
		return false
	}
	var count int64
	if guid != "" && url != "" {
		DB.Model(&NewsFeedItem{}).Where("guid = ? OR url = ?", guid, url).Count(&count)
	} else if guid != "" {
		DB.Model(&NewsFeedItem{}).Where("guid = ?", guid).Count(&count)
	} else if url != "" {
		DB.Model(&NewsFeedItem{}).Where("url = ?", url).Count(&count)
	}
	return count > 0
}

func RecordFeedItem(item *NewsFeedItem) error {
	if DB == nil || item == nil {
		return nil
	}
	return DB.Create(item).Error
}

// NewsAiVisibilityObservation tracks AI search discovery, crawler visits, and citations (Section 22).
type NewsAiVisibilityObservation struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Provider        string `json:"provider" gorm:"type:varchar(64);index;not null"` // "openai", "perplexity", "claude", "gemini"
	ObservationType string `json:"observation_type" gorm:"type:varchar(32);index;not null"` // "citation", "referral", "crawler"
	Query           string `json:"query" gorm:"type:varchar(512)"`
	ToraCited       bool   `json:"tora_cited" gorm:"index"`
	CitedUrl        string `json:"cited_url" gorm:"type:varchar(512)"`
	CheckedAt       int64  `json:"checked_at" gorm:"bigint;index"`
	Details         string `json:"details" gorm:"type:text"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint"`
}

func (o *NewsAiVisibilityObservation) BeforeCreate(tx *gorm.DB) error {
	if o.CreatedAt == 0 {
		o.CreatedAt = common.GetTimestamp()
	}
	if o.CheckedAt == 0 {
		o.CheckedAt = common.GetTimestamp()
	}
	return nil
}

func CreateAiVisibilityObservation(obs *NewsAiVisibilityObservation) error {
	if DB == nil || obs == nil {
		return nil
	}
	return DB.Create(obs).Error
}

func GetRecentAiVisibilityObservations(limit int) ([]*NewsAiVisibilityObservation, error) {
	if DB == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var list []*NewsAiVisibilityObservation
	err := DB.Model(&NewsAiVisibilityObservation{}).Order("checked_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// StoryCluster groups deduplicated stories from one or more sources covering the same technical event.
type StoryCluster struct {
	Id              int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Title           string  `json:"title" gorm:"type:varchar(255);not null"`
	Summary         string  `json:"summary" gorm:"type:text"`
	PrimarySourceId int     `json:"primary_source_id" gorm:"index"`
	PrimaryUrl      string  `json:"primary_url" gorm:"type:varchar(512);not null"`
	Category        string  `json:"category" gorm:"type:varchar(64);default:'model_release';index"`
	Tags            string  `json:"tags" gorm:"type:varchar(255);default:''"`
	RelevanceScore  float64 `json:"relevance_score" gorm:"type:float;default:0"`
	DeveloperScore  float64 `json:"developer_score" gorm:"type:float;default:0"`
	ThailandScore   float64 `json:"thailand_score" gorm:"type:float;default:0"`
	Status          string  `json:"status" gorm:"type:varchar(32);default:'discovered';index"`
	FirstSeenAt     int64   `json:"first_seen_at" gorm:"bigint;index"`
	LastEventAt     int64   `json:"last_event_at" gorm:"bigint"`
	CreatedAt       int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt       int64   `json:"updated_at" gorm:"bigint"`
}

func (c *StoryCluster) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.FirstSeenAt == 0 {
		c.FirstSeenAt = now
	}
	if c.LastEventAt == 0 {
		c.LastEventAt = now
	}
	return nil
}

func (c *StoryCluster) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = common.GetTimestamp()
	return nil
}

// NewsPost represents a published or drafted technology article written in Thai for Tora AI.
type NewsPost struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Slug            string `json:"slug" gorm:"type:varchar(160);uniqueIndex;not null"`
	ContentType     string `json:"content_type" gorm:"type:varchar(32);default:'news';index"` // "news", "guide", "analysis", "changelog"
	Title           string `json:"title" gorm:"type:varchar(255);not null"`
	Summary         string `json:"summary" gorm:"type:text"`
	ContentMarkdown string `json:"content_markdown" gorm:"type:text"`
	ContentHTML     string `json:"content_html" gorm:"type:text"`
	ClusterId       int    `json:"cluster_id" gorm:"index;default:0"`
	SourceId        int    `json:"source_id" gorm:"index;default:0"`
	SourceUrl       string `json:"source_url" gorm:"type:varchar(512)"`
	AuthorName      string `json:"author_name" gorm:"type:varchar(64);default:'Tora Technical Editorial'"`
	CanonicalUrl    string `json:"canonical_url" gorm:"type:varchar(512)"`
	Status          string `json:"status" gorm:"type:varchar(32);default:'draft';index"`
	ContentRisk     string `json:"content_risk" gorm:"type:varchar(32);default:'low'"`
	FactCheckStatus string `json:"fact_check_status" gorm:"type:varchar(32);default:'verified'"`
	FactCheckNotes  string `json:"fact_check_notes" gorm:"type:varchar(512);default:''"`
	SeoTitle        string `json:"seo_title" gorm:"type:varchar(255)"`
	SeoDescription  string `json:"seo_description" gorm:"type:varchar(320)"`
	SeoKeywords     string `json:"seo_keywords" gorm:"type:varchar(255)"`
	OgImageUrl        string `json:"og_image_url" gorm:"type:varchar(512)"`
	IsSeed            bool   `json:"is_seed" gorm:"default:false;index"`
	PublicationOrigin string `json:"publication_origin" gorm:"type:varchar(32);default:'UNKNOWN_LEGACY';index"`
	PublishedAt       int64  `json:"published_at" gorm:"bigint;index"`
	ViewCount         int    `json:"view_count" gorm:"default:0"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

func (p *NewsPost) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.ContentType == "" {
		p.ContentType = ContentTypeNews
	}
	if p.PublicationOrigin == "" {
		if p.IsSeed {
			p.PublicationOrigin = PublicationOriginSeed
		} else {
			p.PublicationOrigin = PublicationOriginUnknownLegacy
		}
	}
	if p.PublishedAt == 0 && p.Status == NewsStatusPublished {
		p.PublishedAt = now
	}
	if p.CanonicalUrl == "" && p.Slug != "" {
		p.CanonicalUrl = fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), p.Slug)
	}
	if p.OgImageUrl == "" && p.Slug != "" {
		p.OgImageUrl = fmt.Sprintf("%s/news/%s/og.png", common.GetCanonicalBaseURL(), p.Slug)
	}
	return nil
}

// NewsPublicationEvent records immutable, idempotent publication events (Section 6)
type NewsPublicationEvent struct {
	Id                     int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId                 int    `json:"post_id" gorm:"index;not null"`
	PublicationOrigin      string `json:"publication_origin" gorm:"type:varchar(32);not null"`
	EventType              string `json:"event_type" gorm:"type:varchar(32);not null"`
	PublishedAt            int64  `json:"published_at" gorm:"bigint;index;not null"`
	BangkokPublicationDate string `json:"bangkok_publication_date" gorm:"type:varchar(10);index;not null"`
	SourceId               int    `json:"source_id" gorm:"index;not null"`
	StoryClusterId         int    `json:"story_cluster_id" gorm:"index;not null"`
	AutopilotRunId         string `json:"autopilot_run_id" gorm:"type:varchar(64)"`
	BatchId                string `json:"batch_id" gorm:"type:varchar(64)"`
	CreatedAt              int64  `json:"created_at" gorm:"bigint;autoCreateTime"`
}

// NewsAutopilotDailyQuota maintains authoritative, atomic daily quota ledgers in PostgreSQL (Section 8)
type NewsAutopilotDailyQuota struct {
	BangkokDate     string `json:"bangkok_date" gorm:"primaryKey;type:varchar(10)"`
	PublishedCount  int    `json:"published_count" gorm:"default:0;not null"`
	LastPublishedAt int64  `json:"last_published_at" gorm:"bigint"`
	UpdatedAt       int64  `json:"updated_at" gorm:"bigint"`
}

func (p *NewsPost) BeforeUpdate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.UpdatedAt = now
	if p.Status == NewsStatusPublished && p.PublishedAt == 0 {
		p.PublishedAt = now
	}
	return nil
}

// NewsDistribution tracks derivative posts scheduled or published across external channels.
type NewsDistribution struct {
	Id                  int    `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId              int    `json:"post_id" gorm:"index;not null"`
	Platform            string `json:"platform" gorm:"type:varchar(32);index;not null"`
	DistributionVersion int    `json:"distribution_version" gorm:"default:1;index"`
	IsRepublish         bool   `json:"is_republish" gorm:"default:false"`
	Status              string `json:"status" gorm:"type:varchar(32);default:'pending';index"`
	ContentPayload      string `json:"content_payload" gorm:"type:text"`
	ExternalPostId      string `json:"external_post_id" gorm:"type:varchar(128);default:''"`
	RemotePostId        string `json:"remote_post_id" gorm:"type:varchar(128);default:''"`
	ExternalUrl         string `json:"external_url" gorm:"type:varchar(512);default:''"`
	RemoteUrl           string `json:"remote_url" gorm:"type:varchar(512);default:''"`
	ScheduledAt         int64  `json:"scheduled_at" gorm:"bigint;default:0"`
	PublishedAt         int64  `json:"published_at" gorm:"bigint;default:0"`
	AttemptCount        int    `json:"attempt_count" gorm:"default:0"`
	IdempotencyKey      string `json:"idempotency_key" gorm:"type:varchar(128);index;default:''"`
	ErrorMessage        string `json:"error_message" gorm:"type:varchar(512);default:''"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint"`
}

func (d *NewsDistribution) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	d.CreatedAt = now
	d.UpdatedAt = now
	if d.DistributionVersion <= 0 {
		d.DistributionVersion = 1
	}
	if d.IdempotencyKey == "" {
		h := sha256.New()
		h.Write([]byte(fmt.Sprintf("%d:%s:%d", d.PostId, d.Platform, d.DistributionVersion)))
		d.IdempotencyKey = hex.EncodeToString(h.Sum(nil))
	}
	if d.RemotePostId != "" && d.ExternalPostId == "" {
		d.ExternalPostId = d.RemotePostId
	}
	if d.ExternalPostId != "" && d.RemotePostId == "" {
		d.RemotePostId = d.ExternalPostId
	}
	if d.RemoteUrl != "" && d.ExternalUrl == "" {
		d.ExternalUrl = d.RemoteUrl
	}
	if d.ExternalUrl != "" && d.RemoteUrl == "" {
		d.RemoteUrl = d.ExternalUrl
	}
	return nil
}

func (d *NewsDistribution) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = common.GetTimestamp()
	if d.RemotePostId != "" && d.ExternalPostId == "" {
		d.ExternalPostId = d.RemotePostId
	}
	if d.ExternalPostId != "" && d.RemotePostId == "" {
		d.RemotePostId = d.ExternalPostId
	}
	if d.RemoteUrl != "" && d.ExternalUrl == "" {
		d.ExternalUrl = d.RemoteUrl
	}
	if d.ExternalUrl != "" && d.RemoteUrl == "" {
		d.RemoteUrl = d.ExternalUrl
	}
	return nil
}

// NewsAnalyticEvent records page interactions and qualified acquisition conversions.
type NewsAnalyticEvent struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId      int    `json:"post_id" gorm:"index;not null"`
	EventType   string `json:"event_type" gorm:"type:varchar(64);index;not null"`
	Referrer    string `json:"referrer" gorm:"type:varchar(512)"`
	UtmSource   string `json:"utm_source" gorm:"type:varchar(64)"`
	UtmMedium   string `json:"utm_medium" gorm:"type:varchar(64)"`
	UtmCampaign string `json:"utm_campaign" gorm:"type:varchar(64)"`
	IpHash      string `json:"ip_hash" gorm:"type:varchar(64)"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
}

var (
	newsCacheMu sync.RWMutex
	slugCache   = make(map[string]*NewsPost)
)

// InvalidateNewsCache clears local cached posts
func InvalidateNewsCache() {
	newsCacheMu.Lock()
	defer newsCacheMu.Unlock()
	slugCache = make(map[string]*NewsPost)
}

// --- Query Functions ---

func GetPublishedNewsPosts(page, pageSize int, category, tag, contentType string) ([]*NewsPost, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := DB.Model(&NewsPost{}).Where("status = ?", NewsStatusPublished)
	if category != "" {
		query = query.Where("seo_keywords LIKE ?", "%"+category+"%")
	}
	if tag != "" {
		query = query.Where("seo_keywords LIKE ?", "%"+tag+"%")
	}
	if contentType != "" {
		query = query.Where("content_type = ?", contentType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []*NewsPost
	err := query.Order("published_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

func GetNewsPostBySlug(slug string) (*NewsPost, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, errors.New("empty slug")
	}

	newsCacheMu.RLock()
	if cached, ok := slugCache[slug]; ok {
		newsCacheMu.RUnlock()
		return cached, nil
	}
	newsCacheMu.RUnlock()

	var post NewsPost
	err := DB.Where("slug = ? AND status = ?", slug, NewsStatusPublished).First(&post).Error
	if err != nil {
		return nil, err
	}

	newsCacheMu.Lock()
	slugCache[slug] = &post
	newsCacheMu.Unlock()

	return &post, nil
}

func IncrementNewsPostView(id int) error {
	if DB == nil {
		return nil
	}
	return DB.Model(&NewsPost{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func GetNewsPostById(id int) (*NewsPost, error) {
	var post NewsPost
	if err := DB.First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// GetBangkokDateRange returns UTC unix timestamps for 00:00:00 (inclusive) and 24:00:00 (exclusive) in Asia/Bangkok (UTC+7) (Section 3)
func GetBangkokDateRange(t time.Time) (int64, int64, string) {
	loc := time.FixedZone("Asia/Bangkok", 7*3600)
	local := t.In(loc)
	startOfDay := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	nextDay := startOfDay.AddDate(0, 0, 1)
	dateStr := startOfDay.Format("2006-01-02")
	return startOfDay.Unix(), nextDay.Unix(), dateStr
}

// NewsPublicationCounters exposes authoritative, distinct publication metrics (Section 4)
type NewsPublicationCounters struct {
	TotalPublishedPosts      int `json:"total_published_posts"`
	PublishedTodayTotal      int `json:"published_today_total"`
	AutopilotPublishedToday  int `json:"autopilot_published_today"`
	ManualPublishedToday     int `json:"manual_published_today"`
	SeedOrHistoricalToday    int `json:"seed_or_historical_today"`
	ImportedToday            int `json:"imported_today"`
	LegacyUnknownToday       int `json:"legacy_unknown_today"`
	DraftsToday              int `json:"drafts_today"`
	ReviewsToday             int `json:"reviews_today"`
}

// GetAuthoritativePublicationCounters calculates distinct publication counters (Section 4)
func GetAuthoritativePublicationCounters() (*NewsPublicationCounters, error) {
	if DB == nil {
		return &NewsPublicationCounters{}, nil
	}
	start, end, dateStr := GetBangkokDateRange(time.Now())

	var totalPublished int64
	_ = DB.Model(&NewsPost{}).Where("status = ?", NewsStatusPublished).Count(&totalPublished).Error

	var publishedTodayTotal int64
	_ = DB.Model(&NewsPost{}).Where("status = ? AND published_at >= ? AND published_at < ?", NewsStatusPublished, start, end).Count(&publishedTodayTotal).Error

	// Autopilot quota for today (from atomic ledger or publication events)
	var autoQuota NewsAutopilotDailyQuota
	if err := DB.Where("bangkok_date = ?", dateStr).First(&autoQuota).Error; err != nil {
		var eventCount int64
		_ = DB.Model(&NewsPublicationEvent{}).
			Where("bangkok_publication_date = ? AND event_type = ?", dateStr, EventTypeAutopilotInitialPublish).
			Count(&eventCount).Error
		autoQuota.PublishedCount = int(eventCount)
	}

	var manualToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND published_at >= ? AND published_at < ? AND publication_origin = ?", NewsStatusPublished, start, end, PublicationOriginManualAdmin).
		Count(&manualToday).Error

	var seedToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND published_at >= ? AND published_at < ? AND (is_seed = true OR publication_origin = ?)", NewsStatusPublished, start, end, PublicationOriginSeed).
		Count(&seedToday).Error

	var importedToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND published_at >= ? AND published_at < ? AND publication_origin = ?", NewsStatusPublished, start, end, PublicationOriginImport).
		Count(&importedToday).Error

	var legacyToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND published_at >= ? AND published_at < ? AND is_seed = false AND (publication_origin = ? OR publication_origin = '' OR publication_origin IS NULL)", NewsStatusPublished, start, end, PublicationOriginUnknownLegacy).
		Count(&legacyToday).Error

	var draftsToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", NewsStatusDraft, start, end).
		Count(&draftsToday).Error

	var reviewsToday int64
	_ = DB.Model(&NewsPost{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", NewsStatusReviewRequired, start, end).
		Count(&reviewsToday).Error

	return &NewsPublicationCounters{
		TotalPublishedPosts:     int(totalPublished),
		PublishedTodayTotal:     int(publishedTodayTotal),
		AutopilotPublishedToday: autoQuota.PublishedCount,
		ManualPublishedToday:    int(manualToday),
		SeedOrHistoricalToday:   int(seedToday),
		ImportedToday:           int(importedToday),
		LegacyUnknownToday:      int(legacyToday),
		DraftsToday:             int(draftsToday),
		ReviewsToday:            int(reviewsToday),
	}, nil
}

// GetAutopilotPublishedToday returns the authoritative autonomous publication count for today (Section 4)
func GetAutopilotPublishedToday() (int, error) {
	counters, err := GetAuthoritativePublicationCounters()
	if err != nil {
		return 0, err
	}
	return counters.AutopilotPublishedToday, nil
}

// GetPublishedCountToday returns the authoritative autonomous publication count for today (Section 4)
func GetPublishedCountToday() (int, error) {
	return GetAutopilotPublishedToday()
}

// GetPublishedCountBySourceToday counts autonomous posts published today for a specific primary source (Section 10)
func GetPublishedCountBySourceToday(sourceId int) (int, error) {
	if DB == nil {
		return 0, nil
	}
	_, _, dateStr := GetBangkokDateRange(time.Now())
	var count int64
	err := DB.Model(&NewsPublicationEvent{}).
		Where("source_id = ? AND bangkok_publication_date = ? AND event_type = ?", sourceId, dateStr, EventTypeAutopilotInitialPublish).
		Count(&count).Error
	return int(count), err
}

// AutopilotPublishParams defines atomic publication parameters (Section 7, 8, 9, 10)
type AutopilotPublishParams struct {
	Post            *NewsPost
	ClusterId       int
	PrimarySourceId int
	AutopilotRunId  string
	BatchId         string
	MaxDailyCap     int
	MaxSourceCap    int
}

// AutopilotPublishResult defines the atomic publication result
type AutopilotPublishResult struct {
	Post         *NewsPost
	Success      bool
	RejectReason string
	Event        *NewsPublicationEvent
}

// PublishPostWithAtomicQuota publishes a post atomically under row-level database lock (Section 7 & 8)
func PublishPostWithAtomicQuota(params *AutopilotPublishParams) (*AutopilotPublishResult, error) {
	if DB == nil {
		return nil, errors.New("db not initialized")
	}

	maxDaily := params.MaxDailyCap
	if maxDaily <= 0 {
		maxDaily = 20
	}
	maxSource := params.MaxSourceCap
	if maxSource <= 0 {
		maxSource = 3
	}

	now := time.Now()
	nowUnix := now.Unix()
	_, _, dateStr := GetBangkokDateRange(now)

	var pubResult *AutopilotPublishResult

	err := DB.Transaction(func(tx *gorm.DB) error {
		// 1. Lock daily quota row atomically using SELECT ... FOR UPDATE (via lockForUpdate helper)
		var quota NewsAutopilotDailyQuota
		err := lockForUpdate(tx).Where("bangkok_date = ?", dateStr).First(&quota).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			quota = NewsAutopilotDailyQuota{
				BangkokDate:     dateStr,
				PublishedCount:  0,
				LastPublishedAt: 0,
				UpdatedAt:       nowUnix,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&quota).Error; err != nil {
				return err
			}
			if err := lockForUpdate(tx).Where("bangkok_date = ?", dateStr).First(&quota).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		// 2. Enforce atomic daily cap
		if quota.PublishedCount >= maxDaily {
			if params.Post != nil && params.Post.Id == 0 {
				params.Post.Status = NewsStatusDraft
				_ = tx.Create(params.Post).Error
			}
			pubResult = &AutopilotPublishResult{
				Success:      false,
				RejectReason: fmt.Sprintf("Daily autopilot quota reached (%d/%d for %s)", quota.PublishedCount, maxDaily, dateStr),
			}
			return nil
		}

		// 3. Enforce primary source daily cap (Section 10)
		if params.PrimarySourceId > 0 {
			var sourcePublishedCount int64
			if err := tx.Model(&NewsPublicationEvent{}).
				Where("bangkok_publication_date = ? AND source_id = ? AND event_type = ?", dateStr, params.PrimarySourceId, EventTypeAutopilotInitialPublish).
				Count(&sourcePublishedCount).Error; err != nil {
				return err
			}
			if int(sourcePublishedCount) >= maxSource {
				if params.Post != nil && params.Post.Id == 0 {
					params.Post.Status = NewsStatusDraft
					_ = tx.Create(params.Post).Error
				}
				pubResult = &AutopilotPublishResult{
					Success:      false,
					RejectReason: fmt.Sprintf("Source daily cap reached (%d/%d for source %d on %s)", sourcePublishedCount, maxSource, params.PrimarySourceId, dateStr),
				}
				return nil
			}
		}

		// 4. Ensure cluster does not already have an existing post
		if params.ClusterId > 0 {
			var existingPost NewsPost
			if err := tx.Where("cluster_id = ?", params.ClusterId).First(&existingPost).Error; err == nil {
				pubResult = &AutopilotPublishResult{
					Success:      false,
					RejectReason: fmt.Sprintf("Post already exists for cluster %d (post id %d)", params.ClusterId, existingPost.Id),
				}
				return nil
			}
		}

		// 5. Create or save the post with AUTOPILOT origin
		post := params.Post
		post.PublicationOrigin = PublicationOriginAutopilot
		post.Status = NewsStatusPublished
		post.PublishedAt = nowUnix
		if post.CreatedAt == 0 {
			post.CreatedAt = nowUnix
		}
		post.UpdatedAt = nowUnix

		if post.Id > 0 {
			if err := tx.Save(post).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(post).Error; err != nil {
				return err
			}
		}

		// 6. Record immutable publication event (Section 6)
		event := &NewsPublicationEvent{
			PostId:                 post.Id,
			PublicationOrigin:      PublicationOriginAutopilot,
			EventType:              EventTypeAutopilotInitialPublish,
			PublishedAt:            nowUnix,
			BangkokPublicationDate: dateStr,
			SourceId:               params.PrimarySourceId,
			StoryClusterId:         params.ClusterId,
			AutopilotRunId:         params.AutopilotRunId,
			BatchId:                params.BatchId,
			CreatedAt:              nowUnix,
		}
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		// 7. Increment daily quota
		if err := tx.Model(&NewsAutopilotDailyQuota{}).
			Where("bangkok_date = ?", dateStr).
			Updates(map[string]interface{}{
				"published_count":   gorm.Expr("published_count + 1"),
				"last_published_at": nowUnix,
				"updated_at":        nowUnix,
			}).Error; err != nil {
			return err
		}

		pubResult = &AutopilotPublishResult{
			Post:    post,
			Success: true,
			Event:   event,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if pubResult != nil && pubResult.Success {
		InvalidateNewsCache()
	}
	return pubResult, nil
}

// GetTodayPublishedPosts returns all posts published today in Asia/Bangkok time
func GetTodayPublishedPosts() ([]*NewsPost, error) {
	if DB == nil {
		return nil, nil
	}
	start, end, _ := GetBangkokDateRange(time.Now())
	var posts []*NewsPost
	err := DB.Model(&NewsPost{}).
		Where("status = ? AND published_at >= ? AND published_at < ?", NewsStatusPublished, start, end).
		Order("published_at DESC").
		Find(&posts).Error
	return posts, err
}

func CreateNewsPost(post *NewsPost) error {
	if err := DB.Create(post).Error; err != nil {
		return err
	}
	InvalidateNewsCache()
	return nil
}

func UpdateNewsPost(post *NewsPost) error {
	if err := DB.Save(post).Error; err != nil {
		return err
	}
	InvalidateNewsCache()
	return nil
}

func DeleteNewsPost(id int) error {
	if err := DB.Delete(&NewsPost{}, id).Error; err != nil {
		return err
	}
	InvalidateNewsCache()
	return nil
}

func GetAllNewsPosts(page, pageSize int, status, keyword string) ([]*NewsPost, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := DB.Model(&NewsPost{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR summary LIKE ? OR slug LIKE ?", kw, kw, kw)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []*NewsPost
	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

// --- News Source Queries ---

func GetAllNewsSources(enabledOnly bool) ([]*NewsSource, error) {
	query := DB.Model(&NewsSource{})
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	var sources []*NewsSource
	err := query.Order("id ASC").Find(&sources).Error
	return sources, err
}

func GetNewsSourceById(id int) (*NewsSource, error) {
	var src NewsSource
	if err := DB.First(&src, id).Error; err != nil {
		return nil, err
	}
	return &src, nil
}

func CreateNewsSource(src *NewsSource) error {
	return DB.Create(src).Error
}

func UpdateNewsSource(src *NewsSource) error {
	return DB.Save(src).Error
}

func DeleteNewsSource(id int) error {
	return DB.Delete(&NewsSource{}, id).Error
}

// InitDefaultNewsSources seeds authoritative primary technical news sources (Section 2)
func InitDefaultNewsSources() error {
	defaults := []NewsSource{
		{
			Name:                   "OpenAI Newsroom",
			Slug:                   "openai-official",
			FeedUrl:                "https://openai.com/news/rss.xml",
			SiteUrl:                "https://openai.com/news",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 30,
			Tags:                   "openai,gpt,chatgpt,api",
		},
		{
			Name:                   "Google AI Blog",
			Slug:                   "google-ai-official",
			FeedUrl:                "https://blog.google/technology/ai/rss/",
			SiteUrl:                "https://blog.google/technology/ai/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "google,gemini,gemma,deepmind",
		},
		{
			Name:                   "Google DeepMind",
			Slug:                   "google-deepmind",
			FeedUrl:                "https://deepmind.google/blog/rss.xml",
			SiteUrl:                "https://deepmind.google/discover/blog/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "deepmind,alphafold,gemini,research",
		},
		{
			Name:                   "Google Research",
			Slug:                   "google-research",
			FeedUrl:                "https://research.google/blog/rss/",
			SiteUrl:                "https://research.google/blog/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "google,research,ml,algorithms",
		},
		{
			Name:                   "Google Developers Blog",
			Slug:                   "google-developers",
			FeedUrl:                "https://developers.googleblog.com/feeds/posts/default",
			SiteUrl:                "https://developers.googleblog.com/",
			SourceType:             "atom",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "google,android,web,cloud,developer",
		},
		{
			Name:                   "AWS Machine Learning Blog",
			Slug:                   "aws-ml-official",
			FeedUrl:                "https://aws.amazon.com/blogs/machine-learning/feed/",
			SiteUrl:                "https://aws.amazon.com/blogs/machine-learning/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "aws,bedrock,sagemaker,cloud",
		},
		{
			Name:                   "NVIDIA Technical Blog",
			Slug:                   "nvidia-developer",
			FeedUrl:                "https://developer.nvidia.com/blog/feed",
			SiteUrl:                "https://developer.nvidia.com/blog",
			SourceType:             "atom",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "nvidia,gpu,cuda,nim,inference",
		},
		{
			Name:                   "Apple Machine Learning Research",
			Slug:                   "apple-ml-research",
			FeedUrl:                "https://machinelearning.apple.com/rss.xml",
			SiteUrl:                "https://machinelearning.apple.com/research",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "apple,coreml,apple-intelligence,research",
		},
		{
			Name:                   "Microsoft Official Blog - AI",
			Slug:                   "microsoft-official",
			FeedUrl:                "https://blogs.microsoft.com/feed/",
			SiteUrl:                "https://blogs.microsoft.com/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "microsoft,azure,copilot,openai",
		},
		{
			Name:                   "Hugging Face Blog",
			Slug:                   "hugging-face-blog",
			FeedUrl:                "https://huggingface.co/blog/feed.xml",
			SiteUrl:                "https://huggingface.co/blog",
			SourceType:             "atom",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "huggingface,transformers,open-source,models",
		},
		{
			Name:                   "MIT News - Artificial Intelligence",
			Slug:                   "mit-news-ai",
			FeedUrl:                "https://news.mit.edu/topic/mitartificial-intelligence2-rss.xml",
			SiteUrl:                "https://news.mit.edu/topic/artificial-intelligence2",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 120,
			Tags:                   "mit,research,academia,breakthroughs",
		},
		{
			Name:                   "Berkeley AI Research (BAIR)",
			Slug:                   "bair-blog",
			FeedUrl:                "https://bair.berkeley.edu/blog/feed.xml",
			SiteUrl:                "https://bair.berkeley.edu/blog/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 120,
			Tags:                   "berkeley,bair,robotics,rl,llm",
		},
		{
			Name:                   "GitHub Blog",
			Slug:                   "github-blog",
			FeedUrl:                "https://github.blog/feed/",
			SiteUrl:                "https://github.blog/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "github,engineering,git,devops",
		},
		{
			Name:                   "GitHub Changelog",
			Slug:                   "github-changelog",
			FeedUrl:                "https://github.blog/changelog/feed/",
			SiteUrl:                "https://github.blog/changelog/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "github,changelog,features,releases",
		},
		{
			Name:                   "GitHub Copilot Changelog",
			Slug:                   "github-copilot-changelog",
			FeedUrl:                "https://github.blog/tag/github-copilot/feed/",
			SiteUrl:                "https://github.blog/tag/github-copilot/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "copilot,ai-coding,agents,github",
		},
		{
			Name:                   "Cloudflare Blog",
			Slug:                   "cloudflare-blog",
			FeedUrl:                "https://blog.cloudflare.com/rss/",
			SiteUrl:                "https://blog.cloudflare.com/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "cloudflare,workers,workers-ai,edge,cdn",
		},
		{
			Name:                   "Cohere Blog",
			Slug:                   "cohere-blog",
			FeedUrl:                "https://docs.cohere.com/changelog.rss",
			SiteUrl:                "https://docs.cohere.com/changelog",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "cohere,command,rerank,embeddings",
		},
		{
			Name:                   "Replicate Blog",
			Slug:                   "replicate-blog",
			FeedUrl:                "https://replicate.com/blog/rss",
			SiteUrl:                "https://replicate.com/blog",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "replicate,diffusion,open-source,inference",
		},
		{
			Name:                   "Amazon Science",
			Slug:                   "amazon-science",
			FeedUrl:                "https://www.amazon.science/index.rss",
			SiteUrl:                "https://www.amazon.science/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 120,
			Tags:                   "amazon,science,deep-learning,nlp",
		},
		{
			Name:                   "Carnegie Mellon ML Blog",
			Slug:                   "cmu-ml-blog",
			FeedUrl:                "https://blog.ml.cmu.edu/feed/",
			SiteUrl:                "https://blog.ml.cmu.edu/",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 120,
			Tags:                   "cmu,machine-learning,research,academia",
		},
		{
			Name:                   "Anthropic Research & News",
			Slug:                   "anthropic-official",
			FeedUrl:                "https://www.anthropic.com/news",
			SiteUrl:                "https://www.anthropic.com/news",
			SourceType:             "html_watch",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 30,
			Tags:                   "anthropic,claude,api,safety",
		},
		{
			Name:                   "DeepSeek Releases",
			Slug:                   "deepseek-official",
			FeedUrl:                "https://github.com/deepseek-ai/DeepSeek-V3/releases.atom",
			SiteUrl:                "https://github.com/deepseek-ai/DeepSeek-V3",
			SourceType:             "github_release",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "deepseek,v3,r1,open-source",
		},
		{
			Name:                   "OpenRouter Announcements",
			Slug:                   "openrouter-official",
			FeedUrl:                "https://openrouter.ai/docs/changelog/rss.xml",
			SiteUrl:                "https://openrouter.ai/docs/changelog",
			SourceType:             "rss",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 60,
			Tags:                   "openrouter,routing,pricing",
		},
		{
			Name:                   "New-API Core Releases",
			Slug:                   "new-api-upstream",
			FeedUrl:                "https://github.com/QuantumNous/new-api/releases.atom",
			SiteUrl:                "https://github.com/QuantumNous/new-api",
			SourceType:             "github_release",
			TrustTier:              "tier_1_official",
			Enabled:                true,
			PollingIntervalMinutes: 120,
			Tags:                   "new-api,gateway,backend",
		},
	}

	for _, src := range defaults {
		var existing NewsSource
		err := DB.Where("slug = ?", src.Slug).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s := src
			if err := DB.Create(&s).Error; err != nil {
				return err
			}
			common.SysLog(fmt.Sprintf("seeded authoritative news source: %s", src.Name))
		} else if err == nil {
			// Ensure FeedUrl and SourceType are updated if changed
			if existing.FeedUrl != src.FeedUrl || existing.SourceType != src.SourceType {
				existing.FeedUrl = src.FeedUrl
				existing.SourceType = src.SourceType
				_ = DB.Save(&existing)
			}
		}
	}
	return nil
}

// --- Story Clusters ---

func CreateStoryCluster(cluster *StoryCluster) error {
	return DB.Create(cluster).Error
}

func GetActiveStoryClusters(limit int) ([]*StoryCluster, error) {
	if limit <= 0 {
		limit = 50
	}
	var clusters []*StoryCluster
	err := DB.Model(&StoryCluster{}).Order("last_event_at DESC").Limit(limit).Find(&clusters).Error
	return clusters, err
}

// --- Distributions ---

func CreateNewsDistribution(dist *NewsDistribution) error {
	return DB.Create(dist).Error
}

func GetDistributionsByPostId(postId int) ([]*NewsDistribution, error) {
	var dists []*NewsDistribution
	err := DB.Where("post_id = ?", postId).Order("id ASC").Find(&dists).Error
	return dists, err
}

func GetDistributionByPostPlatformAndVersion(postId int, platform string, version int) (*NewsDistribution, error) {
	var dist NewsDistribution
	err := DB.Where("post_id = ? AND platform = ? AND distribution_version = ?", postId, platform, version).First(&dist).Error
	if err != nil {
		return nil, err
	}
	return &dist, nil
}

// --- Analytics ---

func RecordNewsAnalyticEvent(evt *NewsAnalyticEvent) error {
	if DB == nil || evt == nil {
		return nil
	}
	if evt.CreatedAt == 0 {
		evt.CreatedAt = common.GetTimestamp()
	}
	return DB.Create(evt).Error
}

func GetNewsAnalyticsSummary(postId int) (map[string]int64, error) {
	type result struct {
		EventType string
		Count     int64
	}
	var res []result
	query := DB.Model(&NewsAnalyticEvent{})
	if postId > 0 {
		query = query.Where("post_id = ?", postId)
	}
	err := query.Select("event_type, count(*) as count").Group("event_type").Scan(&res).Error
	if err != nil {
		return nil, err
	}
	summary := make(map[string]int64)
	for _, r := range res {
		summary[r.EventType] = r.Count
	}
	return summary, nil
}

func UpdateNewsDistribution(dist *NewsDistribution) error {
	return DB.Save(dist).Error
}

func GetPendingNewsDistributions(limit int) ([]*NewsDistribution, error) {
	if limit <= 0 {
		limit = 20
	}
	var dists []*NewsDistribution
	err := DB.Where("status = ?", "pending").Order("id ASC").Limit(limit).Find(&dists).Error
	return dists, err
}

// --- SEO Metrics & Opportunities Models ---

// NewsSeoMetric records Search Console impressions, clicks, CTR, and positions.
type NewsSeoMetric struct {
	Id           int     `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId       int     `json:"post_id" gorm:"index;default:0"`
	PageUrl      string  `json:"page_url" gorm:"type:varchar(512);index;not null"`
	Query        string  `json:"query" gorm:"type:varchar(255);index;not null"`
	Country      string  `json:"country" gorm:"type:varchar(8);default:''"`
	Device       string  `json:"device" gorm:"type:varchar(16);default:''"`
	Clicks       int     `json:"clicks" gorm:"default:0"`
	Impressions  int     `json:"impressions" gorm:"default:0"`
	Ctr          float64 `json:"ctr" gorm:"type:float;default:0"`
	Position     float64 `json:"position" gorm:"type:float;default:0"`
	SnapshotDate string  `json:"snapshot_date" gorm:"type:varchar(32);index;not null"`
	DataState    string  `json:"data_state" gorm:"type:varchar(16);default:'FINAL'"` // FINAL or PARTIAL
	IsFinal      bool    `json:"is_final" gorm:"default:true"`
	CreatedAt    int64   `json:"created_at" gorm:"bigint"`
}

func (m *NewsSeoMetric) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAt == 0 {
		m.CreatedAt = common.GetTimestamp()
	}
	if m.DataState == "" {
		m.DataState = "FINAL"
		m.IsFinal = true
	}
	return nil
}

func SaveNewsSeoMetric(m *NewsSeoMetric) error {
	if DB == nil || m == nil {
		return nil
	}
	return DB.Create(m).Error
}

func GetLatestFinalizedSeoMetricDate() (string, error) {
	if DB == nil {
		return "", nil
	}
	var maxDate string
	err := DB.Model(&NewsSeoMetric{}).Where("is_final = ?", true).Select("COALESCE(MAX(snapshot_date), '')").Scan(&maxDate).Error
	return maxDate, err
}

func GetNewsSeoMetricsByPostId(postId int, limit int) ([]*NewsSeoMetric, error) {
	if limit <= 0 {
		limit = 50
	}
	var metrics []*NewsSeoMetric
	err := DB.Where("post_id = ?", postId).Order("snapshot_date DESC").Limit(limit).Find(&metrics).Error
	return metrics, err
}

func GetRecentNewsSeoMetrics(sinceDate string) ([]*NewsSeoMetric, error) {
	var metrics []*NewsSeoMetric
	query := DB.Model(&NewsSeoMetric{})
	if sinceDate != "" {
		query = query.Where("snapshot_date >= ?", sinceDate)
	}
	err := query.Order("snapshot_date DESC, impressions DESC").Find(&metrics).Error
	return metrics, err
}

func CreateNewsSeoMetric(m *NewsSeoMetric) error {
	return SaveNewsSeoMetric(m)
}

const (
	OpportunityHighImpLowCTR           = "HIGH_IMPRESSION_LOW_CTR"
	OpportunityHighImpLowCTRAlt        = "HIGH_IMPRESSIONS_LOW_CTR"
	OpportunityPosition8To20           = "POSITION_8_TO_20"
	OpportunityPosition5To20           = "POSITION_5_TO_20"
	OpportunityRisingQuery             = "RISING_QUERY"
	OpportunityContentDecay            = "CONTENT_DECAY"
	OpportunityQueryNoLanding          = "QUERY_WITHOUT_GOOD_LANDING_PAGE"
	OpportunityCanonicalMismatch       = "CANONICAL_MISMATCH"
	OpportunityIndexingAnomaly         = "INDEXING_ANOMALY"
	OpportunityIndexingProblem         = "INDEXING_PROBLEM"
	OpportunityNewQuery                = "NEW_QUERY_OPPORTUNITY"
	OpportunityCannibalization         = "CANNIBALIZATION"
	OpportunityInternalLink            = "INTERNAL_LINK_OPPORTUNITY"
	OpportunityCtrUnderperform         = "CTR_UNDERPERFORMANCE"
)

// NewsUrlInspection stores Google Search Console URL inspection results
type NewsUrlInspection struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId          int    `json:"post_id" gorm:"index;not null"`
	InspectionUrl   string `json:"inspection_url" gorm:"type:varchar(512);index;not null"`
	Verdict         string `json:"verdict" gorm:"type:varchar(32);not null"` // PASS, NEUTRAL, FAIL
	CoverageState   string `json:"coverage_state" gorm:"type:varchar(128);default:''"`
	RobotsTxtState  string `json:"robots_txt_state" gorm:"type:varchar(64);default:''"`
	IndexingState   string `json:"indexing_state" gorm:"type:varchar(64);default:''"`
	LastCrawlTime   string `json:"last_crawl_time" gorm:"type:varchar(64);default:''"`
	PageFetchState  string `json:"page_fetch_state" gorm:"type:varchar(64);default:''"`
	GoogleCanonical string `json:"google_canonical" gorm:"type:varchar(512);default:''"`
	UserCanonical   string `json:"user_canonical" gorm:"type:varchar(512);default:''"`
	InspectionTime  int64  `json:"inspection_time" gorm:"bigint;index"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint"`
}

func (u *NewsUrlInspection) BeforeCreate(tx *gorm.DB) error {
	if u.CreatedAt == 0 {
		u.CreatedAt = common.GetTimestamp()
	}
	if u.InspectionTime == 0 {
		u.InspectionTime = u.CreatedAt
	}
	return nil
}

func CreateNewsUrlInspection(insp *NewsUrlInspection) error {
	if DB == nil || insp == nil {
		return nil
	}
	return DB.Create(insp).Error
}

func GetLatestUrlInspectionByPostId(postId int) (*NewsUrlInspection, error) {
	var insp NewsUrlInspection
	err := DB.Where("post_id = ?", postId).Order("inspection_time DESC").First(&insp).Error
	if err != nil {
		return nil, err
	}
	return &insp, nil
}

// NewsSeoExperiment tracks an optimization change as an experiment with baseline & follow-up metrics
type NewsSeoExperiment struct {
	Id                  int     `json:"id" gorm:"primaryKey;autoIncrement"`
	OpportunityId       int     `json:"opportunity_id" gorm:"index;default:0"`
	PostId              int     `json:"post_id" gorm:"index;not null"`
	ChangeType          string  `json:"change_type" gorm:"type:varchar(64);not null"` // title, meta_description, faq_section, etc.
	BeforeValue         string  `json:"before_value" gorm:"type:text"`
	AfterValue          string  `json:"after_value" gorm:"type:text"`
	AppliedAt           int64   `json:"applied_at" gorm:"bigint;not null"`
	BaselineImpressions int     `json:"baseline_impressions" gorm:"default:0"`
	BaselineClicks      int     `json:"baseline_clicks" gorm:"default:0"`
	BaselineCTR         float64 `json:"baseline_ctr" gorm:"type:float;default:0"`
	BaselinePosition    float64 `json:"baseline_position" gorm:"type:float;default:0"`
	Metrics7dJSON       string  `json:"metrics_7d_json" gorm:"type:text"`
	Metrics14dJSON      string  `json:"metrics_14d_json" gorm:"type:text"`
	Metrics28dJSON      string  `json:"metrics_28d_json" gorm:"type:text"`
	ResultVerdict       string  `json:"result_verdict" gorm:"type:varchar(32);default:'in_progress'"` // in_progress, improved, neutral, regressed
	CreatedAt           int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64   `json:"updated_at" gorm:"bigint"`
}

func (e *NewsSeoExperiment) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	e.CreatedAt = now
	e.UpdatedAt = now
	if e.AppliedAt == 0 {
		e.AppliedAt = now
	}
	return nil
}

func (e *NewsSeoExperiment) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = common.GetTimestamp()
	return nil
}

func CreateNewsSeoExperiment(exp *NewsSeoExperiment) error {
	if DB == nil || exp == nil {
		return nil
	}
	return DB.Create(exp).Error
}

func UpdateNewsSeoExperiment(exp *NewsSeoExperiment) error {
	if DB == nil || exp == nil {
		return nil
	}
	return DB.Save(exp).Error
}

func GetSeoExperimentsByPostId(postId int) ([]*NewsSeoExperiment, error) {
	var exps []*NewsSeoExperiment
	err := DB.Where("post_id = ?", postId).Order("applied_at DESC").Find(&exps).Error
	return exps, err
}

// NewsSeoOpportunity stores detected SEO opportunities and proposed remediations.
type NewsSeoOpportunity struct {
	Id                  int     `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId              int     `json:"post_id" gorm:"index;not null"`
	OpportunityType     string  `json:"opportunity_type" gorm:"type:varchar(64);index;not null"`
	Query               string  `json:"query" gorm:"type:varchar(255);default:''"`
	CurrentImpressions  int     `json:"current_impressions" gorm:"default:0"`
	CurrentClicks       int     `json:"current_clicks" gorm:"default:0"`
	CurrentCTR          float64 `json:"current_ctr" gorm:"type:float;default:0"`
	CurrentPosition     float64 `json:"current_position" gorm:"type:float;default:0"`
	Observation         string  `json:"observation" gorm:"type:text"`
	EvidenceJSON        string  `json:"evidence_json" gorm:"type:text"`
	Hypothesis          string  `json:"hypothesis" gorm:"type:text"`
	ProposedAction      string  `json:"proposed_action" gorm:"type:text"`
	ProposedChange      string  `json:"proposed_change" gorm:"type:text"`
	ProposedChangesDiff string  `json:"proposed_changes_diff" gorm:"type:text"`
	BeforeContent       string  `json:"before_content" gorm:"type:text"`
	AfterContent        string  `json:"after_content" gorm:"type:text"`
	RiskClass           string  `json:"risk_class" gorm:"type:varchar(32);default:'low'"`
	Status              string  `json:"status" gorm:"type:varchar(32);default:'detected';index"` // detected, applied, rejected, evaluated
	AppliedAt           int64   `json:"applied_at" gorm:"bigint;default:0"`
	CooldownExpiresAt   int64   `json:"cooldown_expires_at" gorm:"bigint;default:0"`
	CooldownUntil       int64   `json:"cooldown_until" gorm:"bigint;default:0"`
	CreatedAt           int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64   `json:"updated_at" gorm:"bigint"`
}

func (o *NewsSeoOpportunity) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	o.CreatedAt = now
	o.UpdatedAt = now
	if o.CooldownExpiresAt > 0 && o.CooldownUntil == 0 {
		o.CooldownUntil = o.CooldownExpiresAt
	}
	return nil
}

func (o *NewsSeoOpportunity) BeforeUpdate(tx *gorm.DB) error {
	o.UpdatedAt = common.GetTimestamp()
	return nil
}

func CreateNewsSeoOpportunity(opp *NewsSeoOpportunity) error {
	if DB == nil || opp == nil {
		return nil
	}
	return DB.Create(opp).Error
}

func UpdateNewsSeoOpportunity(opp *NewsSeoOpportunity) error {
	if DB == nil || opp == nil {
		return nil
	}
	return DB.Save(opp).Error
}

func GetNewsSeoOpportunityById(id int) (*NewsSeoOpportunity, error) {
	var opp NewsSeoOpportunity
	err := DB.Where("id = ?", id).First(&opp).Error
	if err != nil {
		return nil, err
	}
	return &opp, nil
}

func GetSeoOpportunitiesByPost(postId int) ([]*NewsSeoOpportunity, error) {
	var opps []*NewsSeoOpportunity
	err := DB.Where("post_id = ?", postId).Find(&opps).Error
	return opps, err
}

func GetActiveNewsSeoOpportunities(postId int) ([]*NewsSeoOpportunity, error) {
	var opps []*NewsSeoOpportunity
	query := DB.Where("status = ?", "detected")
	if postId > 0 {
		query = query.Where("post_id = ?", postId)
	}
	err := query.Order("id DESC").Find(&opps).Error
	return opps, err
}

func GetAllNewsSeoOpportunities(status string, limit int) ([]*NewsSeoOpportunity, error) {
	if limit <= 0 {
		limit = 50
	}
	var opps []*NewsSeoOpportunity
	query := DB.Model(&NewsSeoOpportunity{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("id DESC").Limit(limit).Find(&opps).Error
	return opps, err
}

// NewsConversionEvent tracks privacy-safe product funnel progression.
type NewsConversionEvent struct {
	Id           int     `json:"id" gorm:"primaryKey;autoIncrement"`
	PostId       int     `json:"post_id" gorm:"index;default:0"`
	EventType    string  `json:"event_type" gorm:"type:varchar(64);index;not null"` // organic_landing, social_landing, signup, api_key_created, first_successful_inference, topup, subscription
	UtmSource    string  `json:"utm_source" gorm:"type:varchar(64)"`
	UtmMedium    string  `json:"utm_medium" gorm:"type:varchar(64)"`
	UtmCampaign  string  `json:"utm_campaign" gorm:"type:varchar(64)"`
	UtmContent   string  `json:"utm_content" gorm:"type:varchar(64)"`
	ContentId    string  `json:"content_id" gorm:"type:varchar(64)"`
	VisitorHash  string  `json:"visitor_hash" gorm:"type:varchar(64);index"`
	UserId       int     `json:"user_id" gorm:"default:0;index"`
	RevenueTHB   float64 `json:"revenue_thb" gorm:"type:float;default:0"`
	MetadataJSON string  `json:"metadata_json" gorm:"type:text"`
	CreatedAt    int64   `json:"created_at" gorm:"bigint;index"`
}

func (c *NewsConversionEvent) BeforeCreate(tx *gorm.DB) error {
	if c.CreatedAt == 0 {
		c.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func RecordNewsConversionEvent(evt *NewsConversionEvent) error {
	if DB == nil || evt == nil {
		return nil
	}
	return DB.Create(evt).Error
}

func CreateNewsConversionEvent(evt *NewsConversionEvent) error {
	return RecordNewsConversionEvent(evt)
}

func GetConversionFunnelSummary(postId int) (map[string]int64, error) {
	type result struct {
		EventType string
		Count     int64
	}
	var res []result
	query := DB.Model(&NewsConversionEvent{})
	if postId > 0 {
		query = query.Where("post_id = ?", postId)
	}
	err := query.Select("event_type, count(*) as count").Group("event_type").Scan(&res).Error
	if err != nil {
		return nil, err
	}
	summary := make(map[string]int64)
	for _, r := range res {
		summary[r.EventType] = r.Count
	}
	return summary, nil
}

// NewsDailyGrowthReview persists daily recurring growth and SEO review reports (Section 23).
type NewsDailyGrowthReview struct {
	Id                     int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ReviewDate             string  `json:"review_date" gorm:"type:varchar(32);uniqueIndex;not null"` // YYYY-MM-DD
	PostsPublishedToday    int     `json:"posts_published_today" gorm:"default:0"`
	NewsCount              int     `json:"news_count" gorm:"default:0"`
	GuideCount             int     `json:"guide_count" gorm:"default:0"`
	AnalysisCount          int     `json:"analysis_count" gorm:"default:0"`
	ChangelogCount         int     `json:"changelog_count" gorm:"default:0"`
	TotalImpressions       int     `json:"total_impressions" gorm:"default:0"`
	TotalClicks            int     `json:"total_clicks" gorm:"default:0"`
	AverageCTR             float64 `json:"average_ctr" gorm:"type:float;default:0"`
	HighPriorityOppsCount  int     `json:"high_priority_opps_count" gorm:"default:0"`
	DistributionsPublished int     `json:"distributions_published" gorm:"default:0"`
	DistributionsFailed    int     `json:"distributions_failed" gorm:"default:0"`
	DistributionsBlocked   int     `json:"distributions_blocked" gorm:"default:0"`
	SignupsAttributed      int     `json:"signups_attributed" gorm:"default:0"`
	ConversionsAttributed  int     `json:"conversions_attributed" gorm:"default:0"`
	PipelineHealthStatus   string  `json:"pipeline_health_status" gorm:"type:varchar(32);default:'healthy'"`
	ReviewNotes            string  `json:"review_notes" gorm:"type:text"`

	// Section 23: Newsroom Autopilot Report metrics
	SourcesChecked          int    `json:"sources_checked" gorm:"default:0"`
	SourceFailures          int    `json:"source_failures" gorm:"default:0"`
	CandidateStories        int    `json:"candidate_stories" gorm:"default:0"`
	ClustersCreated         int    `json:"clusters_created" gorm:"default:0"`
	DuplicatesRemoved       int    `json:"duplicates_removed" gorm:"default:0"`
	DraftsRequiringReview   int    `json:"drafts_requiring_review" gorm:"default:0"`
	RejectedStoriesCount    int    `json:"rejected_stories_count" gorm:"default:0"`
	RejectedReasonsJSON     string `json:"rejected_reasons_json" gorm:"type:text"`
	CategorySourceBreakdown string `json:"category_source_breakdown" gorm:"type:text"`
	MainSitemapStatus       string `json:"main_sitemap_status" gorm:"type:varchar(64);default:'ok'"`
	NewsSitemapStatus       string `json:"news_sitemap_status" gorm:"type:varchar(64);default:'ok'"`
	IndexNowStatus          string `json:"indexnow_status" gorm:"type:varchar(64);default:'idle'"`
	DevToStatus             string `json:"devto_status" gorm:"type:varchar(64);default:'active'"`
	AiVisibilitySummary     string `json:"ai_visibility_summary" gorm:"type:text"`
	TopOpportunitiesTomorrow string `json:"top_opportunities_tomorrow" gorm:"type:text"`
	NewsroomRunStatus       string `json:"newsroom_run_status" gorm:"type:varchar(64);default:'NEWSROOM RUN COMPLETE'"`

	CreatedAt              int64   `json:"created_at" gorm:"bigint"`
}

func (r *NewsDailyGrowthReview) BeforeCreate(tx *gorm.DB) error {
	if r.CreatedAt == 0 {
		r.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func CreateNewsDailyGrowthReview(review *NewsDailyGrowthReview) error {
	if DB == nil || review == nil {
		return nil
	}
	return DB.Create(review).Error
}

func CreateOrUpdateDailyGrowthReview(review *NewsDailyGrowthReview) error {
	if DB == nil || review == nil {
		return nil
	}
	var existing NewsDailyGrowthReview
	err := DB.Where("review_date = ?", review.ReviewDate).First(&existing).Error
	if err == nil {
		review.Id = existing.Id
		return DB.Save(review).Error
	}
	return DB.Create(review).Error
}

func GetNewsDailyGrowthReview(date string) (*NewsDailyGrowthReview, error) {
	var review NewsDailyGrowthReview
	err := DB.Where("review_date = ?", date).First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func GetLatestNewsDailyGrowthReviews(limit int) ([]*NewsDailyGrowthReview, error) {
	if limit <= 0 {
		limit = 30
	}
	var reviews []*NewsDailyGrowthReview
	err := DB.Order("review_date DESC").Limit(limit).Find(&reviews).Error
	return reviews, err
}

func GetRecentDailyGrowthReviews(limit int) ([]*NewsDailyGrowthReview, error) {
	return GetLatestNewsDailyGrowthReviews(limit)
}
