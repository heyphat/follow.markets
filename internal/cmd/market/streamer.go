package market

import (
	"errors"
	"fmt"
	"sync"
	"time"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	ta "github.com/itsphat/techan"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/log"
	"follow.markets/pkg/util"
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
	return s, nil
}

type controller struct {
	name  string
	uName string
	from  []string
	stops []chan struct{}
}

// connect connects the streamer to other market participants py listening to
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
	go func() {
		for msg := range s.communicator.evaluator2Streamer {
			go s.processingEvaluatorRequest(msg)
		}
	}()
	s.connected = true
}

// isConnected returns true if the streamer is connected to the system, false otherwise.
func (s *streamer) isConnected() bool { return s.connected }

// isStreamingOn returns true if the given ticker is actually being streamed for the market
// participant given by the from parameter.
func (s *streamer) isStreamingOn(ticker, from string) bool {
	s.Lock()
	defer s.Unlock()
	valid := false
	s.controllers.Range(func(key, value interface{}) bool {
		valid = key.(string) == ticker && util.StringSliceContains(value.(controller).from, from)
		return !valid
	})
	return valid
}

// streamList returns a list of streamed tickers for a market participant given by the from parameter.
func (s *streamer) streamList(from string) []string {
	s.Lock()
	defer s.Unlock()
	tickers := []string{}
	s.controllers.Range(func(key, value interface{}) bool {
		if util.StringSliceContains(value.(controller).from, from) {
			tickers = append(tickers, key.(string))
		}
		return true
	})
	return tickers
}

// get returns a controller struct where it hass on information the streamer holds for a ticker.
func (s *streamer) get(name string) *controller {
	if val, ok := s.controllers.Load(name); ok {
		c := val.(controller)
		return &c
	}
	return nil
}

func (s *streamer) processingWatcherRequest(msg *message) {
	//s.Lock()
	//defer s.Unlock()
	m := msg.request.what.(wmember)
	if s.isStreamingOn(m.runner.GetUniqueName(), WATCHER) {
		// TODO: this only works if streamer receives request from one market participant.
		s.unsubscribe(m.runner.GetUniqueName())
		close(m.bChann)
		close(m.tChann)
	} else {
		// TODO: need to check if it is streaming for other participants
		bChann := []chan *ta.Candle{m.bChann}
		tChann := []chan *tax.Trade{m.tChann}
		from := []string{}
		c := s.get(m.runner.GetUniqueName())
		if c != nil {
			for _, f := range c.from {
				bc, tc := s.collectStreamingChannels(m.runner.GetUniqueName(), f)
				if bc != nil {
					bChann = append(bChann, bc)
				}
				if tc != nil {
					tChann = append(tChann, tc)
				}
			}
			from = c.from
			s.unsubscribe(m.runner.GetUniqueName())
		}
		bStopC, tStopC := s.subscribe(m.runner.GetName(), m.runner.GetMarketType(), bChann, tChann)
		s.controllers.Store(m.runner.GetUniqueName(),
			controller{
				name:  m.runner.GetName(),
				uName: m.runner.GetUniqueName(),
				from:  append(from, WATCHER),
				stops: []chan struct{}{bStopC, tStopC},
			},
		)
	}
	if msg.response != nil {
		msg.response <- s.communicator.newPayload(true)
		close(msg.response)
	}
}

func (s *streamer) processingEvaluatorRequest(msg *message) {
	return
	//m := msg.request.what.(emember)
	//if s.isStreamingOn(m.name, EVALUATOR) {
	//	s.logger.Info.Println(s.newLog(m.name, "already streaming this ticker"))
	//} else {
	//	bChann := []chan *ta.Candle{}
	//	tChann := []chan *tax.Trade{m.tChann}
	//	from := []string{}
	//	c := s.get(m.name)
	//	if c != nil {
	//		for _, f := range c.from {
	//			bc, tc := s.collectStreamingChannels(m.name, f)
	//			if bc != nil {
	//				bChann = append(bChann, bc)
	//			}
	//			if tc != nil {
	//				tChann = append(tChann, tc)
	//			}
	//		}
	//		from = c.from
	//		s.unsubscribe(m.name)
	//	}
	//	bStopC, tStopC := s.subscribe(m.name, bChann, tChann)
	//	s.controllers.Store(m.name,
	//		controller{
	//			name:  m.name,
	//			from:  append(from, EVALUATOR),
	//			stops: []chan struct{}{bStopC, tStopC},
	//		},
	//	)
	//}
	//if msg.response != nil {
	//	msg.response <- s.communicator.newPayload(true)
	//	close(msg.response)
	//}
}

