package strategy

import (
	ta "github.com/heyphat/techan"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
)

type GenericRule struct {
	Signal Signal

	runner *runner.Runner
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

func (gr GenericRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	if gr.runner == nil {
		return false
	}
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
	if tol < 0 {
		return &StopLossRule{LossTolerance: big.NewDecimal(tol)}
	}
	return &StopLossRule{LossTolerance: big.NewDecimal(-tol)}
}

func (sr *StopLossRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
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
	ProfitMargin     big.Decimal
	PassedLimitNTime float64
	WithTrailing     bool

	runner *runner.Runner
}

func NewTakeProfitRule(mgr float64, tl bool) *TakeProfitRule {
	return &TakeProfitRule{ProfitMargin: big.NewDecimal(mgr), WithTrailing: tl}
}

func (pr *TakeProfitRule) SetRunner(r *runner.Runner) *TakeProfitRule {
	if pr == nil {
		return nil
	}
	pr.runner = r
	return pr
}

func (pr *TakeProfitRule) SetPassedLimit() {
	pr.PassedLimitNTime += 1.0
}

func (pr *TakeProfitRule) ResetPassedLimit() {
	pr.PassedLimitNTime = 0
}

// only works on bullish signal
func (pr *TakeProfitRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
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
	if pr.PassedLimitNTime > 0 && pr.WithTrailing {
		halfSize := pr.ProfitMargin.Div(big.NewDecimal(2.0))
		openPrice = openPrice.Mul(big.ONE.Add(halfSize.Mul(big.NewDecimal(pr.PassedLimitNTime))))
		if candle.ClosePrice.LTE(openPrice) {
			pr.ResetPassedLimit()
			return true
		}
	}
	profit := candle.ClosePrice.Div(openPrice).Sub(big.ONE)
	if !pr.WithTrailing {
		return profit.GTE(pr.ProfitMargin)
	}
	if profit.GTE(pr.ProfitMargin) && pr.WithTrailing {
		pr.SetPassedLimit()
	}
	return false
}

type RiskRewardRule struct {
	StopLoss     *StopLossRule
	TakeProfit   *TakeProfitRule
	WithTrailing bool
}

func NewRiskRewardRule(lossTolerance, profitMargin float64, withTrailing bool) *RiskRewardRule {
	return &RiskRewardRule{StopLoss: NewStopLossRule(lossTolerance),
		TakeProfit: NewTakeProfitRule(profitMargin, withTrailing),
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

func (rr *RiskRewardRule) IsSatisfied(index int, record *ta.TradingRecord) bool {
	return ta.Or(rr.StopLoss, rr.TakeProfit).IsSatisfied(index, record)
}
