package market

import (
	"errors"
	"fmt"
	"sync"

	"follow.market/internal/pkg/strategy"
	"follow.market/pkg/log"
)

type evaluator struct {
	sync.Mutex
	connected  bool
	strategies strategy.Strategies

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

func newEvaluator(participants *sharedParticipants) (*evaluator, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	e := &evaluator{
		connected:  false,
		strategies: strategy.Strategies{},

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
	e.connected = true
}

func (e *evaluator) processingWatcherReques(msg *message) {
	r := msg.request.what.(member).runner
	for _, s := range e.strategies {
		if s.Evaluate(r) {
			e.logger.Info.Println(e.newLog(r.GetName(), fmt.Sprintf("strategy matched %s", s.Name)))
		}
	}
}

func (e *evaluator) newLog(ticker, message string) string {
	return fmt.Sprintf("[evaluator] %s: %s", ticker, message)
}
