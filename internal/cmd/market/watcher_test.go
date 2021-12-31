package market

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"follow.markets/pkg/config"
)

func Test_Watcher(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	watcher, err := newWatcher(initSharedParticipants(configs))
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, watcher.isConnected())
	assert.EqualValues(t, 0, len(watcher.watchlist()))

	ticker := "ETHUSDT"

	go func() {
		for msg := range watcher.communicator.watcher2Streamer {
			mem := msg.request.what.(wmember)
			assert.EqualValues(t, ticker, mem.runner.GetName())
			msg.response <- watcher.communicator.newPayload(true)
			close(msg.response)
		}
	}()

	err = watcher.watch(ticker, nil)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, watcher.isConnected())
	assert.EqualValues(t, true, watcher.isWatchingOn(ticker))
	assert.EqualValues(t, []string{ticker}, watcher.watchlist())
	r := watcher.get(ticker)
	assert.EqualValues(t, ticker, r.GetName())
	for _, d := range r.GetConfigs().LFrames {
		line, ok := r.GetLines(d)
		assert.EqualValues(t, true, ok)
		switch d {
		case time.Minute:
			assert.EqualValues(t, 500, len(line.Candles.Candles))
		case time.Minute * 5:
			assert.EqualValues(t, 500, len(line.Candles.Candles))
		case time.Minute * 15:
			assert.EqualValues(t, 500, len(line.Candles.Candles))
		}
	}
}
