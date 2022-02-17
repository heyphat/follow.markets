package market

import (
	"context"
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

	klines, err := provider.fetchBinanceFuturesKlinesV3("BTCUSDT", time.Minute, &fetchOptions{limit: 60})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 60, len(klines))

}

func Test_Provider_BinSpotExchangeInfo(t *testing.T) {
	_, provider, err := providerTestSuit()
	assert.EqualValues(t, nil, err)

	precision, lotSize, err := provider.fetchBinSpotExchangeInfo("BTCUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 2, precision)
	assert.EqualValues(t, 5, lotSize)

	precision, lotSize, err = provider.fetchBinSpotExchangeInfo("THETAUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 3, precision)
	assert.EqualValues(t, 1, lotSize)

	precision, lotSize, err = provider.fetchBinSpotExchangeInfo("SHIBUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 8, precision)
	assert.EqualValues(t, 0, lotSize)
}

func Test_Provider_BinFutuExchangeInfo(t *testing.T) {
	_, provider, err := providerTestSuit()
	assert.EqualValues(t, nil, err)

	pricePrecision, quantityPrecision, err := provider.fetchBinFutuExchangeInfo("BTCUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1, pricePrecision)
	assert.EqualValues(t, 3, quantityPrecision)

	pricePrecision, quantityPrecision, err = provider.fetchBinFutuExchangeInfo("THETAUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 3, pricePrecision)
	assert.EqualValues(t, 1, quantityPrecision)

	pricePrecision, quantityPrecision, err = provider.fetchBinFutuExchangeInfo("SHIBUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 0, pricePrecision)
	assert.EqualValues(t, 0, quantityPrecision)
}

func Test_Balances(t *testing.T) {
	_, provider, err := providerTestSuit()
	assert.EqualValues(t, nil, err)

	acc, err := provider.binSpot.NewGetAccountService().Do(context.Background())
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, len(acc.Balances) > 0)

	bls, err := provider.binFutu.NewGetBalanceService().Do(context.Background())
	assert.EqualValues(t, true, len(bls) > 0)
}
