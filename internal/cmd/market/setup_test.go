package market

import (
	"io/ioutil"
	"testing"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"
)

func Test_Setup_BinSpotUpdate(t *testing.T) {
	r := runner.NewRunner("BTCUSDT", runner.NewRunnerDefaultConfigs())
	assert.EqualValues(t, "BTCUSDT", r.GetName())

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)
	s, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	o := &bn.CreateOrderResponse{Price: "10", OrigQuantity: "20"}
	st := newSetup(r, s, big.ONE, o)

	u := bn.WsOrderUpdate{
		ExecutionType: "TRADE",
		Status:        "FILLED",
		FilledVolume:  "4",
		LatestPrice:   "10",
		LatestVolume:  "4",
	}
	st.binSpotUpdateTrade(u)
	assert.EqualValues(t, "10.00", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "4.00", st.accFilledQtity.FormattedString(2))

	u = bn.WsOrderUpdate{
		Status:        "FILLED",
		ExecutionType: "TRADE",
		FilledVolume:  "20",
		LatestPrice:   "9",
		LatestVolume:  "16",
	}

	st.binSpotUpdateTrade(u)
	assert.EqualValues(t, "9.20", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "20.00", st.accFilledQtity.FormattedString(2))
}

func Test_Setup_BinFutuUpdate(t *testing.T) {
	rConfigs := runner.NewRunnerDefaultConfigs()
	rConfigs.Market = runner.Futures
	r := runner.NewRunner("BTCUSDT", rConfigs)
	assert.EqualValues(t, "BTCUSDT", r.GetName())

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)
	s, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	o := &bnf.CreateOrderResponse{Price: "10", OrigQuantity: "20"}
	st := newSetup(r, s, big.TEN, o)

	u := bnf.WsOrderTradeUpdate{
		ExecutionType:        "TRADE",
		Status:               "FILLED",
		AccumulatedFilledQty: "4",
		LastFilledPrice:      "10",
		LastFilledQty:        "4",
	}
	st.binFutuUpdateTrade(u)
	assert.EqualValues(t, "10.00", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "4.00", st.accFilledQtity.FormattedString(2))

	u = bnf.WsOrderTradeUpdate{
		Status:               "FILLED",
		ExecutionType:        "TRADE",
		AccumulatedFilledQty: "20",
		LastFilledPrice:      "9",
		LastFilledQty:        "16",
	}

	st.binFutuUpdateTrade(u)
	assert.EqualValues(t, "9.20", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "20.00", st.accFilledQtity.FormattedString(2))
}
