package strategy

import (
	"io/ioutil"
	"testing"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	bn "github.com/adshao/go-binance/v2"
	"github.com/stretchr/testify/assert"
)

func Test_Signal(t *testing.T) {
	path := "./signals/signal.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	signal, err := NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	ok := signal.Evaluate(nil, nil)
	assert.EqualValues(t, false, ok)

	r := runner.NewRunner("BTCUSDT", nil)
	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle1 := tax.ConvertBinanceKline(kline, nil)
	ok = r.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	for _, c := range signal.Conditions {
		err := c.This.validate()
		assert.EqualValues(t, nil, err)

		err = c.That.validate()
		assert.EqualValues(t, nil, err)

		ok := c.evaluate(nil, nil)
		assert.EqualValues(t, false, ok)

		ok = c.evaluate(r, nil)
		assert.EqualValues(t, true, ok)
	}

	for _, g := range signal.Groups {
		err := g.validate()
		assert.EqualValues(t, nil, err)

		ok = g.evaluate(nil, nil)
		assert.EqualValues(t, false, ok)

		ok = g.evaluate(r, nil)
		assert.EqualValues(t, true, ok)
	}

	//newSignal := signal.copy()
	//fmt.Println(signal, newSignal)
}
