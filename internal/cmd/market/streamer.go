package market

import (
	"errors"
	"fmt"
	"sync"
	"time"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/log"
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

// newStreamer returns a streamer, meant to be called by the MarketStruct only once.
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
	from  string
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
	go func() {
		for msg := range s.communicator.trader2Streamer {
			go s.processingTraderRequest(msg)
		}
	}()
	s.connected = true
}

// isConnected returns true if the streamer is connected to the system, false otherwise.
func (s *streamer) isConnected() bool { return s.connected }

// isStreamingOn returns true if the given ticker is actually being streamed for a market
// participant given by the `from` parameter.
func (s *streamer) isStreamingOn(ticker, from string) bool {
	s.Lock()
	defer s.Unlock()
	valid := false
	s.controllers.Range(func(key, value interface{}) bool {
		valid = key.(string) == ticker && value.(controller).from == from
		return !valid
	})
	return valid
}

// streamList returns a list of streamed tickers for a market participant given by the `from` parameter.
func (s *streamer) streamList(from string) []string {
	s.Lock()
	defer s.Unlock()
	tickers := []string{}
	s.controllers.Range(func(key, value interface{}) bool {
		if value.(controller).from == from {
			tickers = append(tickers, key.(string))
		}
		return true
	})
	return tickers
}

// get returns a controller that the streamer holds for a runner.
func (s *streamer) get(name string) *controller {
	if val, ok := s.controllers.Load(name); ok {
		c := val.(controller)
		return &c
	}
	return nil
}

// this method processes requests from the watcher.
func (s *streamer) processingWatcherRequest(msg *message) {
	r := msg.request.what.runner
	cs := msg.request.what.channels
	if s.isStreamingOn(r.GetUniqueName(WATCHER), WATCHER) {
		s.unsubscribe(r.GetUniqueName(WATCHER))
		cs.close()
	} else {
		bStopC, tStopC, dStopC := s.subscribe(r, cs)
		s.controllers.Store(r.GetUniqueName(WATCHER),
			controller{
				name:  r.GetUniqueName(WATCHER),
				from:  WATCHER,
				stops: []chan struct{}{bStopC, tStopC, dStopC},
			},
		)
	}
	if msg.response != nil {
		msg.response <- s.communicator.newPayload(nil, nil, nil, true).addRequestID(&msg.request.requestID).addResponseID()
		close(msg.response)
	}
}

// this method processes requests from the trader.
func (s *streamer) processingTraderRequest(msg *message) {
	r := msg.request.what.runner
	cs := msg.request.what.channels
	if s.isStreamingOn(r.GetUniqueName(TRADER), TRADER) {
		s.unsubscribe(r.GetUniqueName(TRADER))
		cs.close()
	} else {
		bStopC, tStopC, dStopC := s.subscribe(r, cs)
		s.controllers.Store(r.GetUniqueName(TRADER),
			controller{
				name:  r.GetUniqueName(TRADER),
				from:  TRADER,
				stops: []chan struct{}{bStopC, tStopC, dStopC},
			},
		)
	}
	if msg.response != nil {
		msg.response <- s.communicator.newPayload(nil, nil, nil, true).addRequestID(&msg.request.requestID).addResponseID()
		close(msg.response)
	}
}

// this method processes requests from the evaluator.
func (s *streamer) processingEvaluatorRequest(msg *message) {
	//m := msg.request.what.(emember)
	//	if s.isStreamingOn(EVALUATOR+m.name, EVALUATOR) {
	//		s.unsubscribe(EVALUATOR + m.name)
	//		close(m.tChann)
	//	} else {
	//		//TODO: it's not always cash market
	//		bStopC, tStopC := s.subscribe(m.name, runner.Cash, nil, m.tChann)
	//		s.controllers.Store(EVALUATOR+m.name,
	//			controller{
	//				name:  m.name,
	//				uName: EVALUATOR + m.name,
	//				from:  EVALUATOR,
	//				stops: []chan struct{}{bStopC, tStopC},
	//			},
	//		)
	//	}
	if msg.response != nil {
		msg.response <- s.communicator.newPayload(nil, nil, nil, true).addRequestID(&msg.request.requestID).addResponseID()
		close(msg.response)
	}
}

