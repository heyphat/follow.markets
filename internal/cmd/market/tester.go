package market

import (
	"errors"
	"sync"

	"follow.market/pkg/log"
)

type tester struct {
	sync.Mutex
	connected bool

	// shared properties with other market participants
	logger   *log.Logger
	provider *provider
}

func newTester(participants *sharedParticipants) (*tester, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &tester{
		connected: false,

		logger:   participants.logger,
		provider: participants.provider,
	}, nil
}

func (t *tester) test() {}
