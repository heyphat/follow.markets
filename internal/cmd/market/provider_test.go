package market

import (
	"testing"
	"time"

	"follow.market/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Provider(t *testing.T) {
	path := "./../../../configs/dev_configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	provider := newProvider(configs)

	candles, err := provider.fetchBinanceKlines("BTCUSDT", time.Minute)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 6000, len(candles))
}
