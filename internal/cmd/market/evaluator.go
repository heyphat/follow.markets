package market

import (
	"errors"
	"fmt"
	"sync"

	"github.com/dlclark/regexp2"

	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/log"
	"follow.markets/pkg/util"
)

type evaluator struct {
	sync.Mutex
	connected bool
	signals   *sync.Map

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type emember struct {
	name     string
	regex    []*regexp2.Regexp
	tChann   chan *tax.Trade
	signals  strategy.Signals
	patterns []string
}

func newEvaluator(participants *sharedParticipants) (*evaluator, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	e := &evaluator{
		connected: false,
		signals:   &sync.Map{},

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
			go e.processingWatcherRequest(msg)
		}
	}()
	go func() {
		for msg := range e.communicator.streamer2Evaluator {
			go e.processStreamerRequest(msg)
		}
	}()
	e.connected = true
}

// add adds a new signal to the evalulator. The evaluator will evaluate the signal
// every minute on all tickers that match the given patterns.
func (e *evaluator) add(patterns []string, s *strategy.Signal) error {
	e.Lock()
	defer e.Unlock()

	var mem emember
	val, ok := e.signals.Load(s.Name)
	if !ok {
		reges := make([]*regexp2.Regexp, 0)
		for _, t := range patterns {
			reg, err := regexp2.Compile(t, 0)
			if err != nil {
				return err
			}
			reges = append(reges, reg)
		}
		mem = emember{
			name:     s.Name,
			regex:    reges,
			tChann:   nil,
			signals:  strategy.Signals{s},
			patterns: patterns,
			//tChann:  make(chan *tax.Trade),
		}
		e.signals.Store(s.Name, mem)
	} else {
		mem = val.(emember)
		mem.signals = append(mem.signals, s)
		e.signals.Store(s.Name, mem)
	}
	//if s.IsOnTrade() {
	//	go e.await(mem, s)
	//}
	return nil
}

// drop removes the given signal from the evaluator. After the removal, the singal won't be
// evaluated any longer.
func (e *evaluator) drop(name string) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.signals.Load(name); !ok {
		return nil
	}
	e.signals.Delete(name)
	return nil
}

// getByTicker returns a slice of signals that are applicable to the given ticker.
func (e *evaluator) getByTicker(ticker string) strategy.Signals {
	out := strategy.Signals{}
	e.signals.Range(func(k, v interface{}) bool {
		m := v.(emember)
		for _, re := range m.regex {
			if isMatched, err := re.MatchString(ticker); err == nil && isMatched {
				out = append(out, m.signals.Copy()...)
			}
		}
		return true
	})
	return out
}

// getByName return a slice of signals with the given name.
func (e *evaluator) getByNames(names []string) strategy.Signals {
	out := strategy.Signals{}
	e.signals.Range(func(k, v interface{}) bool {
		if len(names) == 0 || util.StringSliceContains(names, k.(string)) {
			out = append(out, v.(emember).signals.Copy()...)
		}
		return true
	})
	return out
}

//func (e *evaluator) await(mem emember, s *strategy.Signal) {
//	for !e.registerStreamingChannel(mem) {
//		e.logger.Error.Println(e.newLog(mem.name, "failed to register streaming data"))
//	}
//	go func() {
//		for msg := range mem.tChann {
//			if s.Evaluate(nil, msg) {
//				e.communicator.evaluator2Notifier <- e.communicator.newMessage(s, nil)
//			}
//		}
//	}()
//}

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

func (e *evaluator) processingWatcherRequest(msg *message) {
	r := msg.request.what.(wmember).runner
	signals := e.getByTicker(r.GetName())
	for _, s := range signals {
		if s.Evaluate(r, nil) {
			e.communicator.evaluator2Notifier <- e.communicator.newMessageWithPayloadID(r.GetUniqueName()+"-"+s.Name, s, nil)
			if s.IsOnetime() {
				_ = e.drop(s.Name)
			}
		}
	}
}

func (e *evaluator) processStreamerRequest(msg *message) {
	if mem, ok := e.signals.Load(msg.request.what.(string)); ok && msg.response != nil {
		msg.response <- e.communicator.newPayload(mem)
		close(msg.response)
	}
}

func (e *evaluator) newLog(ticker, message string) string {
	return fmt.Sprintf("[evaluator] %s: %s", ticker, message)
}
