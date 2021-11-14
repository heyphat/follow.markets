package market

import (
	"follow.market/pkg/config"
	"follow.market/pkg/log"
)

type MarketConfigs struct {
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

func NewMarketConfigs(configs *config.Configs) *MarketConfigs {
	return &MarketConfigs{
		communicator: newCommunicator(),
		provider:     newProvider(configs),
		logger:       log.NewLogger(),
	}
}
