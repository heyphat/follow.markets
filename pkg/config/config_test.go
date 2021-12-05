package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	path := "../../configs/configs.json"
	configs, err := NewConfigs(&path)

	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 6868, configs.Server.Port)
}
