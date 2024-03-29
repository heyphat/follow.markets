package database

import (
	"strings"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
	ta "github.com/heyphat/techan"
)

// The database interface. Each type of database has it own implementation of these methods
// that are called on the market package.
type Client interface {
	Disconnect()
	IsInitialized() bool

	// trade setup methods
	InsertSetups(ss []*Setup) (bool, error)
	InsertOrUpdateSetups(ss []*Setup) (bool, error)
	GetSetups(r *runner.Runner, opts *QueryOptions) ([]*Setup, error)

	// signal notification methods
	InsertNotifications(ns []*Notification) (bool, error)

	// backtest methods
	InsertBacktest(bt *Backtest) error
	GetBacktest(id int64) (*Backtest, error)
	UpdateBacktestStatus(id int64, st *BacktestStatus, isResult bool) error
	UpdateBacktestResult(id int64, rs map[string]float64, ts ...*ta.Position) error
	CreateBacktestResultItem(bt *Backtest) (int64, error)
}

// Create a new db client based on user configuration options.
func NewClient(configs *config.Configs) Client {
	//if strings.ToLower(configs.Database.Use) == "mongodb" && configs.Database.MongoDB != nil {
	//	return newMongDBClient(configs)
	//}
	if strings.ToLower(configs.Database.Use) == "notion" && configs.Database.Notion != nil {
		return newNotionClient(configs)
	}
	//if configs.Database.MongoDB != nil {
	//	return newMongDBClient(configs)
	//} else if configs.Database.Notion != nil {
	if configs.Database.Notion != nil {
		return newNotionClient(configs)
	} else {
		return newNotionClient(configs)
	}
}
