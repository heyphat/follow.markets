package market

import (
	"testing"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Provider(t *testing.T) {
	// The test has been done on watcher test, more will be added if more methods to be inplemented.
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	provider := newProvider(configs)

	//candles, err := provider.fetchBinanceKlines("BTCUSDT", time.Minute)
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, 6000, len(candles))

	//candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*30)
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, 1000, len(candles))
	////fmt.Println(candles[len(candles)-1])

	//candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*60)
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, 1000, len(candles))
	////fmt.Println(candles[len(candles)-1])

	//candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Minute*120)
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, 1000, len(candles))
	////fmt.Println(candles[len(candles)-1])

	//candles, err = provider.fetchBinanceKlines("BTCUSDT", time.Hour*4)
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, 1000, len(candles))
	////fmt.Println(candles[len(candles)-1])

	listings, err := provider.fetchCoinFundamentals(configs.Market.Watcher.BaseMarket, 1)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, len(listings) == 1)
}
