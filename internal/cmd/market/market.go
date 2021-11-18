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
	Watcher  *Watcher
	streamer *streamer
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
	once.Do(func() {
		Market = &MarketStruct{}
		Market.Watcher = watcher
		Market.streamer = streamer
	})
	return Market, nil
}