func (s *streamer) collectStreamingChannels(name string, from string) (chan *ta.Candle, chan *tax.Trade) {
	var bChann chan *ta.Candle
	var tChann chan *tax.Trade
	resC := make(chan *payload)
	switch from {
	case WATCHER:
		s.communicator.streamer2Watcher <- s.communicator.newMessage(name, resC)
		mem := (<-resC).what.(wmember)
		bChann = mem.bChann
		tChann = mem.tChann
	case EVALUATOR:
		s.communicator.streamer2Evaluator <- s.communicator.newMessage(name, resC)
		mem := (<-resC).what.(emember)
		tChann = mem.tChann
	}
	return bChann, tChann
}

func (s *streamer) subscribe(
	name string,
	market runner.MarketType,
	bChann []chan *ta.Candle,
	tChann []chan *tax.Trade) (chan struct{}, chan struct{}) {
	s.Lock()
	defer s.Unlock()
	// cash handlers
	tradeHandler := func(event *bn.WsAggTradeEvent) {
		for _, c := range tChann {
			c <- tax.ConvertBinanceStreamingAggTrade(event)
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
	// futures handlers
	futuTradeHandler := func(event *bnf.WsAggTradeEvent) {
		for _, c := range tChann {
			c <- tax.ConvertBinanceFrturesStreamingAggTrade(event)
		}
	}
	futuKlineHandler := func(event *bnf.WsKlineEvent) {
		if !event.Kline.IsFinal {
			return
		}
		for _, c := range bChann {
			c <- tax.ConvertBinanceFuturesStreamingKline(event, nil)
		}
	}
	var bStopC, tStopC chan struct{}
	switch market {
	case runner.Cash:
		bStopC = s.streamingBinanceKline(name, bStopC, klineHandler)
		tStopC = s.streamingBinanceTrade(name, tStopC, tradeHandler)
	case runner.Futures:
		bStopC = s.streamingBinanceFuturesKline(name, bStopC, futuKlineHandler)
		tStopC = s.streamingBinanceFuturesTrade(name, tStopC, futuTradeHandler)
	}
	return bStopC, tStopC
}

func (s *streamer) unsubscribe(uName string) {
	s.Lock()
	defer s.Unlock()
	s.controllers.Range(func(key, value interface{}) bool {
		if uName == key.(string) {
			for _, c := range value.(controller).stops {
				c <- struct{}{}
			}
			return false
		}
		return true
	})
	s.controllers.Delete(uName)
}

func (s *streamer) streamingBinanceKline(name string, stop chan struct{},
	klineHandler func(e *bn.WsKlineEvent)) chan struct{} {
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())) }
	go func(stopC chan struct{}) {
		err := errors.New("not an error")
		var done chan struct{}
		for err != nil {
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

func (s *streamer) streamingBinanceFuturesKline(name string, stop chan struct{},
	klineHandler func(e *bnf.WsKlineEvent)) chan struct{} {
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())) }
	go func(stopC chan struct{}) {
		err := errors.New("not an error")
		var done chan struct{}
		for err != nil {
			done, stop, err = bnf.WsKlineServe(name, "1m", klineHandler, errorHandler)
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
	tradeHandler func(e *bn.WsAggTradeEvent)) chan struct{} {
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())) }
	go func(stopC chan struct{}) {
		err := errors.New("not an error")
		var done chan struct{}
		for err != nil {
			done, stop, err = bn.WsAggTradeServe(name, tradeHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceFuturesTrade(name string, stop chan struct{},
	tradeHandler func(e *bnf.WsAggTradeEvent)) chan struct{} {
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())) }
	go func(stopC chan struct{}) {
		err := errors.New("not an error")
		var done chan struct{}
		for err != nil {
			done, stop, err = bnf.WsAggTradeServe(name, tradeHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

// returns a log for the streamer
func (s *streamer) newLog(name, message string) string {
	return fmt.Sprintf("[streamer] %s: %s", name, message)
}
