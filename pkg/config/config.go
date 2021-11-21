package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"
)

type Configs struct {
	// server configuration, panic on missing
	Stage  string `json:"env"`
	Server struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Limit   int    `json:"limit"`
		Timeout struct {
			Read  int `json:"read"`
			Write int `json:"write"`
			Idle  int `json:"idle"`
		} `json:"timeout"`
	} `json:"server"`
	// datadog for system monitoring (optional)
	Datadog struct {
		Host    string `json:"host"`
		Port    string `json:"port"`
		Env     string `json:"env"`
		Service string `json:"service"`
		Version string `json:"version"`
	} `json:"datadog"`
	Markets struct {
		Binance struct {
			APIKey    string `json:"api_key"`
			SecretKey string `json:"secret_key"`
		} `json:"binance"`
	} `json:"markets"`
	Telegram struct {
		BotToken string   `json:"bot_token"`
		ChatIDs  []string `json:"chat_ids"`
	} `json:"telegram"`
	Watchlist []string `json:"watchlist"`
}

func (c Configs) IsProduction() bool {
	if found, err := regexp.MatchString("production|prod", strings.ToLower(c.Stage)); err != nil || !found {
		return false
	}
	return true
}

func NewConfigs(filePath *string) (*Configs, error) {
	configFilePath := "./configs/configs.json"
	if filePath != nil {
		configFilePath = *filePath
	}
	raw, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	configs := Configs{}
	if err = json.Unmarshal(raw, &configs); err != nil {
		return &configs, err
	}
	if configs.Server.Port == 0 {
		configs.Server.Port = 6868
	}
	if configs.Server.Timeout.Read == 0 {
		configs.Server.Timeout.Read = 10
	}
	if configs.Server.Timeout.Write == 0 {
		configs.Server.Timeout.Write = 10
		configs.Server.Timeout.Idle = 10
	}
	if len(configs.Markets.Binance.APIKey) == 0 {
		return &configs, errors.New("missing binance api key and secret")
	}
	return &configs, err
}
