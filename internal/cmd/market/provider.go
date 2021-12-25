package market

import (
	"context"
	"errors"
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

func (p *provider) fetchBinanceKlinesV2(ticker string, d time.Duration, start, end time.Time) ([]*ta.Candle, error) {
	startUnix := start.Truncate(d).Unix() * 1000
	endUnix := end.Truncate(d).Unix() * 1000
	re, _ := regexp.Compile(timeFramePattern)
	interval := re.FindString(d.String())
	if d >= time.Hour*24 {
		interval = "1d"
	}
	var service *bn.KlinesService
	var klines []*bn.Kline
	for len(klines) == 0 || (startUnix < endUnix) {
		service = p.binSpot.NewKlinesService().Symbol(ticker).Interval(interval).StartTime(startUnix).EndTime(endUnix).Limit(1000)
		kls, err := service.Do(context.Background())
		if err != nil {
			return nil, err
		}
		if len(kls) == 0 && endUnix != startUnix {
			return nil, errors.New("no candles on frame " + interval)
		}
		klines = append(klines, kls...)
		startUnix = klines[len(klines)-1].CloseTime
	}
	var candles []*ta.Candle
	for _, kline := range klines {
		candles = append(candles, tax.ConvertBinanceKline(kline, &d))
	}
	return candles, nil
}

func (p *provider) fetchBinanceKlinesV3(ticker string, d time.Duration) ([]*ta.Candle, error) {
	re, _ := regexp.Compile(timeFramePattern)
	interval := re.FindString(d.String())
	if d >= time.Hour*24 {
		interval = "1d"
	}
	if d == time.Minute*10 {
		interval = "5m"
	}
	end := time.Now()
	service := p.binSpot.NewKlinesService().Symbol(ticker).Interval(interval).EndTime(end.Unix() * 1000).Limit(1000)
	klines, err := service.Do(context.Background())
	if err != nil {
		return nil, err
	}
	var candles []*ta.Candle
	for _, kline := range klines {
		candles = append(candles, tax.ConvertBinanceKline(kline, &d))
	}
	return candles, nil
}
