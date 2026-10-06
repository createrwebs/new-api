package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type RouteRequest struct {
	Name           string            `json:"name" binding:"required"`
	Slug           string            `json:"slug" binding:"required"`
	Description    string            `json:"description"`
	Enabled        *bool             `json:"enabled"`
	Kind           string            `json:"kind"`
	RoutingPolicy  string            `json:"routing_policy"`
	ChannelIds     []int             `json:"channel_ids"`
	ChannelTags    []string          `json:"channel_tags"`
	ModelMapping   map[string]string `json:"model_mapping"`
	CostMultiplier float64           `json:"cost_multiplier"`
	MinUserGroup   string            `json:"min_user_group"`
}

type RouteResponse struct {
	*model.Route
	ChannelIds   []int             `json:"channel_ids"`
	ChannelTags  []string          `json:"channel_tags"`
	ModelMapping map[string]string `json:"model_mapping"`
}

func buildRouteResponse(r *model.Route) *RouteResponse {
	if r == nil {
		return nil
	}
	channelIds, _ := r.GetChannelIds()
	channelTags, _ := r.GetChannelTags()
	modelMapping, _ := r.GetModelMapping()
	return &RouteResponse{
		Route:        r,
		ChannelIds:   channelIds,
		ChannelTags:  channelTags,
		ModelMapping: modelMapping,
	}
}

func buildRouteResponses(routes []*model.Route) []*RouteResponse {
	responses := make([]*RouteResponse, 0, len(routes))
	for _, r := range routes {
		responses = append(responses, buildRouteResponse(r))
	}
	return responses
}

// GetAllRoutes handles GET /api/routes
func GetAllRoutes(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	kind := c.Query("kind")
	isAdmin := c.GetInt("role") >= common.RoleAdminUser

	// Normal users can only view enabled routes
	enabledOnly := !isAdmin

	routes, total, err := model.GetAllRoutes(pageInfo.GetPage(), pageInfo.GetPageSize(), keyword, kind, enabledOnly)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// If not admin, filter out routes requiring higher entitlement than user has
	if !isAdmin {
		userGroup := c.GetString("group")
		filtered := make([]*model.Route, 0, len(routes))
		for _, r := range routes {
			if service.ValidateRouteEntitlement(r, userGroup) == nil {
				filtered = append(filtered, r)
			}
		}
		routes = filtered
		total = int64(len(filtered))
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildRouteResponses(routes))
	common.ApiSuccess(c, pageInfo)
}

// GetRoute handles GET /api/routes/:id
func GetRoute(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid route id"))
		return
	}
	route, err := service.CacheGetRoute(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	isAdmin := c.GetInt("role") >= common.RoleAdminUser
	if !isAdmin {
		userGroup := c.GetString("group")
		if err := service.ValidateRouteEntitlement(route, userGroup); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
			return
		}
	}
	common.ApiSuccess(c, buildRouteResponse(route))
}

// GetAvailableRoutes handles GET /api/routes/available
// Returns all enabled routes permitted for the authenticated user to configure on their API keys.
func GetAvailableRoutes(c *gin.Context) {
	userGroup := c.GetString("group")
	routes, _, err := model.GetAllRoutes(1, 100, "", "", true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	available := make([]*RouteResponse, 0, len(routes))
	for _, r := range routes {
		if service.ValidateRouteEntitlement(r, userGroup) == nil {
			available = append(available, buildRouteResponse(r))
		}
	}
	common.ApiSuccess(c, available)
}

// CreateRoute handles POST /api/routes (Admin only)
func CreateRoute(c *gin.Context) {
	var req RouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	multiplier := req.CostMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	route := &model.Route{
		Name:           strings.TrimSpace(req.Name),
		Slug:           strings.TrimSpace(req.Slug),
		Description:    strings.TrimSpace(req.Description),
		Enabled:        enabled,
		Kind:           strings.TrimSpace(req.Kind),
		RoutingPolicy:  strings.TrimSpace(req.RoutingPolicy),
		CostMultiplier: multiplier,
		MinUserGroup:   strings.TrimSpace(req.MinUserGroup),
	}

	if err := route.SetChannelIds(req.ChannelIds); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := route.SetChannelTags(req.ChannelTags); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := route.SetModelMapping(req.ModelMapping); err != nil {
		common.ApiError(c, err)
		return
	}

	if err := route.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}

	service.InvalidateRouteCache(route.Id)
	common.ApiSuccess(c, buildRouteResponse(route))
}

// UpdateRoute handles PUT /api/routes/:id (Admin only)
func UpdateRoute(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid route id"))
		return
	}

	route, err := model.GetRouteById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req RouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if req.Name != "" {
		route.Name = strings.TrimSpace(req.Name)
	}
	if req.Slug != "" {
		route.Slug = strings.TrimSpace(req.Slug)
	}
	route.Description = strings.TrimSpace(req.Description)
	if req.Enabled != nil {
		route.Enabled = *req.Enabled
	}
	if req.Kind != "" {
		route.Kind = strings.TrimSpace(req.Kind)
	}
	if req.RoutingPolicy != "" {
		route.RoutingPolicy = strings.TrimSpace(req.RoutingPolicy)
	}
	if req.CostMultiplier > 0 {
		route.CostMultiplier = req.CostMultiplier
	}
	route.MinUserGroup = strings.TrimSpace(req.MinUserGroup)

	if req.ChannelIds != nil {
		if err := route.SetChannelIds(req.ChannelIds); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if req.ChannelTags != nil {
		if err := route.SetChannelTags(req.ChannelTags); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if req.ModelMapping != nil {
		if err := route.SetModelMapping(req.ModelMapping); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	if err := route.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	service.InvalidateRouteCache(route.Id)
	common.ApiSuccess(c, buildRouteResponse(route))
}

// DeleteRoute handles DELETE /api/routes/:id (Admin only)
func DeleteRoute(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid route id"))
		return
	}

	if err := model.DeleteRouteSafe(id); err != nil {
		if errors.Is(err, model.ErrRouteReferencedByKeys) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
			return
		}
		common.ApiError(c, err)
		return
	}

	service.InvalidateRouteCache(id)
	common.ApiSuccess(c, gin.H{"id": id, "deleted": true})
}
