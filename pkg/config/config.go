package config

import (
	"encoding/json"
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
	err = json.Unmarshal(raw, &configs)
	return &configs, err
}
