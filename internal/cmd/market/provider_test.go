package market

import (
	"context"
	"fmt"
	"testing"
	"time"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Provider(t *testing.T) {
	// The test has been done on watcher test, more will be added if more methods to be inplemented.
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	provider := newProvider(configs)

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
