package strategy

import (
	"time"

	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
)

type GenericRule struct {
	Signal Signal

	runner *runner.Runner
	trade  *tax.Trade
}

func NewRule(signal Signal) *GenericRule {
	return &GenericRule{Signal: signal}
}

func (gr *GenericRule) SetRunner(r *runner.Runner) *GenericRule {
	gr.runner = r
	return gr
}

func (gr *GenericRule) SetTrade(t *tax.Trade) *GenericRule {
	gr.trade = t
	return gr
}

func (gr GenericRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	if gr.runner == nil && gr.trade == nil {
		return false
	}
	return gr.Signal.Evaluate(gr.runner, gr.trade)
}

type StopLossRule struct {
	LossTolerance big.Decimal

	runner *runner.Runner
}

func (sr *StopLossRule) SetRunner(r *runner.Runner) *StopLossRule {
	sr.runner = r
	return sr
}

func NewStopLossRule(tol float64) *StopLossRule {
	return &StopLossRule{LossTolerance: big.NewDecimal(tol)}
}

func (sr StopLossRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	if sr.runner == nil {
		return false
	}
	if !record.CurrentPosition().IsOpen() {
		return false
	}
	line, ok := sr.runner.GetLines(time.Minute)
	if !ok || line == nil {
		return false
	}
	candle := line.Candles.LastCandle()
	if candle == nil {
		return false
	}
	openPrice := record.CurrentPosition().CostBasis()
	loss := candle.ClosePrice.Div(openPrice).Sub(big.ONE)
	return loss.LTE(sr.LossTolerance)
}

type TakeProfitRule struct {
	ProfitMargin big.Decimal

	runner *runner.Runner
}

func NewTakeProfitRule(mgr float64) *TakeProfitRule {
	return &TakeProfitRule{ProfitMargin: big.NewDecimal(mgr)}
}

func (pr *TakeProfitRule) SetRunner(r *runner.Runner) *TakeProfitRule {
	pr.runner = r
	return pr
}

func (pr TakeProfitRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	if pr.runner == nil {
		return false
	}
	if !record.CurrentPosition().IsOpen() {
		return false
	}
	line, ok := pr.runner.GetLines(time.Minute)
	if !ok || line == nil {
		return false
	}
	candle := line.Candles.LastCandle()
	if candle == nil {
		return false
	}
	openPrice := record.CurrentPosition().CostBasis()
	loss := candle.ClosePrice.Div(openPrice).Sub(big.ONE)
	return loss.GTE(pr.ProfitMargin)
}

func NewRiskRewardRule(lossTolerance, profitMargin float64, r *runner.Runner) ta.Rule {
	if r == nil {
		panic("runner must not be nil")
	}
	sl := NewStopLossRule(lossTolerance).SetRunner(r)
	tp := NewTakeProfitRule(profitMargin).SetRunner(r)
	return ta.Or(sl, tp)
}
