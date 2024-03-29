package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
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
	Market struct {
		Base struct {
			LocalTime string `json:"local_timezone"`
			Crypto    struct {
				QuoteCurrency string `json:"quote_currency"`
			} `json:"crypto"`
		} `json:"base"`
		Provider struct {
			Polygon struct {
				APIKey *string `json:"api_key"`
			}
			Binance struct {
				APIKey    string `json:"api_key"`
				SecretKey string `json:"secret_key"`
			} `json:"binance"`
			CoinMarketCap struct {
				APIKey string `json:"api_key"`
			} `json:"coinmarketcap"`
		} `json:"provider"`
		Notifier struct {
			ShowDescription bool `json:"show_signal_description"`
			Telegram        struct {
				ChatIDs     []string `json:"chat_ids"`
				BotToken    string   `json:"bot_token"`
				BotPassword string   `json:"bot_password"`
			} `json:"telegram"`
		} `json:"notifier"`
		Watcher struct {
			Watchlist []string `json:"watchlist"`
			Runner    struct {
				Frames     []int            `json:"frames"`
				Indicators map[string][]int `json:"indicators"`
			} `json:"runner"`
		} `json:"watcher"`
		Evaluator struct {
			SourcePath string `json:"source_path"`
		} `json:evaluator`
		Tester struct {
			SavePath      string  `json:"save_path"`
			InitBalance   float64 `json:"init_balance"`
			ProfitMargin  float64 `json:"profit_margin"`
			LossTolerance float64 `json:"loss_tolerance"`
		} `json:"tester"`
		Trader struct {
			Allowed           bool     `json:"allowed"`
			AllowedPatterns   []string `json:"allowed_patterns"`
			AllowedMarkets    []string `json:"allowed_markets"`
			MinBalance        float64  `json:"min_balance_to_trade"`
			MaxLeverage       float64  `json:"max_leverage"`
			MaxPositions      float64  `json:"max_concurrent_positions"`
			MaxWaitToFill     float64  `json:"max_wait_to_fill"`
			LossTolerance     float64  `json:"loss_tolerance"`
			ProfitMargin      float64  `json:"profit_margin"`
			MaxLossPerTrade   float64  `json:"max_loss_per_trade"`
			MinProfitPerTrade float64  `json:"min_profit_per_trade"`
		} `json:"trader"`
	} `json:"market"`
	Database struct {
		Use     string   `json:"use"`
		MongoDB *MongoDB `json:"mongodb"`
		Notion  *Notion  `json:"notion"`
	} `json:"database"`
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
	if len(configs.Datadog.Host) > 0 {
		os.Setenv("DD_AGENT_HOST", configs.Datadog.Host)
	}
	if len(configs.Market.Base.Crypto.QuoteCurrency) == 0 {
		configs.Market.Base.Crypto.QuoteCurrency = "USDT"
	}
	if len(configs.Market.Provider.Binance.APIKey) == 0 {
		return &configs, errors.New("missing binance api key and secret")
	}
	//if len(configs.Market.Provider.Polygon.APIKey) == 0 {
	//	return &configs, errors.New("missing polygon api key")
	//}
	return &configs, err
}
