package market

import (
	"io/ioutil"
	"testing"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	bn "github.com/adshao/go-binance/v2"
	"github.com/stretchr/testify/assert"
)

func Test_Setup(t *testing.T) {
	r := runner.NewRunner("BTCUSDT", runner.NewRunnerDefaultConfigs())
	assert.EqualValues(t, "BTCUSDT", r.GetName())

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)
	s, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	o := bn.CreateOrderResponse{Price: "10", OrigQuantity: "20"}
	st := newSetup(r, s, o)

	u := bn.WsOrderUpdate{
		Status:       "TRADE",
		FilledVolume: "4",
		LatestPrice:  "10",
		LatestVolume: "4",
	}
	st.binSpotUpdateTrade(u)
	assert.EqualValues(t, "10.00", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "4.00", st.accFilledQtity.FormattedString(2))

	u = bn.WsOrderUpdate{
		Status:       "TRADE",
		FilledVolume: "20",
		LatestPrice:  "9",
		LatestVolume: "16",
	}

	st.binSpotUpdateTrade(u)
	assert.EqualValues(t, "9.20", st.avgFilledPrice.FormattedString(2))
	assert.EqualValues(t, "20.00", st.accFilledQtity.FormattedString(2))
}
