package strategy

import (
	"encoding/json"
	"strings"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
)

type Strategy struct {
	Name            string          `json:"name"`
	Conditions      Conditions      `json:"conditions"`
	ConditionGroups ConditionGroups `json:"condition_groups"`
}

type Strategies []*Strategy

func NewStrategyFromBytes(bytes []byte) (*Strategy, error) {
	stg := Strategy{}
	err := json.Unmarshal(bytes, &stg)
	return &stg, err
}

func (s *Strategy) Evaluate(r *runner.Runner, t *tax.Trade) bool {
	if r == nil {
		return false
	}
	for _, c := range s.Conditions {
		if !c.evaluate(r, t) {
			return false
		}
	}
	for _, g := range s.ConditionGroups {
		if !g.evaluate(r, t) {
			return false
		}
	}
	return true
}

// Description returns a text description of all valid(true) conditions.
func (s Strategy) Description() string {
	var out []string
	for _, c := range s.Conditions {
		if c.Msg != nil {
			out = append(out, *c.Msg)
		}
	}
	for _, g := range s.ConditionGroups {
		for _, c := range g.Conditions {
			if c.Msg != nil {
				out = append(out, *c.Msg)
			}
		}
	}
	return strings.Join(out, "\n")
}

// IsOnTrade returns if a strategy is valid in term of trade evaluation support.
// A valid trade strategy is the one which has conditions only on `s.Trade` or condition
// groups only on `s.Trade`. Currently the system doesn't support a strategy which
// is a combination of `Candle` and `Trade` or `Indicator` and `Trade`.
func (s Strategy) IsOnTrade() bool {
	for _, c := range s.Conditions {
		if err := c.validate(); err != nil {
			return false
		}
		if c.This.Trade != nil || c.That.Trade != nil {
			return false
		}
	}
	for _, g := range s.ConditionGroups {
		for _, c := range g.Conditions {
			if err := c.validate(); err != nil {
				return false
			}
			if c.This.Trade != nil || c.That.Trade != nil {
				return false
			}
		}
	}
	return true
}
