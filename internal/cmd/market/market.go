package market

import (
	"sync"

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
	once.Do(func() {
		Market = &MarketStruct{}
		Market.watcher = watcher
		Market.streamer = streamer
		Market.evaluator = evaluator
		Market.connect()
	})
	return Market, nil
}

func (m *MarketStruct) connect() {
	m.watcher.connect()
	m.streamer.connect()
	m.evaluator.connect()
}

func (m *MarketStruct) Watch(name string) error {
	return m.watcher.watch(name)
}

func (m *MarketStruct) Watchlist() []string {
	return m.watcher.watchlist()
}
