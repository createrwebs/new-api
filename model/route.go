package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RouteKindLLM   = "llm"
	RouteKindMedia = "media"
	RouteKindBYOK  = "byok"

	RoutePolicyPriority = "priority"
	RoutePolicyWeighted = "weighted"
)

var (
	ErrRouteNotFound         = errors.New("route not found")
	ErrRouteSlugExists       = errors.New("route slug already exists")
	ErrRouteReferencedByKeys = errors.New("route is referenced by active API keys and cannot be deleted")
	ErrInvalidRouteMultiplier = errors.New("cost multiplier must be greater than 0 and less than or equal to 10.0")
)

// Route represents a first-class routing policy and channel pool entity.
// It is completely decoupled from User / Subscription Groups (commercial entitlement).
type Route struct {
	Id             int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string         `json:"name" gorm:"type:varchar(64);not null;index"`
	Slug           string         `json:"slug" gorm:"type:varchar(64);uniqueIndex;not null"`
	Description    string         `json:"description" gorm:"type:text"`
	Enabled        bool           `json:"enabled" gorm:"default:true;index"`
	Kind           string         `json:"kind" gorm:"type:varchar(32);default:'llm';index"` // 'llm', 'media', 'byok'
	RoutingPolicy  string         `json:"routing_policy" gorm:"type:varchar(32);default:'priority'"` // 'priority', 'weighted'
	ChannelIds     string         `json:"channel_ids" gorm:"type:text;not null"` // JSON array of int channel IDs: [1, 5, 12]
	ChannelTags    string         `json:"channel_tags" gorm:"type:text"` // JSON array of string tags: ["fast", "us-east"]
	ModelMapping   string         `json:"model_mapping" gorm:"type:text"` // JSON object mapping: {"gpt-4":"gpt-4o"}
	CostMultiplier float64        `json:"cost_multiplier" gorm:"type:decimal(5,4);default:1.0000"` // Route billing multiplier
	MinUserGroup   string         `json:"min_user_group" gorm:"type:varchar(32);default:''"` // e.g., 'pro', or '' for all
	CreatedAt      int64          `json:"created_at" gorm:"bigint"`
	UpdatedAt      int64          `json:"updated_at" gorm:"bigint"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (r *Route) GetChannelIds() ([]int, error) {
	if strings.TrimSpace(r.ChannelIds) == "" {
		return []int{}, nil
	}
	var ids []int
	if err := common.UnmarshalJsonStr(r.ChannelIds, &ids); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel_ids for route %d: %w", r.Id, err)
	}
	return ids, nil
}

func (r *Route) SetChannelIds(ids []int) error {
	if len(ids) == 0 {
		r.ChannelIds = "[]"
		return nil
	}
	// Deduplicate channel IDs
	seen := make(map[int]struct{}, len(ids))
	cleanIds := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			cleanIds = append(cleanIds, id)
		}
	}
	data, err := common.Marshal(cleanIds)
	if err != nil {
		return err
	}
	r.ChannelIds = string(data)
	return nil
}

func (r *Route) GetChannelTags() ([]string, error) {
	if strings.TrimSpace(r.ChannelTags) == "" {
		return []string{}, nil
	}
	var tags []string
	if err := common.UnmarshalJsonStr(r.ChannelTags, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel_tags for route %d: %w", r.Id, err)
	}
	return tags, nil
}

func (r *Route) SetChannelTags(tags []string) error {
	if len(tags) == 0 {
		r.ChannelTags = "[]"
		return nil
	}
	cleanTags := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			if _, ok := seen[trimmed]; !ok {
				seen[trimmed] = struct{}{}
				cleanTags = append(cleanTags, trimmed)
			}
		}
	}
	data, err := common.Marshal(cleanTags)
	if err != nil {
		return err
	}
	r.ChannelTags = string(data)
	return nil
}

func (r *Route) GetModelMapping() (map[string]string, error) {
	if strings.TrimSpace(r.ModelMapping) == "" {
		return make(map[string]string), nil
	}
	var mapping map[string]string
	if err := common.UnmarshalJsonStr(r.ModelMapping, &mapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model_mapping for route %d: %w", r.Id, err)
	}
	return mapping, nil
}

func (r *Route) SetModelMapping(mapping map[string]string) error {
	if len(mapping) == 0 {
		r.ModelMapping = "{}"
		return nil
	}
	data, err := common.Marshal(mapping)
	if err != nil {
		return err
	}
	r.ModelMapping = string(data)
	return nil
}

func (r *Route) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("route name is required")
	}
	r.Slug = strings.TrimSpace(r.Slug)
	if r.Slug == "" {
		return errors.New("route slug is required")
	}
	if r.CostMultiplier <= 0 || r.CostMultiplier > 10.0 {
		return ErrInvalidRouteMultiplier
	}
	if r.Kind == "" {
		r.Kind = RouteKindLLM
	}
	if r.RoutingPolicy == "" {
		r.RoutingPolicy = RoutePolicyPriority
	}
	if r.ChannelIds == "" {
		r.ChannelIds = "[]"
	}
	if r.ChannelTags == "" {
		r.ChannelTags = "[]"
	}
	if r.ModelMapping == "" {
		r.ModelMapping = "{}"
	}
	return nil
}

func (r *Route) BeforeCreate(tx *gorm.DB) error {
	r.CreatedAt = common.GetTimestamp()
	r.UpdatedAt = common.GetTimestamp()
	return r.Validate()
}

func (r *Route) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return r.Validate()
}

func GetAllRoutes(page int, pageSize int, keyword string, kind string, enabledOnly bool) ([]*Route, int64, error) {
	var routes []*Route
	var total int64

	db := DB.Model(&Route{})
	if keyword != "" {
		pattern := "%" + strings.TrimSpace(keyword) + "%"
		db = db.Where("name LIKE ? OR slug LIKE ?", pattern, pattern)
	}
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	if enabledOnly {
		db = db.Where("enabled = ?", true)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&routes).Error
	return routes, total, err
}

func GetRouteById(id int) (*Route, error) {
	if id <= 0 {
		return nil, errors.New("invalid route id")
	}
	var route Route
	err := DB.First(&route, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRouteNotFound
		}
		return nil, err
	}
	return &route, nil
}

func GetRouteBySlug(slug string) (*Route, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, errors.New("empty route slug")
	}
	var route Route
	err := DB.First(&route, "slug = ?", slug).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRouteNotFound
		}
		return nil, err
	}
	return &route, nil
}

func GetActiveRoutesByIds(ids []int) (map[int]*Route, error) {
	if len(ids) == 0 {
		return make(map[int]*Route), nil
	}
	var routes []*Route
	err := DB.Where("id IN (?) AND enabled = ?", ids, true).Find(&routes).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]*Route, len(routes))
	for _, r := range routes {
		result[r.Id] = r
	}
	return result, nil
}

func (r *Route) Insert() error {
	if err := r.Validate(); err != nil {
		return err
	}
	// Check slug uniqueness
	var count int64
	DB.Model(&Route{}).Where("slug = ?", r.Slug).Count(&count)
	if count > 0 {
		return ErrRouteSlugExists
	}
	return DB.Create(r).Error
}

func (r *Route) Update() error {
	if err := r.Validate(); err != nil {
		return err
	}
	// Check slug uniqueness excluding self
	var count int64
	DB.Model(&Route{}).Where("slug = ? AND id != ?", r.Slug, r.Id).Count(&count)
	if count > 0 {
		return ErrRouteSlugExists
	}
	return DB.Model(r).Select(
		"name", "slug", "description", "enabled", "kind",
		"routing_policy", "channel_ids", "channel_tags",
		"model_mapping", "cost_multiplier", "min_user_group",
		"updated_at",
	).Updates(r).Error
}

// DeleteSafe checks referential integrity against active tokens before deleting.
func DeleteRouteSafe(id int) error {
	if id <= 0 {
		return errors.New("invalid route id")
	}
	// Check if any active token uses this route as PrimaryRouteId
	var primaryCount int64
	if err := DB.Model(&Token{}).Where("primary_route_id = ?", id).Count(&primaryCount).Error; err != nil {
		return err
	}
	if primaryCount > 0 {
		return fmt.Errorf("%w: %d token(s) use this route as primary route", ErrRouteReferencedByKeys, primaryCount)
	}

	// Check if any active token uses this route in fallback_route_ids
	// SQLite / MySQL JSON search or string like
	likePattern := fmt.Sprintf("%%[%d]%%", id)
	likePatternMid := fmt.Sprintf("%%,%d,%%", id)
	likePatternStart := fmt.Sprintf("%%[%d,%%", id)
	likePatternEnd := fmt.Sprintf("%%,%d]%%", id)
	var fallbackCount int64
	err := DB.Model(&Token{}).Where(
		"fallback_route_ids LIKE ? OR fallback_route_ids LIKE ? OR fallback_route_ids LIKE ? OR fallback_route_ids LIKE ?",
		likePattern, likePatternMid, likePatternStart, likePatternEnd,
	).Count(&fallbackCount).Error
	if err != nil {
		return err
	}
	if fallbackCount > 0 {
		return fmt.Errorf("%w: %d token(s) use this route in fallback routes", ErrRouteReferencedByKeys, fallbackCount)
	}

	return DB.Delete(&Route{}, "id = ?", id).Error
}
