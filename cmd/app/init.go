package main

import (
	mk "follow.markets/internal/cmd/market"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

var (
	logger  *log.Logger
	market  *mk.MarketStruct
	configs *config.Configs
)

func init() {
	var err error
	logger = log.NewLogger()
	configPath := "./configs/configs.json"
	configs, err = config.NewConfigs(&configPath)
	if err != nil {
		panic(err)
	}
	market, err = mk.NewMarket(&configPath)
	if err != nil {
		panic(err)
	}
}
