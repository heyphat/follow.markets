package market

import (
	"time"

	"github.com/google/uuid"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	ta "github.com/itsphat/techan"
)

// this agent handles all the communications between other agents.
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

// this returns a communicator with initialized communicating channels.
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

// the messge structure to communicate between agents.
// an agent can ask for a request and expect a response, a response channel needs to be added.
// somtimes if an agent doesn't expect the response, the response channel can be nil.
type message struct {
	request  *payload
	response chan *payload
}

// the message's payload structure.
type payload struct {
	requestID  uuid.UUID
	responseID uuid.UUID

	what data
	when time.Time
}

// the data of the payload, agents are passing around runner, signal and streaming channels,
// sometimes with an unidentified type of data.
type data struct {
	runner   *runner.Runner
	signal   *strategy.Signal
	channels *streamingChannels

	dynamic interface{}
}

// this is a handy method to create a message.
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

// this is a handy method to create a payload.
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

// this method adds a unique ID to a request. We might need it later.
func (pl *payload) addRequestID(id *uuid.UUID) *payload {
	if id != nil {
		pl.responseID = *id
	} else {
		pl.requestID = uuid.New()
	}
	return pl
}

// this method adds a unique ID to a response. We might need it later.
func (pl *payload) addResponseID() *payload {
	pl.responseID = uuid.New()
	return pl
}

// strreaming channels of a payload data if agents request for streaming data.
type streamingChannels struct {
	bar   chan *ta.Candle
	trade chan *tax.Trade
	depth chan interface{}
}

// This method is to close all the streaming channels when an agent is done with streaming data.
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
