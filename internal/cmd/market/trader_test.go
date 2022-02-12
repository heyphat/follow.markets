package market

import (
	"io/ioutil"
	"testing"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	"follow.markets/pkg/config"
	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"
)

func Test_Trader(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	trader, err := newTrader(initSharedParticipants(configs), configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	mem := tdmember{
		runner: runner.NewRunner("BTCUSDT", runner.NewRunnerDefaultConfigs()),
		channels: &streamingChannels{
			bar:   nil,
			trade: nil,
			depth: make(chan interface{}, 10),
		},
	}

	assert.EqualValues(t, "BTCUSDT", mem.runner.GetName())
}

func Test_Trader_Evaluator(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	common := initSharedParticipants(configs)
	trader, err := newTrader(common, configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	trader.connect()
	assert.EqualValues(t, true, trader.isConnected())

	r := runner.NewRunner("BTCUSDT", runner.NewRunnerDefaultConfigs())
	assert.EqualValues(t, "BTCUSDT", r.GetName())

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)
	s, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	common.communicator.evaluator2Trader <- common.communicator.newMessage(r, s, nil, nil, nil)

	time.Sleep(time.Minute)
}

func Test_Trader_StopLoss(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	common := initSharedParticipants(configs)
	trader, err := newTrader(common, configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	trader.connect()
	assert.EqualValues(t, true, trader.isConnected())

	orderPrice := big.NewFromString("10.0")

	tradingSide := "BUY"
	currentPrice := big.NewFromString("10.3")
	shouldStop, pnl := trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "0.03", pnl.FormattedString(2))

	currentPrice = big.NewFromString("9.5")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "-0.05", pnl.FormattedString(2))

	currentPrice = big.NewFromString("10.05")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, false, shouldStop)
	assert.EqualValues(t, "0.005", pnl.FormattedString(3))

	currentPrice = big.NewFromString("9.99")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, false, shouldStop)
	assert.EqualValues(t, "-0.001", pnl.FormattedString(3))

	tradingSide = "SELL"
	currentPrice = big.NewFromString("10.5")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "-0.05", pnl.FormattedString(2))

	currentPrice = big.NewFromString("9.5")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "0.05", pnl.FormattedString(2))

	currentPrice = big.NewFromString("10.01")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, false, shouldStop)
	assert.EqualValues(t, "-0.001", pnl.FormattedString(3))

	currentPrice = big.NewFromString("9.99")
	shouldStop, pnl = trader.shouldClose(orderPrice, currentPrice, tradingSide)
	assert.EqualValues(t, false, shouldStop)
	assert.EqualValues(t, "0.001", pnl.FormattedString(3))

}
