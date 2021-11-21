package market

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"follow.market/pkg/config"
)

func Test_Watcher(t *testing.T) {
	path := "./../../../configs/dev_configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	watcher, err := newWatcher(initSharedParticipants(configs))
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, watcher.isConnected())
	assert.EqualValues(t, 0, len(watcher.watchlist()))

	go func() {
		for msg := range watcher.communicator.watcher2Streamer {
			mem := msg.request.what.(wmember)
			assert.EqualValues(t, "BTCUSDT", mem.runner.GetName())
			msg.response <- watcher.communicator.newPayload(true)
			close(msg.response)
		}
	}()

	err = watcher.watch("BTCUSDT")
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, watcher.isConnected())
	assert.EqualValues(t, true, watcher.isWatchingOn("BTCUSDT"))
	assert.EqualValues(t, []string{"BTCUSDT"}, watcher.watchlist())
	r := watcher.get("BTCUSDT")
	assert.EqualValues(t, "BTCUSDT", r.GetName())
	for _, d := range r.GetConfigs().LFrames {
		line, ok := r.GetLines(d)
		assert.EqualValues(t, true, ok)
		switch d {
		case time.Minute:
			assert.EqualValues(t, 6000, len(line.Candles.Candles))
			//case time.Minute * 5:
			//	assert.EqualValues(t, 6000/5+1, len(line.Candles.Candles))
			//case time.Minute * 15:
			//	assert.EqualValues(t, 6000/15+1, len(line.Candles.Candles))
		}
	}
}
