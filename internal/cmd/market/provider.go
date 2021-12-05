package market

import (
	"context"
	"regexp"
	"time"

	"follow.market/pkg/config"
	bn "github.com/adshao/go-binance/v2"
	ta "github.com/itsphat/techan"

	tax "follow.market/internal/pkg/techanex"
)

const (
	timeFramePattern = `\d+(m|h)`
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
	re, _ := regexp.Compile(timeFramePattern)
	limit := 1000  // binance limit per request
	iteration := 6 // minute candle need to be fetched multiple times
	interval := re.FindString(d.String())
	end := int64(time.Now().Unix() * 1000) // 1000 is second to millisecond
	start := end - int64(limit*60000)
	if d > time.Minute*15 && d < time.Hour*24 {
		iteration = 1
		start = end - (d * time.Duration(limit)).Milliseconds()
	} else if d >= time.Hour*24 {
		interval = "1d"
		iteration = 1
		start = end - (d * time.Duration(limit)).Milliseconds()
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
		candles = append(candles, tax.ConvertBinanceKline(kline, &d))
	}
	return candles, nil
}
