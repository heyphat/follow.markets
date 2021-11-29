package market

import (
	"context"
	"time"

	"follow.market/pkg/config"
	bn "github.com/adshao/go-binance/v2"
	ta "github.com/itsphat/techan"

	tax "follow.market/internal/pkg/techanex"
)

type provider struct {
	binSpot *bn.Client
}

func newProvider(configs *config.Configs) *provider {
	return &provider{
		binSpot: bn.NewClient(configs.Markets.Binance.APIKey, configs.Markets.Binance.SecretKey),
	}
}

func (p *provider) fetchBinanceKlines(ticker string, d time.Duration) ([]*ta.Candle, error) {
	limit := 1000  // binance limit per request
	iteration := 6 // minute candle need to be fetched multiple times
	interval := "1m"
	end := int64(time.Now().Unix() * 1000)
	start := end - int64(limit*60000)
	if d == time.Hour*24 {
		interval = "1d"
		iteration = 1
		start = end - time.Duration(4*7*24*time.Hour).Milliseconds() // 4 weeks
	}
	var service *bn.KlinesService
	var klines []*bn.Kline
	for i := 0; i < iteration; i++ {
		service = p.binSpot.NewKlinesService().Symbol(ticker).Interval(interval).StartTime(start).EndTime(end).Limit(limit)
		kls, err := service.Do(context.Background())
		if err != nil {
			return nil, err
		}
		klines = append(kls, klines...)
		end = start
		start = end - int64(limit*60000)
	}
	var candles []*ta.Candle
	for _, kline := range klines {
		candles = append(candles, tax.ConvertBinanceKline(kline, nil))
	}
	return candles, nil
}

func (p *provider) fetchActiveTickers() ([]string, error) {
	var out []string
	return out, nil
}
