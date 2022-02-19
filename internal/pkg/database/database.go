package database

import (
	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
)

type Client interface {
	Disconnect()
	IsInitialized() bool
	InsertSetups(ss []*Setup) (bool, error)
	InsertOrUpdateSetups(ss []*Setup) (bool, error)
	GetSetups(r *runner.Runner, opts *QueryOptions) ([]*Setup, error)
}

func NewClient(configs *config.Configs) Client {
	if configs.Database.MongoDB != nil {
		return newMongDBClient(configs)
	}
	return MongoDB{}
}
