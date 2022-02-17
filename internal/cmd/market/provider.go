package market

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"follow.markets/pkg/config"
	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	ta "github.com/itsphat/techan"
	cc "github.com/miguelmota/go-coinmarketcap/pro/v1"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
)

const (
	timeFramePattern = `\d+(m|h)`
)

type provider struct {
	binSpot *bn.Client
	binFutu *bnf.Client
	coinCap *cc.Client
}

func newProvider(configs *config.Configs) *provider {
	return &provider{
		binSpot: bn.NewClient(configs.Market.Provider.Binance.APIKey, configs.Market.Provider.Binance.SecretKey),
		binFutu: bnf.NewClient(configs.Market.Provider.Binance.APIKey, configs.Market.Provider.Binance.SecretKey),
		coinCap: cc.NewClient(&cc.Config{
			ProAPIKey: configs.Market.Provider.CoinMarketCap.APIKey,
		}),
	}
}

type fetchOptions struct {
	limit int
	start *time.Time
	end   *time.Time
}

func (p *provider) fetchBinanceSpotKlinesV3(ticker string, d time.Duration, opt *fetchOptions) ([]*ta.Candle, error) {
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

func (p *provider) fetchBinanceFuturesKlinesV3(ticker string, d time.Duration, opt *fetchOptions) ([]*ta.Candle, error) {
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
	var klines []*bnf.Kline
	for len(klines) < lmt || (opt.start != nil && len(klines) > 0 && klines[0].OpenTime > opt.start.Unix()*1000) {
		service := p.binFutu.NewKlinesService().Symbol(ticker).Interval(interval).EndTime(end).Limit(lmt)
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
		candles = append(candles, tax.ConvertBinanceFuturesKline(kline, &d))
	}
	return candles, nil
}

func (p *provider) fetchCoinFundamentals(base string, limit int) (map[string]runner.Fundamental, error) {
	out := make(map[string]runner.Fundamental)
	listings, err := p.coinCap.Cryptocurrency.LatestListings(&cc.ListingOptions{Limit: limit})
	if err != nil {
		return out, err
	}
	for _, l := range listings {
		out[l.Symbol+base] = runner.Fundamental{
			MaxSupply:         l.MaxSupply,
			TotalSupply:       l.TotalSupply,
			CirculatingSupply: l.CirculatingSupply,
		}
	}
	return out, nil
}

func (p *provider) fetchBinFutuBalances(quoteCurrency string) (*sync.Map, error) {
	balances, err := p.binFutu.NewGetBalanceService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	out := sync.Map{}
	for _, b := range balances {
		if big.NewFromString(b.Balance).GT(big.ZERO) {
			out.Store(b.Asset+quoteCurrency, *b)
		}
	}
	return &out, nil
}

func (p *provider) fetchBinSpotBalances(quoteCurrency string) (*sync.Map, error) {
	acc, err := p.binSpot.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	out := sync.Map{}
	for _, b := range acc.Balances {
		if big.NewFromString(b.Free).GT(big.ZERO) {
			out.Store(b.Asset+quoteCurrency, b)
		}
	}
	return &out, nil
}

func (p *provider) fetchBinUserDataListenKey() (string, string, error) {
	binSpotListenKey, err := p.binSpot.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return "", "", err
	}
	binFutuListenKey, err := p.binFutu.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return "", "", err
	}
	go func() {
		defer p.binSpot.NewCloseUserStreamService().ListenKey(binSpotListenKey).Do(context.Background())
		defer p.binFutu.NewCloseUserStreamService().ListenKey(binFutuListenKey).Do(context.Background())
		for {
			p.binSpot.NewKeepaliveUserStreamService().ListenKey(binSpotListenKey).Do(context.Background())
			p.binFutu.NewKeepaliveUserStreamService().ListenKey(binFutuListenKey).Do(context.Background())
			time.Sleep(time.Duration(30) * time.Minute)
		}
	}()
	return binSpotListenKey, binFutuListenKey, nil
}

func (p *provider) fetchBinSpotExchangeInfo(ticker string) (int, int, error) {
	i, err := p.binSpot.NewExchangeInfoService().Symbol(ticker).Do(context.Background())
	if err != nil {
		return 0, 0, err
	}
	precision := 0
	lotSize := 0
	for _, s := range i.Symbols {
		if strings.ToUpper(s.Symbol) != strings.ToUpper(ticker) {
			continue
		}
		for _, m := range s.Filters {
			switch m["filterType"] {
			case "PRICE_FILTER":
				val, ok := m["tickSize"]
				if !ok {
					return precision, lotSize, errors.New("couldn't find precision and lotSize from exchange")
				}
				for !(big.NewFromString("10").Pow(precision).Mul(big.NewFromString(val.(string))).GTE(big.NewFromString("1"))) {
					precision += 1
				}
			case "LOT_SIZE":
				val, ok := m["stepSize"]
				if !ok {
					return precision, lotSize, errors.New("couldn't find precision and lotSize from exchange")
				}
				for !(big.NewFromString("10").Pow(lotSize).Mul(big.NewFromString(val.(string))).GTE(big.NewFromString("1"))) {
					lotSize += 1
				}
			default:
				continue
			}
		}
	}
	return precision, lotSize, nil
}

func (p *provider) fetchBinFutuExchangeInfo(ticker string) (int, int, error) {
	i, err := p.binFutu.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return 0, 0, err
	}
	precision := 0
	lotSize := 0
	for _, s := range i.Symbols {
		if strings.ToUpper(s.Symbol) != strings.ToUpper(ticker) {
			continue
		}
		for _, m := range s.Filters {
			switch m["filterType"] {
			case "PRICE_FILTER":
				val, ok := m["tickSize"]
				if !ok {
					return precision, lotSize, errors.New("couldn't find precision and lotSize from exchange")
				}
				for !(big.TEN.Pow(precision).Mul(big.NewFromString(val.(string))).GTE(big.ONE)) {
					precision += 1
				}
			case "LOT_SIZE":
				val, ok := m["stepSize"]
				if !ok {
					return precision, lotSize, errors.New("couldn't find precision and lotSize from exchange")
				}
				for !(big.TEN.Pow(lotSize).Mul(big.NewFromString(val.(string))).GTE(big.ONE)) {
					lotSize += 1
				}
			default:
				continue
			}
		}
	}
	return precision, lotSize, nil
}
