package market

import (
	"time"

	"github.com/google/uuid"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	ta "github.com/itsphat/techan"
)

type communicator struct {
	watcher2Streamer   chan *message
	watcher2Evaluator  chan *message
	streamer2Watcher   chan *message
	streamer2Evaluator chan *message
	evaluator2Notifier chan *message
	evaluator2Streamer chan *message
	evaluator2Trader   chan *message
	trader2Streamer    chan *message
	trader2Notifier    chan *message
	notifier2Trader    chan *message
}

func newCommunicator() *communicator {
	return &communicator{
		watcher2Streamer:   make(chan *message),
		watcher2Evaluator:  make(chan *message),
		streamer2Watcher:   make(chan *message),
		streamer2Evaluator: make(chan *message),
		evaluator2Notifier: make(chan *message),
		evaluator2Streamer: make(chan *message),
		evaluator2Trader:   make(chan *message, 10),
		trader2Streamer:    make(chan *message),
		trader2Notifier:    make(chan *message),
		notifier2Trader:    make(chan *message),
	}
}

type message struct {
	request  *payload
	response chan *payload
}

type payload struct {
	requestID  uuid.UUID
	responseID uuid.UUID

	what data
	when time.Time
}

type data struct {
	runner   *runner.Runner
	signal   *strategy.Signal
	channels *streamingChannels

	dynamic interface{}
}

func (c *communicator) newMessage(
	r *runner.Runner,
	s *strategy.Signal,
	cs *streamingChannels,
	u interface{},
	responseChannel chan *payload) *message {
	return &message{
		request:  c.newPayload(r, s, cs, u).addRequestID(nil),
		response: responseChannel,
	}
}

func (c *communicator) newPayload(
	r *runner.Runner,
	s *strategy.Signal,
	cs *streamingChannels,
	u interface{}) *payload {
	data := data{}
	if r != nil {
		data.runner = r
	}
	if s != nil {
		data.signal = s
	}
	if cs != nil {
		data.channels = cs
	}
	if u != nil {
		data.dynamic = u
	}
	return &payload{
		what: data,
		when: time.Now(),
	}
}

func (pl *payload) addRequestID(id *uuid.UUID) *payload {
	if id != nil {
		pl.responseID = *id
	} else {
		pl.requestID = uuid.New()
	}
	return pl
}

func (pl *payload) addResponseID() *payload {
	pl.responseID = uuid.New()
	return pl
}

type streamingChannels struct {
	bar   chan *ta.Candle
	trade chan *tax.Trade
	depth chan interface{}
}

func (scs *streamingChannels) close() {
	if scs == nil {
		return
	}
	if scs.bar != nil {
		close(scs.bar)
	}
	if scs.trade != nil {
		close(scs.trade)
	}
	if scs.depth != nil {
		close(scs.depth)
	}
}
