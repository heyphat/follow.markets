package market

import (
	"sync"

	"follow.market/internal/pkg/strategy"
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

func (m *MarketStruct) watch(configs *config.Configs) {
	for _, t := range configs.Watchlist {
		m.watcher.watch(t)
	}
}

func (m *MarketStruct) connect() {
	m.watcher.connect()
	m.streamer.connect()
	m.evaluator.connect()
	m.notifier.connect()
}

func (m *MarketStruct) Watch(ticker string) error {
	return m.watcher.watch(ticker)
}

func (m *MarketStruct) Watchlist() []string {
	return m.watcher.watchlist()
}

func (m *MarketStruct) IsWatchingOn(ticker string) bool {
	return m.watcher.isWatchingOn(ticker)
}

func (m *MarketStruct) AddStrategy(ticker string, s strategy.Strategy) {
	m.evaluator.add(ticker, &s)
}

func (m *MarketStruct) AddChatID(cids []int64) {
	m.notifier.add(cids)
}
