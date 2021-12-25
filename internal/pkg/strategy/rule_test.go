package strategy

import (
	"io/ioutil"
	"testing"

	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"

	tax "follow.market/internal/pkg/techanex"
)

func Test_Rule(t *testing.T) {
	return
	path := "./signal_trade.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	signal, err := NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	td := tax.NewTrade()
	td.Price = big.NewFromInt(2000)
	td.Quantity = big.NewFromInt(1)

	rule := NewRule(*signal) //.SetTrade(td)

	assert.EqualValues(t, true, rule.IsSatisfied(0, nil))

	//risk := NewRiskRewardRule(0.5, 0.6, runner.NewRunner("BTCUSDT", nil))
}
