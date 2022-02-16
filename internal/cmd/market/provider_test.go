package market

import (
	"context"
	"fmt"
	"testing"
	"time"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func providerTestSuit() (*config.Configs, *provider, error) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	if err != nil {
		return nil, nil, err
	}
	provider := newProvider(configs)
	return configs, provider, nil
}
func Test_Provider(t *testing.T) {
	// The test has been done on watcher test, more will be added if more methods to be inplemented.
	configs, provider, err := providerTestSuit()
	assert.EqualValues(t, nil, err)

	listings, err := provider.fetchCoinFundamentals(configs.Market.Base.Crypto.QuoteCurrency, 1)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, len(listings) == 1)

	_, err = provider.binFutu.NewListPriceChangeStatsService().Do(context.Background())
	assert.EqualValues(t, nil, err)
	//for _, s := range stats {
	//	fmt.Println(fmt.Sprintf("%v", s.Symbol))
	//}

	klines, err := provider.fetchBinanceFuturesKlinesV3("BTCUSDT", time.Minute, &fetchOptions{limit: 60})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 60, len(klines))

	acc, err := provider.binSpot.NewGetAccountService().Do(context.Background())
	assert.EqualValues(t, nil, err)
	fmt.Println(fmt.Sprintf("%+v", *acc))
	for _, b := range acc.Balances {
		if b.Asset == "BNB" {
			fmt.Println(fmt.Sprintf("%+v", b))
		}
	}
}

func Test_ExchangeInfo(t *testing.T) {
	_, provider, err := providerTestSuit()
	assert.EqualValues(t, nil, err)

	precision, err := provider.fetchBinSpotExchangeInfo("BTCUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 2, precision)

	precision, err = provider.fetchBinSpotExchangeInfo("THETAUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 3, precision)

	precision, err = provider.fetchBinSpotExchangeInfo("SHIBUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 8, precision)
}
