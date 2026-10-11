package service_test

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
)

func TestRouteEntitlementBoundary(t *testing.T) {
	// Invariant 0.1: Route != Entitlement Group. A route cannot grant Pro entitlement to a Free user.
	proRoute := &model.Route{
		Id:           1,
		Name:         "Pro Fast Route",
		Enabled:      true,
		MinUserGroup: "pro",
	}

	// 1. Free user on "default" tier attempting to use Pro route -> REJECTED
	err := service.ValidateRouteEntitlement(proRoute, "default")
	assert.Error(t, err, "free user must not access pro-only route")
	assert.Contains(t, err.Error(), "requires pro entitlement")

	// 2. Pro user on "pro" tier -> ALLOWED
	err = service.ValidateRouteEntitlement(proRoute, "pro")
	assert.NoError(t, err)

	// 3. Admin / Root user -> ALLOWED
	err = service.ValidateRouteEntitlement(proRoute, "root")
	assert.NoError(t, err)

	// 4. Disabled route -> REJECTED for all users
	disabledRoute := &model.Route{
		Id:           2,
		Name:         "Disabled Route",
		Enabled:      false,
		MinUserGroup: "default",
	}
	err = service.ValidateRouteEntitlement(disabledRoute, "default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestRouteModelResolutionAndMapping(t *testing.T) {
	route := &model.Route{
		Name: "Claude Direct",
	}
	_ = route.SetModelMapping(map[string]string{
		"gpt-4": "gpt-4o",
	})

	// Model mapped
	resolved := service.ResolveModelForRoute(route, "gpt-4")
	assert.Equal(t, "gpt-4o", resolved)

	// Model unmapped
	resolved = service.ResolveModelForRoute(route, "claude-3-5-sonnet")
	assert.Equal(t, "claude-3-5-sonnet", resolved)
}

func TestChannelCooldown(t *testing.T) {
	channelId := 9999

	assert.False(t, service.IsChannelInCooldown(channelId))

	service.SetChannelCooldown(channelId, 50*time.Millisecond)
	assert.True(t, service.IsChannelInCooldown(channelId))

	time.Sleep(60 * time.Millisecond)
	assert.False(t, service.IsChannelInCooldown(channelId), "cooldown must expire automatically")
}
