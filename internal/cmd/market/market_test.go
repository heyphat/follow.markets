package market

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Market(t *testing.T) {
	path := "./../../../configs/dev_configs.json"
	market, err := NewMarket(&path)
	assert.EqualValues(t, nil, err)

	assert.EqualValues(t, true, market.watcher.connected)
	assert.EqualValues(t, true, market.streamer.connected)
	assert.EqualValues(t, true, market.evaluator.connected)
	assert.EqualValues(t, true, market.notifier.connected)

	watchlist := market.Watchlist()
	assert.EqualValues(t, 1, len(watchlist))

	err = market.Watch("ETHUSDT")
	assert.EqualValues(t, nil, err)

	watchlist = market.Watchlist()
	assert.EqualValues(t, 2, len(watchlist))
}
