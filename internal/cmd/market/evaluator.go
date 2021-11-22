package market

import (
	"errors"
	"fmt"
	"sync"

	"follow.market/internal/pkg/strategy"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/log"
)

type evaluator struct {
	sync.Mutex
	connected bool
	runners   *sync.Map

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type emember struct {
	name    string
	tChann  chan *tax.Trade
	signals strategy.Signals
}

func newEvaluator(participants *sharedParticipants) (*evaluator, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	e := &evaluator{
		connected: false,
		runners:   &sync.Map{},

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}
	return e, nil
}

func (e *evaluator) connect() {
	e.Lock()
	defer e.Unlock()
	if e.connected {
		return
	}
	go func() {
		for msg := range e.communicator.watcher2Evaluator {
			go e.processingWatcherReques(msg)
		}
	}()
	go func() {
		for msg := range e.communicator.streamer2Evaluator {
			go e.processStreamerRequest(msg)
		}
	}()
	e.connected = true
}

func (e *evaluator) add(ticker string, s *strategy.Signal) {
	var mem emember
	val, ok := e.runners.Load(ticker)
	if !ok {
		mem = emember{
			name:    ticker,
			tChann:  make(chan *tax.Trade),
			signals: strategy.Signals{s},
		}
		e.runners.Store(ticker, mem)
	} else {
		mem = val.(emember)
		mem.signals = append(mem.signals, s)
		e.runners.Store(ticker, mem)
	}
	if s.IsOnTrade() {
		go e.await(mem, s)
	}
}

func (e *evaluator) await(mem emember, s *strategy.Signal) {
	for !e.registerStreamingChannel(mem) {
		e.logger.Error.Println(e.newLog(mem.name, "failed to register streaming data"))
	}
	go func() {
		for msg := range mem.tChann {
			if s.Evaluate(nil, msg) {
				e.communicator.evaluator2Notifier <- e.communicator.newMessage(s, nil)
			}
		}
	}()
}

func (e *evaluator) registerStreamingChannel(mem emember) bool {
	doneStreamingRegister := false
	var maxTries int
	for !doneStreamingRegister && maxTries <= 3 {
		resC := make(chan *payload)
		e.communicator.evaluator2Streamer <- e.communicator.newMessage(mem, resC)
		doneStreamingRegister = (<-resC).what.(bool)
		maxTries++
	}
	return doneStreamingRegister
}

func (e *evaluator) processingWatcherReques(msg *message) {
	r := msg.request.what.(wmember).runner
	val, ok := e.runners.Load(r.GetName())
	if !ok {
		return
	}
	for _, s := range val.(emember).signals {
		if s.Evaluate(r, nil) {
			e.communicator.evaluator2Notifier <- e.communicator.newMessage(s, nil)
		}
	}
}

func (e *evaluator) processStreamerRequest(msg *message) {
	if mem, ok := e.runners.Load(msg.request.what.(string)); ok && msg.response != nil {
		msg.response <- e.communicator.newPayload(mem)
		close(msg.response)
	}
}

func (e *evaluator) newLog(ticker, message string) string {
	return fmt.Sprintf("[evaluator] %s: %s", ticker, message)
}
