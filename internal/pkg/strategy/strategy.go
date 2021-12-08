package strategy

import (
	"follow.market/internal/pkg/runner"
	ta "github.com/itsphat/techan"
)

type Strategy struct {
	EntryRule      *GenericRule
	ExitRule       *GenericRule
	RiskRewardRule *RiskRewardRule
}

func (s *Strategy) SetRunner(r *runner.Runner) *Strategy {
	s.EntryRule.SetRunner(r)
	s.ExitRule.SetRunner(r)
	s.RiskRewardRule.SetRunner(r)
	return s
}

func (s Strategy) ShouldEnter(index int, record *ta.TradingRecord) bool {
	if s.EntryRule == nil {
		return false
	}
	if record.CurrentPosition().IsNew() {
		return s.EntryRule.IsSatisfied(index, record)
	}
	return false
}

func (s Strategy) ShouldExit(index int, record *ta.TradingRecord) bool {
	if s.ExitRule == nil && s.RiskRewardRule == nil {
		return false
	}
	if s.ExitRule != nil && record.CurrentPosition().IsOpen() {
		return s.ExitRule.IsSatisfied(index, record)
	}
	if s.RiskRewardRule != nil && record.CurrentPosition().IsOpen() {
		return s.RiskRewardRule.IsSatisfied(index, record)
	}
	return false
}
