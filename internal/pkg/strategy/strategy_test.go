package strategy

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"follow.market/internal/pkg/runner"
)

func Test_Strategy(t *testing.T) {
	path := "./strategy.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	strategy, err := NewStrategy(raw)
	assert.EqualValues(t, nil, err)

	ok := strategy.Evaluate(nil)
	assert.EqualValues(t, false, ok)

	for _, c := range strategy.Conditions {
		err := c.This.validate()
		assert.EqualValues(t, nil, err)

		err = c.That.validate()
		assert.EqualValues(t, nil, err)

		ok := c.evaluate(nil)
		assert.EqualValues(t, false, ok)

		ok = c.evaluate(runner.NewRunner("BTCUSDT", nil))
		assert.EqualValues(t, false, ok)
		//TODO: init the runner and test
	}

	for _, g := range strategy.ConditionGroups {
		err := g.validate()
		assert.EqualValues(t, nil, err)

		ok = g.evaluate(nil)
		assert.EqualValues(t, false, ok)

		ok = g.evaluate(runner.NewRunner("BTCUSDT", nil))
		assert.EqualValues(t, true, ok)
	}
}