// subscribe handles subscribing to the market data for a runner.
func (s *streamer) subscribe(r *runner.Runner, cs *streamingChannels) (chan struct{}, chan struct{}, chan struct{}) {
	s.Lock()
	defer s.Unlock()
	// cash handlers
	tradeHandler := func(event *bn.WsAggTradeEvent) {
		if cs.trade != nil {
			cs.trade <- tax.ConvertBinanceStreamingAggTrade(event)
		}
	}
	klineHandler := func(event *bn.WsKlineEvent) {
		if !event.Kline.IsFinal {
			return
		}
		if event.Kline.TradeNum == 0 || big.NewFromString(event.Kline.Volume).EQ(big.ZERO) {
			return
		}
		if cs.bar != nil {
			cs.bar <- tax.ConvertBinanceStreamingKline(event, nil)
		}
	}
	depthHandler := func(event *bn.WsPartialDepthEvent) {
		if cs.depth != nil {
			cs.depth <- event
		}
	}
	// futures handlers
	futuTradeHandler := func(event *bnf.WsAggTradeEvent) {
		if cs.trade != nil {
			cs.trade <- tax.ConvertBinanceFuturesStreamingAggTrade(event)
		}
	}
	futuKlineHandler := func(event *bnf.WsKlineEvent) {
		if !event.Kline.IsFinal {
			return
		}
		if event.Kline.TradeNum == 0 || big.NewFromString(event.Kline.Volume).EQ(big.ZERO) {
			return
		}
		if cs.bar != nil {
			cs.bar <- tax.ConvertBinanceFuturesStreamingKline(event, nil)
		}
	}
	futuDepthHandler := func(event *bnf.WsDepthEvent) {
		if cs.depth != nil {
			cs.depth <- event
		}
	}
	var bStopC, tStopC, dStopC chan struct{}
	switch r.GetMarketType() {
	case runner.Cash:
		if cs.bar != nil {
			bStopC = s.streamingBinanceKline(r.GetName(), bStopC, klineHandler)
		}
		if cs.trade != nil {
			tStopC = s.streamingBinanceTrade(r.GetName(), tStopC, tradeHandler)
		}
		if cs.depth != nil {
			dStopC = s.streamingBinancePartitialDepth(r.GetName(), dStopC, depthHandler)
		}
	case runner.Futures:
		if cs.bar != nil {
			bStopC = s.streamingBinanceFuturesKline(r.GetName(), bStopC, futuKlineHandler)
		}
		if cs.trade != nil {
			tStopC = s.streamingBinanceFuturesTrade(r.GetName(), tStopC, futuTradeHandler)
		}
		if cs.depth != nil {
			dStopC = s.streamingBinanceFuturesPartitialDepth(r.GetName(), dStopC, futuDepthHandler)
		}
	}
	return bStopC, tStopC, dStopC
}

// unsubscribe handles unsubscribing to the market data for a runner.
func (s *streamer) unsubscribe(name string) {
	s.Lock()
	defer s.Unlock()
	s.controllers.Range(func(key, value interface{}) bool {
		if name == key.(string) {
			for _, c := range value.(controller).stops {
				if c != nil {
					c <- struct{}{}
				}
			}
			return false
		}
		return true
	})
	s.controllers.Delete(name)
}

func (s *streamer) streamingBinanceKline(name string, stop chan struct{},
	klineHandler func(e *bn.WsKlineEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func(stop chan struct{}) {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bn.WsKlineServe(name, "1m", klineHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceFuturesKline(name string, stop chan struct{},
	klineHandler func(e *bnf.WsKlineEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func(stop chan struct{}) {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bnf.WsKlineServe(name, "1m", klineHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceTrade(name string, stop chan struct{},
	tradeHandler func(e *bn.WsAggTradeEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func(stop chan struct{}) {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bn.WsAggTradeServe(name, tradeHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceFuturesTrade(name string, stop chan struct{},
	tradeHandler func(e *bnf.WsAggTradeEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func(stop chan struct{}) {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bnf.WsAggTradeServe(name, tradeHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}(stop)
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinancePartitialDepth(name string,
	stop chan struct{}, depthHandler func(e *bn.WsPartialDepthEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func() {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bn.WsPartialDepthServe(name, "5", depthHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}()
	time.Sleep(time.Second)
	return stop
}

func (s *streamer) streamingBinanceFuturesPartitialDepth(name string,
	stop chan struct{}, depthHandler func(e *bnf.WsDepthEvent)) chan struct{} {
	isError, isInit := false, true
	errorHandler := func(err error) { s.logger.Error.Println(s.newLog(name, err.Error())); isError = true }
	go func() {
		var err error
		var done chan struct{}
		for isInit || isError {
			done, stop, err = bnf.WsPartialDepthServeWithRate(name, 5, time.Duration(100*time.Millisecond), depthHandler, errorHandler)
			if err != nil {
				s.logger.Error.Println(s.newLog(name, err.Error()))
			}
			isInit, isError = false, false
			<-done
		}
	}()
	// DO NOT remove the sleep, otherwise the channel won't be initialized
	time.Sleep(time.Second)
	return stop
}

// returns a log for the streamer.
func (s *streamer) newLog(name, message string) string {
	return fmt.Sprintf("[streamer] %s: %s", name, message)
}
