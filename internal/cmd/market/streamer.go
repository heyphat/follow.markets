package market

import (
	"errors"
	"fmt"
	"sync"
	"time"

	bn "github.com/adshao/go-binance/v2"
	ta "github.com/itsphat/techan"

	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/log"
)

type streamer struct {
	sync.Mutex
	connected   bool
	controllers *sync.Map

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

func newStreamer(participants *sharedParticipants) (*streamer, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	s := &streamer{
		connected:   false,
		controllers: &sync.Map{},

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}
	s.connect()
	return s, nil
}

type controller struct {
	name  string
	from  string
	stops []chan struct{}
}

// Connect connects the streamer to other market participants py listening to
// a decicated channels for the communication.
func (s *streamer) connect() {
	s.Lock()
	defer s.Unlock()
	if s.connected {
		return
	}
	go func() {
		for msg := range s.communicator.watcher2Streamer {
			go s.processingWatcherRequest(msg)
		}
	}()
	s.connected = true
}

func (s *streamer) isStreamingOn(ticker, from string) bool {
	valid := false
	s.controllers.Range(func(key, value interface{}) bool {
		valid = key.(string) == ticker && from == value.(controller).from
		return !valid
	})
	return valid
}

func (s *streamer) streamList() []string {
	tickers := []string{}
	s.controllers.Range(func(key, value interface{}) bool {
		if value.(controller).from == WATCHER {
			tickers = append(tickers, key.(string))
		}
		return true
	})
	return tickers
}

func (s *streamer) isConnected() bool { return s.connected }

func (s *streamer) processingWatcherRequest(msg *message) {
	m := msg.request.what.(member)
	if s.isStreamingOn(m.runner.GetName(), WATCHER) {
		return
	}
	// TODO: need to check if it is streaming for other participants
	bStopC, tStopC := s.subscribe(m.runner.GetName(),
		[]chan *ta.Candle{m.bChann},
		[]chan *tax.Trade{m.tChann})
	s.controllers.Store(m.runner.GetName(),
		controller{
			name:  m.runner.GetName(),
			from:  WATCHER,
			stops: []chan struct{}{bStopC, tStopC},
		},
	)
	if msg.response != nil {
		msg.response <- s.communicator.newPayload(true)
		close(msg.response)
	}
}

func (s *streamer) subscribe(name string,
	bChann []chan *ta.Candle,
	tChann []chan *tax.Trade) (chan struct{}, chan struct{}) {
	s.Lock()
	defer s.Unlock()
	tradeHandler := func(event *bn.WsTradeEvent) {
		for _, c := range tChann {
			c <- tax.ConvertBinanceStreamingTrade(event)
		}
	}
	klineHandler := func(event *bn.WsKlineEvent) {
		if !event.Kline.IsFinal {
			return
		}
		for _, c := range bChann {
			c <- tax.ConvertBinanceStreamingKline(event, nil)
		}
	}
	var bStopC, tStopC chan struct{}
	bStopC = s.streamingBinanceKline(name, bStopC, klineHandler)
	tStopC = s.streamingBinanceTrade(name, tStopC, tradeHandler)
	return bStopC, tStopC
}

func (s *streamer) unsubscribe(name string) {
	s.Lock()
	defer s.Unlock()
	s.controllers.Range(func(key, value interface{}) bool {
		if name == key.(string) {
			for _, c := range value.(controller).stops {
				c <- struct{}{}
			}
			return false
		}
		return true
	})
	s.controllers.Delete(name)
}

func (s *streamer) streamingBinanceKline(name string, stop chan struct{},
	klineHandler func(e *bn.WsKlineEvent)) chan struct{} {
	errorHandler := func(err error) { s.logger.Error.Println(err) }
	go func(stopC chan struct{}) {
		var err error
		var done chan struct{}
		for {
			done, stop, err = bn.WsKlineServe(name, "1m", klineHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceTrade(name string, stop chan struct{},
	tradeHandler func(e *bn.WsTradeEvent)) chan struct{} {
	go func(stopC chan struct{}) {
		errorHandler := func(err error) { s.logger.Error.Println(err) }
		var err error
		var done chan struct{}
		for {
			done, stop, err = bn.WsTradeServe(name, tradeHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) newLog(ticker, message string) string {
	return fmt.Sprintf("[watcher] %s: %s", ticker, message)
}
