package database

import (
	"strings"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
)

type Backtest struct {
	ID            int64             `bson:"id" json:"id"`
	Name          string            `bson:"name" json:"backtest"`
	Ticker        string            `bson:"ticker" json:"ticker"`
	Balance       int64             `bson:"balance" json:"balance"`
	Market        runner.MarketType `bson:"market" json:"market"`
	LossTolerance float64           `bson:"loss_tolerance" json:"loss_tolerance"`
	ProfitMargin  float64           `bson:"profit_margin" json:"profit_margin"`
	Signal        *strategy.Signal  `bson:"signal" json:"signal"`
	Start         time.Time         `bson:"start" json:"start"`
	End           time.Time         `bson:"end" json:"end"`
	CreatedAt     time.Time         `bson:"created_at" json:"created_at"`
	Status        BacktestStatus    `bson:"status" json:"status"`
}

func (bt *Backtest) UpdateStatus(s BacktestStatus) { bt.Status = s }

type BacktestStatus string

const (
	BacktestStatusError      BacktestStatus = "ERROR"
	BacktestStatusCompleted  BacktestStatus = "DONE"
	BacktestStatusAccepted   BacktestStatus = "ACCEPTED"
	BacktestStatusProcessing BacktestStatus = "PROCESSING"
	BacktestStatusUnknown    BacktestStatus = "UNKNOWN"
)

func (bs BacktestStatus) String() string { return string(bs) }

func ValidateBacktestStatus(s string) (BacktestStatus, bool) {
	switch strings.ToUpper(s) {
	case "ERROR":
		return BacktestStatusError, true
	case "DONE":
		return BacktestStatusCompleted, true
	case "ACCEPTED":
		return BacktestStatusAccepted, true
	case "PROCESSING":
		return BacktestStatusProcessing, true
	default:
		return BacktestStatusUnknown, false
	}
}

type BacktestResult struct{}
