package main

import (
	"os"

	mk "follow.market/internal/cmd/market"
	"follow.market/pkg/config"
	"follow.market/pkg/log"
)

var (
	configs *config.Configs
	logger  *log.Logger
	market  *mk.MarketStruct
)

func init() {
	var err error
	logger = log.NewLogger()
	configPath := "./configs/configs.json"
	envPath := os.Getenv("MARKET_CONFIG_PATH")
	if len(envPath) != 0 {
		configPath = envPath
	}
	logger.Info.Println("config path: ", configPath)
	configs, err = config.NewConfigs(&configPath)
	if err != nil {
		panic(err)
	}
	market, err = mk.NewMarket(&configPath)
	if err != nil {
		panic(err)
	}
}
