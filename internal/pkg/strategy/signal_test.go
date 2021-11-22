package strategy

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"follow.market/internal/pkg/runner"
)

func Test_Signal(t *testing.T) {
	path := "./signal.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	signal, err := NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	ok := signal.Evaluate(nil, nil)
	assert.EqualValues(t, false, ok)

	for _, c := range signal.Conditions {
		err := c.This.validate()
		assert.EqualValues(t, nil, err)

		err = c.That.validate()
		assert.EqualValues(t, nil, err)

		ok := c.evaluate(nil, nil)
		assert.EqualValues(t, false, ok)

		ok = c.evaluate(runner.NewRunner("BTCUSDT", nil), nil)
		assert.EqualValues(t, false, ok)
		//TODO: init the runner and test
	}

	for _, g := range signal.ConditionGroups {
		err := g.validate()
		assert.EqualValues(t, nil, err)

		ok = g.evaluate(nil, nil)
		assert.EqualValues(t, false, ok)

		ok = g.evaluate(runner.NewRunner("BTCUSDT", nil), nil)
		assert.EqualValues(t, true, ok)
	}
}
