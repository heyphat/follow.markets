package market

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/dlclark/regexp2"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type trader struct {
	sync.Mutex
	connected        bool
	binSpotListenKey string
	binFutuListenKey string
	allowedPatterns  []*regexp2.Regexp

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type tdmember struct {
	runner *runner.Runner
	//bChann chan *ta.Candle
	tChann chan *tax.Trade
}

func newTrader(participants *sharedParticipants, configs *config.Configs) (*trader, error) {
	if configs == nil || participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants or configs")
	}
	t := &trader{
		connected: false,

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}
	reges := make([]*regexp2.Regexp, 0)
	for _, t := range configs.Trader.AllowedPatterns {
		reg, err := regexp2.Compile(t, 0)
		if err != nil {
			return nil, err
		}
		reges = append(reges, reg)
	}
	t.allowedPatterns = reges
	var err error
	t.binSpotListenKey, err = t.provider.binSpot.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	t.binFutuListenKey, err = t.provider.binFutu.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			t.provider.binSpot.NewKeepaliveUserStreamService().ListenKey(t.binSpotListenKey).Do(context.Background())
			t.provider.binFutu.NewKeepaliveUserStreamService().ListenKey(t.binFutuListenKey).Do(context.Background())
			time.Sleep(time.Duration(30) * time.Minute)
		}
	}()
	go t.binSpotUserDataStreaming()
	go t.binFutuUserDataStreaming()
	return t, nil
}

// isConnected returns true when the trader is connected to other market participants, false otherwise.
func (t *trader) isConnected() bool { return t.connected }

// connect connects the trader to other market participants py listening to
// decicated channels for communication.
func (t *trader) connect() {
	t.Lock()
	defer t.Unlock()
	if t.connected {
		return
	}
	go func() {
		for msg := range t.communicator.evaluator2Trader {
			go t.processEvaluatorRequest(msg)
		}
	}()
	t.connected = true
}

func (t *trader) processEvaluatorRequest(msg *message) {
	fmt.Println(msg)
}

// streamingUserData manages all account changing events from trading activities on cash account.
func (t *trader) binSpotUserDataStreaming() {
	isError, isInit := false, true
	errorHandler :=
		func(err error) { t.logger.Error.Println(t.newLog(err.Error())); isError = true }
	for isInit || isError {
		done, _, err := bn.WsUserDataServe(t.binSpotListenKey, binSpotUserDatHandler, errorHandler)
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		isError, isInit = false, false
		<-done
	}
}

func binSpotUserDatHandler(e *bn.WsUserDataEvent) {
	fmt.Println(e)
}

// streamingUserData manages all account changing events from trading activities on futures account.
func (t *trader) binFutuUserDataStreaming() {
	isError, isInit := false, true
	errorHandler :=
		func(err error) { t.logger.Error.Println(t.newLog(err.Error())); isError = true }
	for isInit || isError {
		done, _, err := bnf.WsUserDataServe(t.binFutuListenKey, binFutuUserDataHandler, errorHandler)
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		isError, isInit = false, false
		<-done
	}
}

func binFutuUserDataHandler(e *bnf.WsUserDataEvent) {
	fmt.Println(e)
}

// generates a new log with the format for the watcher
func (t *trader) newLog(message string) string {
	return fmt.Sprintf("[trader] %s", message)
}
