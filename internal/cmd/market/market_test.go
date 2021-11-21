package market

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"follow.market/internal/pkg/strategy"
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

	strategyPath := "./../../pkg/strategy/strategy_trade.json"
	sraw, err := ioutil.ReadFile(strategyPath)
	assert.EqualValues(t, nil, err)

	s, err := strategy.NewStrategyFromBytes(sraw)
	assert.EqualValues(t, nil, err)

	market.evaluator.add("ETHUSDT", s)

	fmt.Println("here")

	time.Sleep(time.Minute)

}
