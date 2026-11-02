package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestTaskPluginChannelHasNoOrdinaryAPIType(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelTypeTaskPlugin)
	assert.Equal(t, -1, apiType)
	assert.False(t, ok)
}

func Test9RouterAPITypeMapsToOpenAI(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelType9Router)
	assert.True(t, ok)
	assert.Equal(t, constant.APITypeOpenAI, apiType)
}
