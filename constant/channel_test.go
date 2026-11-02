package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChannelBaseURLIsBoundsSafe(t *testing.T) {
	assert.Empty(t, GetChannelBaseURL(ChannelTypeTaskPlugin))
	assert.Empty(t, GetChannelBaseURL(9999))
}

func Test9RouterChannelConstant(t *testing.T) {
	assert.Equal(t, 64, ChannelType9Router)
	assert.Equal(t, "http://localhost:20128", GetChannelBaseURL(ChannelType9Router))
	assert.Equal(t, "9Router", ChannelTypeNames[ChannelType9Router])
}
