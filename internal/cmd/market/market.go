package market

import (
	"context"
	"sync"

	"github.com/dlclark/regexp2"

	"follow.market/internal/pkg/strategy"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/config"
	"follow.market/pkg/log"
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

func NewMarket(configPathFile *string) (*MarketStruct, error) {
	path := "./../../../configs/configs.json"
	if configPathFile != nil {
		path = *configPathFile
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
		Market.watch(configs)
	})
	return Market, nil
}

// watch will initialize the watching process from watcher on watchlist specified
// in the config file.
func (m *MarketStruct) watch(configs *config.Configs) error {
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
			isMatch, err := re.MatchString(s.Symbol)
			if err != nil {
				return err
			}
			if isMatch {
				m.watcher.watch(s.Symbol, nil)
			}
		}
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
		out = append(out, tax.Candle2JSON(l))
	}
	return out
}

func (m *MarketStruct) LastIndicators(ticker string) tax.IndicatorsJSON {
	last := m.watcher.lastIndicators(ticker)
	var out tax.IndicatorsJSON
	for _, l := range last {
		out = append(out, l.Indicator2JSON())
	}
	return out
}

// evaluator endpoints
func (m *MarketStruct) AddSignal(ticker string, s strategy.Signal) {
	m.evaluator.add(ticker, &s)
}

// notifier endpoints
func (m *MarketStruct) AddChatID(cids []int64) {
	m.notifier.add(cids)
}
