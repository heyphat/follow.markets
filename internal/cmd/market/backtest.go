package market

import (
	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"

	db "follow.markets/internal/pkg/database"
	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
)

// the tester tests on a backtest.
type backtest struct {
	balance       big.Decimal
	lossTolerance big.Decimal
	profitMargin  big.Decimal

	bt  *db.Backtest
	r   *runner.Runner
	s   *strategy.Strategy
	rcs *ta.TradingRecord
}

func newBacktest(bt *db.Backtest) *backtest {
	out := &backtest{
		bt:            bt,
		balance:       big.NewDecimal(float64(bt.Balance)),
		lossTolerance: big.NewDecimal(bt.LossTolerance),
		profitMargin:  big.NewDecimal(bt.ProfitMargin),
	}
	out.s = &strategy.Strategy{
		ExitRule:       nil,
		EntryRule:      strategy.NewRule(*bt.Signal),
		RiskRewardRule: strategy.NewRiskRewardRule(-bt.LossTolerance, bt.ProfitMargin),
	}
	out.r = runner.NewRunner(bt.Ticker, &runner.RunnerConfigs{
		LFrames:  bt.Signal.GetPeriods(),
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	})
	out.s.SetRunner(out.r)
	out.rcs = ta.NewTradingRecord()
	out.bt.UpdateStatus(db.BacktestStatusProcessing)
	return out
}

func (b backtest) Summary() {
}
