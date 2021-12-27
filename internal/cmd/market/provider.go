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

type fetchOptions struct {
	limit int
	start *time.Time
	end   *time.Time
}

func (p *provider) fetchBinanceKlinesV3(ticker string, d time.Duration, opt *fetchOptions) ([]*ta.Candle, error) {
	lmt := 1000
	if opt != nil && opt.limit > 0 && opt.limit <= 1000 {
		lmt = opt.limit
	}
	re, _ := regexp.Compile(timeFramePattern)
	interval := re.FindString(d.String())
	if d >= time.Hour*24 {
		interval = "1d"
	}
	if d == time.Minute*10 {
		interval = "5m"
	}
	end := time.Now().Unix() * 1000
	if opt != nil && opt.end != nil {
		end = opt.end.Unix() * 1000
	}
	var klines []*bn.Kline
	for len(klines) < lmt || (opt.start != nil && len(klines) > 0 && klines[0].OpenTime > opt.start.Unix()*1000) {
		service := p.binSpot.NewKlinesService().Symbol(ticker).Interval(interval).EndTime(end).Limit(lmt)
		kls, err := service.Do(context.Background())
		if err != nil {
			return nil, err
		}
		klines = append(kls, klines...)
		if len(kls) > 0 {
			end = kls[0].CloseTime
		}
	}
	var candles []*ta.Candle
	for _, kline := range klines {
		candles = append(candles, tax.ConvertBinanceKline(kline, &d))
	}
	return candles, nil
}
