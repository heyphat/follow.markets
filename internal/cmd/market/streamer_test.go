package market

import (
	"fmt"
	"testing"
	"time"

	ta "github.com/itsphat/techan"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Streamer(t *testing.T) {
	path := "./../../../configs/dev_configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	streamer, err := newStreamer(initSharedParticipants(configs))
	streamer.connect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, streamer.isConnected())

	btcT := "BTCUSDT"
	ethT := "ETHUSDT"
	btc := member{
		runner: runner.NewRunner(btcT, nil),
		bChann: make(chan *ta.Candle, 2),
		tChann: make(chan *tax.Trade, 2),
	}
	eth := member{
		runner: runner.NewRunner(ethT, nil),
		bChann: make(chan *ta.Candle, 2),
		tChann: make(chan *tax.Trade, 2),
	}
	go func() {
		for msg := range btc.bChann {
			fmt.Println("btc bar", msg)
		}
	}()
	go func() {
		for msg := range eth.bChann {
			fmt.Println("eth bar", msg)
		}
	}()
	go func() {
		for msg := range eth.tChann {
			fmt.Println("eth trade", msg)
		}
	}()
	go func() {
		for msg := range btc.tChann {
			fmt.Println("btc trade", msg)
		}
	}()

	streamer.communicator.watcher2Streamer <- streamer.communicator.newMessage(btc, nil)
	streamer.communicator.watcher2Streamer <- streamer.communicator.newMessage(eth, nil)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 2, len(streamer.streamList()))

	streamer.unsubscribe(btcT)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 1, len(streamer.streamList()))
}
