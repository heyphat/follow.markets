package strategy

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	bn "github.com/adshao/go-binance/v2"
	ta "github.com/itsphat/techan"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
	"github.com/sdcoffey/big"
)

func Test_Strategy(t *testing.T) {
	path := "./signals/strategy_test.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	signal, err := NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	td := tax.NewTrade()
	td.Price = big.NewFromInt(2000)
	td.Quantity = big.NewFromInt(1)

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

	ok := r.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	entry := NewRule(*signal).SetRunner(r)
	risk := NewRiskRewardRule(0.5, 0.6, r)

	s := ta.RuleStrategy{
		EntryRule:      entry,
		ExitRule:       risk,
		UnstablePeriod: 0,
	}

	yes := s.ShouldEnter(10, ta.NewTradingRecord())
	assert.EqualValues(t, true, yes)
}
