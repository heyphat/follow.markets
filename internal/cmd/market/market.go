package market

import (
	"context"
	"io/ioutil"
	"sync"

	"github.com/dlclark/regexp2"

	"follow.market/internal/pkg/strategy"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/config"
	"follow.market/pkg/log"
	"follow.market/pkg/util"
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
	watcher   *watcher
	streamer  *streamer
	evaluator *evaluator
	notifier  *notifier
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
	notifier, err := newNotifier(common, configs)
	if err != nil {
		return nil, err
	}
	once.Do(func() {
		Market = &MarketStruct{}
		Market.watcher = watcher
		Market.streamer = streamer
		Market.evaluator = evaluator
		Market.notifier = notifier

		Market.connect()
		if err := Market.initSignals(configs); err != nil {
			common.logger.Error.Println("failed to init signals with err: ", err)
		}
		if err := Market.initWatchlist(configs); err != nil {
			common.logger.Error.Println("failed to init watchlist with err: ", err)
		}
	})
	return Market, nil
}

// watch initializes the watching process from watcher on watchlist specified
// in the config file.
func (m *MarketStruct) initWatchlist(configs *config.Configs) error {
	stats, err := m.watcher.provider.binSpot.NewListPriceChangeStatsService().Do(context.Background())
	if err != nil {
		return err
	}
	for _, p := range configs.Watchlist {
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
				if err := m.watcher.watch(s.Symbol, nil); err != nil {
					m.watcher.logger.Error.Println(m.watcher.newLog(s.Symbol, err.Error()))
				}
			}
		}
	}
	return nil
}

// initSignals adds all the singals defined as json files in the configs/signals dir.
func (m *MarketStruct) initSignals(configs *config.Configs) error {
	if len(configs.Signal.Path) == 0 {
		return nil
	}
	files, err := util.IOReadDir(configs.Signal.Path)
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
		tickers := `(?=(?<!(BUSD|BVND|PAX|DAI|TUSD|USDC|VAI|BRL|AUD|BIRD|EUR|GBP|BIDR|DOWN|UP|BEAR|BULL))USDT)(?=USDT$)`
		m.evaluator.add([]string{tickers}, signal)
	}
	return nil
}

func (m *MarketStruct) connect() {
	m.watcher.connect()
	m.streamer.connect()
	m.evaluator.connect()
	m.notifier.connect()
}

// watcher endpoints
func (m *MarketStruct) Watch(ticker string) error {
	return m.watcher.watch(ticker, nil)
}

func (m *MarketStruct) Watchlist() []string {
	return m.watcher.watchlist()
}

func (m *MarketStruct) IsWatchingOn(ticker string) bool {
	return m.watcher.isWatchingOn(ticker)
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
