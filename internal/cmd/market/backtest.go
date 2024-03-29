package market

import (
	"math"
	"time"

	ta "github.com/heyphat/techan"
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
		RiskRewardRule: strategy.NewRiskRewardRule(-bt.LossTolerance, bt.ProfitMargin, true),
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

func (bt backtest) name() string {
	return bt.r.GetName() + "-" + bt.bt.Signal.Name + "-" + time.Now().Format(simpleLayout)
}

func (bt backtest) summary(dir string) map[string]float64 {
	floatingPoint := uint(4)
	sm := make(map[string]float64, 6)
	sm["Profit"] = roundFloat(ta.TotalProfitAnalysis{}.Analyze(bt.rcs), floatingPoint)
	sm["PctGain"] = roundFloat(ta.PercentGainAnalysis{}.Analyze(bt.rcs), floatingPoint)
	sm["TotalTrades"] = roundFloat(float64(len(bt.rcs.Trades)), floatingPoint)
	sm["ProfitableTrades"] = roundFloat(ta.ProfitableTradesAnalysis{}.Analyze(bt.rcs), floatingPoint)
	sm["AverageProfit"] = roundFloat(ta.AverageProfitAnalysis{}.Analyze(bt.rcs), floatingPoint)
	if series, ok := bt.r.GetLines(bt.r.SmallestFrame()); ok {
		sm["Buy&Hold"] = roundFloat(ta.BuyAndHoldAnalysis{
			TimeSeries:    series.Candles,
			StartingMoney: bt.balance.Float(),
		}.Analyze(bt.rcs), floatingPoint)
	}
	//buffer := bytes.NewBufferString("")
	//_ = ta.LogTradesAnalysis{Writer: buffer}.Analyze(bt.rcs)
	//file, err := util.ConcatPath(dir, bt.name())
	//if err != nil {
	//	fmt.Println("ERROR", err)
	//	return sm
	//}
	//if err := os.WriteFile(file+".txt", buffer.Bytes(), 0444); err != nil {
	//	fmt.Println("ERROR", err)
	//}
	return sm
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
