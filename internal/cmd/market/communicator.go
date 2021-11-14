package market

import (
	"time"

	"github.com/google/uuid"
)

type communicator struct {
	watcher2Streamer chan *message
	streamer2Watcher chan *message
	trader2Streamer  chan *message
}

func newCommunicator() *communicator {
	return &communicator{
		watcher2Streamer: make(chan *message),
		streamer2Watcher: make(chan *message),
		trader2Streamer:  make(chan *message),
	}
}

type message struct {
	request  *payload
	response chan *payload
}

type payload struct {
	uuid uuid.UUID
	when time.Time
	what interface{}
}

func (c *communicator) newMessage(data interface{}, responseChannel chan *payload) *message {
	return &message{
		request:  c.newPayload(data),
		response: responseChannel,
	}
}

func (c *communicator) newPayload(data interface{}) *payload {
	return &payload{
		uuid: uuid.New(),
		when: time.Now(),
		what: data,
	}
}
