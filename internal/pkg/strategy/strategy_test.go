package builder

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	tax "follow.market/internal/pkg/techanex"
)

func Test_Strategy(t *testing.T) {
	path := "./strategy.json"
	raw, err := ioutil.ReadFile(path)
	assert.EqualValues(t, nil, err)

	strategy, err := NewStrategy(raw)
	assert.EqualValues(t, nil, err)

	ok := strategy.Evaluate()
	assert.EqualValues(t, false, ok)

	for _, c := range strategy.Conditions {
		err := c.This.validate()
		assert.EqualValues(t, nil, err)

		err = c.That.validate()
		assert.EqualValues(t, nil, err)

		ok := c.evaluate(nil)
		assert.EqualValues(t, false, ok)

		ok = c.evaluate(tax.NewSeries(nil))
		assert.EqualValues(t, true, ok)
	}
}
