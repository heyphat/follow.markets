package market

import (
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

	strategyPath := "./../../pkg/strategy/signals/signal_trade.json"
	sraw, err := ioutil.ReadFile(strategyPath)
	assert.EqualValues(t, nil, err)

	s, err := strategy.NewSignalFromBytes(sraw)
	assert.EqualValues(t, nil, err)

	market.evaluator.add("ETHUSDT", s)

	time.Sleep(time.Second * 5)

	err = market.Watch("ETHUSDT")
	assert.EqualValues(t, nil, err)

	watchlist = market.Watchlist()
	assert.EqualValues(t, 2, len(watchlist))

	//for {
	//	lastCandles := market.LastCandles("ETHUSDT")
	//	fmt.Println(lastCandles)
	//	time.Sleep(time.Minute)
	//}
}
