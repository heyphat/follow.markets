package market

import (
	"time"
)

type communicator struct {
	watcher2Streamer   chan *message
	watcher2Evaluator  chan *message
	streamer2Watcher   chan *message
	streamer2Evaluator chan *message
	trader2Streamer    chan *message
	evaluator2Notifier chan *message
	evaluator2Streamer chan *message
}

func newCommunicator() *communicator {
	return &communicator{
		watcher2Streamer:   make(chan *message),
		watcher2Evaluator:  make(chan *message),
		streamer2Watcher:   make(chan *message),
		streamer2Evaluator: make(chan *message),
		trader2Streamer:    make(chan *message),
		evaluator2Notifier: make(chan *message),
		evaluator2Streamer: make(chan *message),
	}
}

type message struct {
	request  *payload
	response chan *payload
}

type payload struct {
	id   string
	when time.Time
	what interface{}
}

func (c *communicator) newMessage(data interface{}, responseChannel chan *payload) *message {
	return &message{
		request:  c.newPayload(data),
		response: responseChannel,
	}
}

func (c *communicator) newMessageWithPayloadID(id string, data interface{}, responseChannel chan *payload) *message {
	return &message{
		request:  c.newPayloadWithID(id, data),
		response: responseChannel,
	}
}

func (c *communicator) newPayload(data interface{}) *payload {
	return &payload{
		when: time.Now(),
		what: data,
	}
}

func (c *communicator) newPayloadWithID(id string, data interface{}) *payload {
	return &payload{
		id:   id,
		when: time.Now(),
		what: data,
	}
}
