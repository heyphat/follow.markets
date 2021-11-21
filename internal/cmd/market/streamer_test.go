package market

import (
	"fmt"
	"testing"
	"time"

	ta "github.com/itsphat/techan"

	"follow.market/internal/pkg/runner"
	"follow.market/internal/pkg/strategy"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Streamer(t *testing.T) {
	// comment return to test streamer
	//return
	path := "./../../../configs/dev_configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	shared := initSharedParticipants(configs)
	streamer, err := newStreamer(shared)
	streamer.connect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, streamer.isConnected())

	btcT := "BTCUSDT"
	ethT := "ETHUSDT"
	manaT := "MANAUSDT"
	btc := wmember{
		runner: runner.NewRunner(btcT, nil),
		bChann: make(chan *ta.Candle, 2),
		tChann: make(chan *tax.Trade, 2),
	}
	eth := wmember{
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
	assert.EqualValues(t, 2, len(streamer.streamList(WATCHER)))

	streamer.unsubscribe(btcT)
	time.Sleep(time.Second)
	assert.EqualValues(t, 1, len(streamer.streamList(WATCHER)))

	go func() {
		for msg := range streamer.communicator.streamer2Watcher {
			if msg.request.what.(string) == btcT {
				msg.response <- streamer.communicator.newPayload(btc)
			}
			close(msg.response)
		}
	}()

	btcE := emember{
		name:       btcT,
		tChann:     make(chan *tax.Trade, 2),
		strategies: strategy.Strategies{},
	}

	go func() {
		for msg := range btcE.tChann {
			fmt.Println("btc trade from evaluator", msg)
		}
	}()

	streamer.communicator.evaluator2Streamer <- streamer.communicator.newMessage(btcE, nil)
	time.Sleep(time.Second)
	assert.EqualValues(t, 1, len(streamer.streamList(EVALUATOR)))

	manaE := emember{
		name:       manaT,
		tChann:     make(chan *tax.Trade, 2),
		strategies: strategy.Strategies{},
	}

	go func() {
		for msg := range manaE.tChann {
			fmt.Println("mana trade from evaluator", msg)
		}
	}()
	streamer.communicator.evaluator2Streamer <- streamer.communicator.newMessage(manaE, nil)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 2, len(streamer.streamList(EVALUATOR)))
}
