package market

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/sdcoffey/big"

	db "follow.markets/internal/pkg/database"
	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
	"follow.markets/pkg/util"
)

var (
	Market *MarketStruct
	once   sync.Once
)

type sharedParticipants struct {
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

func initSharedParticipants(configs *config.Configs) *sharedParticipants {
	return &sharedParticipants{
		communicator: newCommunicator(),
		provider:     newProvider(configs),
		logger:       log.NewLogger(),
	}
}

type MarketStruct struct {
	configs *config.Configs

	watcher   *watcher
	streamer  *streamer
	evaluator *evaluator
	notifier  *notifier
	tester    *tester
	trader    *trader
}

func NewMarket(configFilePath *string) (*MarketStruct, error) {
	path := "./../../../configs/configs.json"
	if configFilePath != nil {
		path = *configFilePath
	}
	configs, err := config.NewConfigs(&path)
	if err != nil {
		return nil, err
	}
	common := initSharedParticipants(configs)
	watcher, err := newWatcher(common)
	if err != nil {
		return nil, err
	}
	streamer, err := newStreamer(common)
	if err != nil {
		return nil, err
	}
	evaluator, err := newEvaluator(common)
	if err != nil {
		return nil, err
	}
	tester, err := newTester(common, configs)
	if err != nil {
		return nil, err
	}
	trader, err := newTrader(common, configs)
	if err != nil {
		return nil, err
	}
	notifier, err := newNotifier(common, configs)
	if err != nil {
		return nil, err
	}
	once.Do(func() {
		Market = &MarketStruct{}
		Market.configs = configs
		Market.watcher = watcher
		Market.streamer = streamer
		Market.evaluator = evaluator
		Market.notifier = notifier
		Market.tester = tester
		Market.trader = trader

		Market.connect()
		if err := Market.initSignals(); err != nil {
			common.logger.Error.Println("failed to init signals with err: ", err)
		}
		go func() {
			for {
				if err := Market.initWatchlist(); err != nil {
					common.logger.Error.Println("failed to init watchlist with err: ", err)
				}
				time.Sleep(time.Hour * 24)
			}
		}()
		go func() {
			for {
				time.Sleep(time.Minute)
				// the duration must be the time period that the watcher is watching on.
				ticker := "BTCUSDT"
				if synced := Market.IsSynced(ticker, time.Minute*5); !synced {
					Market.notifier.notify(fmt.Sprintf("%s is out of sync for %s", ticker, (time.Minute*5).String()), nil)
				}
			}
		}()
	})
	return Market, nil
}

func (m *MarketStruct) connect() {
	m.watcher.connect()
	m.streamer.connect()
	m.evaluator.connect()
	m.trader.connect()
	m.notifier.connect()
}

func (m *MarketStruct) parseRunnerConfigs(market runner.MarketType) *runner.RunnerConfigs {
	out := runner.NewRunnerDefaultConfigs()
	if market == runner.Cash || market == runner.Futures {
		out.Market = runner.MarketType(market)
	}
	frames := []time.Duration{}
	if len(m.configs.Market.Watcher.Runner.Frames) > 0 {
		for _, f := range m.configs.Market.Watcher.Runner.Frames {
			if runner.ValidateFrame(time.Duration(f) * time.Second) {
				frames = append(frames, time.Duration(f)*time.Second)
			}
		}
		out.LFrames = frames
	}
	if len(m.configs.Market.Watcher.Runner.Indicators) > 0 {
		ic := make(map[tax.IndicatorName][]int, len(m.configs.Market.Watcher.Runner.Indicators))
		for k, v := range m.configs.Market.Watcher.Runner.Indicators {
			if util.StringSliceContains(tax.AvailableIndicators(), k) {
				ic[tax.IndicatorName(k)] = v
			}
		}
		out.IConfigs = ic
	}
	return out
}

// watch initializes the watching process from watcher on watchlist specified
// in the config file.
func (m *MarketStruct) initWatchlist() error {
	stats, err := m.watcher.provider.binSpot.NewListPriceChangeStatsService().Do(context.Background())
	if err != nil {
		return err
	}
	futuStats, err := m.watcher.provider.binFutu.NewListPriceChangeStatsService().Do(context.Background())
	if err != nil {
		return err
	}
	limit := 1
	if m.configs.IsProduction() {
		limit = 5000
	}
	listings, err := m.watcher.provider.fetchCoinFundamentals(m.configs.Market.Base.Crypto.QuoteCurrency, limit)
	if err != nil {
		m.watcher.logger.Error.Println(m.watcher.newLog("CMC", err.Error()))
	}
	for _, p := range m.configs.Market.Watcher.Watchlist {
		re, err := regexp2.Compile(p, 0)
		if err != nil {
			return err
		}
		for _, s := range stats {
			isMatched, err := re.MatchString(s.Symbol)
			if err != nil {
				return err
			}
			if isMatched {
				var fd *runner.Fundamental
				if val, ok := listings[s.Symbol]; ok {
					fd = &val
				}
				if err := m.watcher.watch(s.Symbol, m.parseRunnerConfigs(runner.Cash), fd); err != nil {
					m.watcher.logger.Error.Println(m.watcher.newLog(s.Symbol+"-"+string(runner.Cash), err.Error()))
				}
				time.Sleep(time.Second * 15)
			}
		}
		for _, s := range futuStats {
			if len(strings.Split(s.Symbol, "_")) > 1 {
				continue
			}
			isMatched, err := re.MatchString(s.Symbol)
			if err != nil {
				return err
			}
			if isMatched {
				var fd *runner.Fundamental
				if val, ok := listings[s.Symbol]; ok {
					fd = &val
				}
				if err := m.watcher.watch(s.Symbol, m.parseRunnerConfigs(runner.Futures), fd); err != nil {
					m.watcher.logger.Error.Println(m.watcher.newLog(s.Symbol+"-"+string(runner.Futures), err.Error()))
				}
				time.Sleep(time.Second * 15)
			}
		}
	}
	return nil
}

// initSignals adds all the singals defined as json files in the configs/signals dir.
func (m *MarketStruct) initSignals() error {
	if len(m.configs.Market.Evaluator.SourcePath) == 0 {
		return nil
	}
	files, err := util.IOReadDir(m.configs.Market.Evaluator.SourcePath)
	if err != nil {
		return err
	}
	for _, f := range files {
		bts, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		signal, err := strategy.NewSignalFromBytes(bts)
		if err != nil {
			return err
		}
		tickers := strings.Replace("(?=(?<!(SUSD|BUSD|BVND|PAX|DAI|TUSD|USDC|VAI|BRL|AUD|BIRD|EUR|GBP|BIDR|DOWN|UP|BEAR|BULL))USDT)(?={base_market}$)", "{base_market}", m.configs.Market.Base.Crypto.QuoteCurrency, 1)
		m.evaluator.add([]string{tickers}, signal)
	}
	return nil
}

// watcher endpoints
func (m *MarketStruct) Watch(ticker, market string) error {
	mk, ok := runner.ValidateMarket(market)
	if !ok {
		return errors.New("unsupported market")
	}
	return m.watcher.watch(ticker, m.parseRunnerConfigs(mk), nil)
}

func (m *MarketStruct) Drop(ticker, market string) error {
	mk, ok := runner.ValidateMarket(market)
	if !ok {
		return errors.New("unsupported market")
	}
	return m.watcher.drop(ticker, m.parseRunnerConfigs(mk))
}

func (m *MarketStruct) Watchlist() []string {
	return m.watcher.watchlist()
}

func (m *MarketStruct) IsWatchingOn(ticker string, market string) bool {
	mk, ok := runner.ValidateMarket(market)
	if !ok {
		return ok
	}
	rc := runner.NewRunnerDefaultConfigs()
	rc.Market = mk
	return m.watcher.isWatchingOn(runner.NewRunner(ticker, rc).GetUniqueName())
}

func (m *MarketStruct) LastCandles(ticker string) tax.CandlesJSON {
	last := m.watcher.lastCandles(ticker)
	var out tax.CandlesJSON
	for _, l := range last {
		if l == nil {
			continue
		}
		js := tax.Candle2JSON(l)
		out = append(out, *js)
	}
	return out
}

func (m *MarketStruct) LastIndicators(ticker string) tax.IndicatorsJSON {
	last := m.watcher.lastIndicators(ticker)
	var out tax.IndicatorsJSON
	for _, l := range last {
		if l == nil {
			continue
		}
		js := l.Indicator2JSON()
		out = append(out, *js)
	}
	return out
}

func (m *MarketStruct) IsSynced(ticker string, duration time.Duration) bool {
	return m.watcher.isSynced(ticker, duration)
}

// evaluator endpoints
func (m *MarketStruct) AddSignal(patterns []string, s *strategy.Signal) error {
	return m.evaluator.add(patterns, s)
}

func (m *MarketStruct) DropSignal(name string) error {
	return m.evaluator.drop(name)
}

func (m *MarketStruct) GetSingals(names []string) strategy.Signals {
	return m.evaluator.getByNames(names)
}

// notifier endpoints
func (m *MarketStruct) AddChatIDs(cids []int64) {
	m.notifier.addChatIDs(cids)
}

func (m *MarketStruct) GetNotifications() map[string]time.Time {
	return m.notifier.getNotifications()
}

// tester endpoints
func (m *MarketStruct) Test(id int64) error {
	st := db.BacktestStatusAccepted
	m.tester.provider.dbClient.UpdateBacktestStatus(id, &st)
	_, err := m.tester.test(id)
	if err != nil {
		return err
	}
	return nil
}

// trader endpoints
func (m *MarketStruct) Balances(market string) (map[string]string, error) {
	mk, ok := runner.ValidateMarket(market)
	if !ok {
		return nil, errors.New("unsupported market")
	}
	out := make(map[string]string)
	if mk == runner.Cash {
		balances := m.trader.binSpotGetBalances()
		for _, bl := range balances {
			out[bl.Asset] = big.NewFromString(bl.Free).Add(big.NewFromString(bl.Locked)).FormattedString(5)
		}
	} else if mk == runner.Futures {
		balances := m.trader.binFutuGetBalances()
		for _, bl := range balances {
			out[bl.Asset] = big.NewFromString(bl.Balance).FormattedString(5)
		}
	}
	return out, nil
}

func (m *MarketStruct) UpdateConfigs(c *config.Configs) error {
	m.configs = c
	return m.trader.updateConfigs(c)
}

func (m *MarketStruct) GetConfigs() *config.Configs {
	return m.configs
}
