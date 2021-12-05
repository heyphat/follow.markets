package techanex

import (
	"testing"

	bn "github.com/adshao/go-binance/v2"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	kline := &bn.Kline{
		OpenTime:                 1499040000000,
		Open:                     "0.0",
		High:                     "0.8",
		Low:                      "0.01",
		Close:                    "0.05",
		Volume:                   "148976.1",
		CloseTime:                1499644799999,
		QuoteAssetVolume:         "2434.19055334",
		TradeNum:                 308,
		TakerBuyBaseAssetVolume:  "1756.87402397",
		TakerBuyQuoteAssetVolume: "28.46694368",
	}

	candle := ConvertBinanceKline(kline, nil)
	assert.EqualValues(t, 0.0, candle.OpenPrice.Float())
	assert.EqualValues(t, 0.8, candle.MaxPrice.Float())
	assert.EqualValues(t, 0.01, candle.MinPrice.Float())
}
