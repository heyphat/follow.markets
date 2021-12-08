package strategy

import (
	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"

	"follow.market/internal/pkg/runner"
)

type GenericRule struct {
	Signal Signal

	runner *runner.Runner
	//trade  *tax.Trade
}

func NewRule(signal Signal) *GenericRule {
	return &GenericRule{Signal: signal}
}

func (gr *GenericRule) SetRunner(r *runner.Runner) *GenericRule {
	if gr == nil {
		return nil
	}
	gr.runner = r
	return gr
}

//func (gr *GenericRule) SetTrade(t *tax.Trade) *GenericRule {
//	gr.trade = t
//	return gr
//}

func (gr GenericRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	// the index is supposed to range from 0 to len(r.line.Candles)
	if gr.runner == nil { //&& gr.trade == nil {
		return false
	}
	if gr.Signal.IsOnTrade() {
		return false
	}
	//return gr.Signal.Evaluate(gr.runner, gr.trade)
	return gr.Signal.Evaluate(gr.runner, nil)
}

type StopLossRule struct {
	LossTolerance big.Decimal

	runner *runner.Runner
}

func (sr *StopLossRule) SetRunner(r *runner.Runner) *StopLossRule {
	if sr == nil {
		return nil
	}
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
	line, ok := sr.runner.GetLines(sr.runner.SmallestFrame())
	if !ok || line == nil {
		return false
	}
	candle := line.Candles.LastCandle()
	if candle == nil {
		return false
	}
	openPrice := record.CurrentPosition().EntranceOrder().Price
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
	if pr == nil {
		return nil
	}
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
	line, ok := pr.runner.GetLines(pr.runner.SmallestFrame())
	if !ok || line == nil {
		return false
	}
	candle := line.Candles.LastCandle()
	if candle == nil {
		return false
	}
	openPrice := record.CurrentPosition().EntranceOrder().Price
	profit := candle.ClosePrice.Div(openPrice).Sub(big.ONE)
	return profit.GTE(pr.ProfitMargin)
}

type RiskRewardRule struct {
	StopLoss   *StopLossRule
	TakeProfit *TakeProfitRule
}

func NewRiskRewardRule(lossTolerance, profitMargin float64) *RiskRewardRule {
	return &RiskRewardRule{StopLoss: NewStopLossRule(lossTolerance),
		TakeProfit: NewTakeProfitRule(profitMargin),
	}
}

func (rr *RiskRewardRule) SetRunner(r *runner.Runner) *RiskRewardRule {
	if rr == nil {
		return nil
	}
	rr.StopLoss.SetRunner(r)
	rr.TakeProfit.SetRunner(r)
	return rr
}

func (rr RiskRewardRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	//return rr.StopLoss.IsSatisfied(index, record) || rr.TakeProfit.IsSatisfied(index, record)
	return ta.Or(rr.StopLoss, rr.TakeProfit).IsSatisfied(index, record)
}
