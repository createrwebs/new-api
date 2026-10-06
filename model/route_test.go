package model_test

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteValidationAndSerialization(t *testing.T) {
	route := &model.Route{
		Name:           "Test Route",
		Slug:           "test-route",
		Kind:           model.RouteKindLLM,
		RoutingPolicy:  model.RoutePolicyPriority,
		CostMultiplier: 1.25,
	}

	// Test channel IDs serialization and deduplication
	err := route.SetChannelIds([]int{1, 5, 5, 2, -1, 0})
	require.NoError(t, err)

	ids, err := route.GetChannelIds()
	require.NoError(t, err)
	assert.Equal(t, []int{1, 5, 2}, ids)

	// Test channel tags
	err = route.SetChannelTags([]string{"fast", "fast", "us-east", ""})
	require.NoError(t, err)

	tags, err := route.GetChannelTags()
	require.NoError(t, err)
	assert.Equal(t, []string{"fast", "us-east"}, tags)

	// Test model mapping
	err = route.SetModelMapping(map[string]string{
		"gpt-4": "gpt-4o",
	})
	require.NoError(t, err)

	mapping, err := route.GetModelMapping()
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", mapping["gpt-4"])

	// Test cost multiplier bounds
	route.CostMultiplier = 0
	assert.ErrorIs(t, route.Validate(), model.ErrInvalidRouteMultiplier)

	route.CostMultiplier = 15.0
	assert.ErrorIs(t, route.Validate(), model.ErrInvalidRouteMultiplier)

	route.CostMultiplier = 1.0
	assert.NoError(t, route.Validate())
}

func TestTokenRouteChain(t *testing.T) {
	token := &model.Token{
		PrimaryRouteId: 10,
	}
	assert.True(t, token.IsRouteEnabled())

	err := token.SetFallbackRouteIds([]int{20, 30, 20, 10, -5})
	require.NoError(t, err)

	fallbacks, err := token.GetFallbackRouteIds()
	require.NoError(t, err)
	assert.Equal(t, []int{20, 30}, fallbacks, "primary route and duplicates should be filtered out")

	chain := token.GetRouteChain()
	assert.Equal(t, []int{10, 20, 30}, chain, "chain should be deterministic [primary, fallback1, fallback2]")

	// Legacy token without primary route
	legacyToken := &model.Token{
		PrimaryRouteId: 0,
	}
	assert.False(t, legacyToken.IsRouteEnabled())
	assert.Empty(t, legacyToken.GetRouteChain())
}
