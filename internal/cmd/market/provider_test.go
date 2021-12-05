package market

import (
	"testing"
	"time"

	"follow.market/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Provider(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	provider := newProvider(configs)

	candles, err := provider.fetchBinanceKlines("BTCUSDT", time.Minute)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 6000, len(candles))

	candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*30)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1000, len(candles))
	//fmt.Println(candles[len(candles)-1])

	candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*60)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1000, len(candles))
	//fmt.Println(candles[len(candles)-1])

	candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*120)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1000, len(candles))
	//fmt.Println(candles[len(candles)-1])

	candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Hour*4)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1000, len(candles))
	//fmt.Println(candles[len(candles)-1])
}
