package market

import (
	"fmt"
	"testing"
	"time"

	ta "github.com/itsphat/techan"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Streamer(t *testing.T) {
	// comment return to test streamer
	//return
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	shared := initSharedParticipants(configs)
	streamer, err := newStreamer(shared)
	streamer.connect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, streamer.isConnected())

	btcT := "BTCUSDT"
	ethT := "ETHUSDT"
	btc := wmember{
		runner: runner.NewRunner(btcT, nil),
		channels: &streamingChannels{
			bar:   make(chan *ta.Candle, 2),
			trade: make(chan *tax.Trade, 2),
		},
	}
	eth := wmember{
		runner: runner.NewRunner(ethT, nil),
		channels: &streamingChannels{
			bar:   make(chan *ta.Candle, 2),
			trade: make(chan *tax.Trade, 2),
		},
	}
	go func() {
		for msg := range btc.channels.bar {
			fmt.Println("btc bar", msg)
		}
	}()
	go func() {
		for msg := range eth.channels.trade {
			fmt.Println("eth bar", msg)
		}
	}()
	go func() {
		for msg := range eth.channels.bar {
			fmt.Println("eth trade", msg)
		}
	}()
	go func() {
		for msg := range btc.channels.trade {
			fmt.Println("btc trade", msg)
		}
	}()

	streamer.communicator.watcher2Streamer <- streamer.communicator.newMessage(btc.runner, nil, btc.channels, nil, nil)
	streamer.communicator.watcher2Streamer <- streamer.communicator.newMessage(eth.runner, nil, eth.channels, nil, nil)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 2, len(streamer.streamList(WATCHER)))

	streamer.communicator.watcher2Streamer <- streamer.communicator.newMessage(btc.runner, nil, btc.channels, nil, nil)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 1, len(streamer.streamList(WATCHER)))

	//go func() {
	//	for msg := range streamer.communicator.streamer2Watcher {
	//		if msg.request.what.dynamic.(string) == btcT {
	//			msg.response <- streamer.communicator.newPayload(btc)
	//		}
	//		close(msg.response)
	//	}
	//}()

	//btcE := emember{
	//	name: btcT,
	//	channels: &streamingChannels{
	//		trade: make(chan *tax.Candle, 2),
	//	},
	//	signals: strategy.Signals{},
	//}

	//go func() {
	//	for msg := range btcE.channels.trade {
	//		fmt.Println("btc trade from evaluator", msg)
	//	}
	//}()

	//streamer.communicator.evaluator2Streamer <- streamer.communicator.newMessage(nil, nil, nil, nil, nil)
	//time.Sleep(time.Second)
	//assert.EqualValues(t, 1, len(streamer.streamList(EVALUATOR)))

	//manaE := emember{
	//	name:    manaT,
	//	signals: strategy.Signals{},
	//	channels: &streamingChannels{
	//		trade: make(chan *tax.Candle, 2),
	//	},
	//}

	//go func() {
	//	for msg := range manaE.channels.trade {
	//		fmt.Println("mana trade from evaluator", msg)
	//	}
	//}()
	//streamer.communicator.evaluator2Streamer <- streamer.communicator.newMessage(nil, nil, nil, nil, nil)
	//time.Sleep(time.Second * 5)
	//assert.EqualValues(t, 2, len(streamer.streamList(EVALUATOR)))

	//streamer.communicator.evaluator2Streamer <- streamer.communicator.newMessage(nil, nil, nil, nil, nil)
	//time.Sleep(time.Second * 2)
	//assert.EqualValues(t, 1, len(streamer.streamList(EVALUATOR)))

}
