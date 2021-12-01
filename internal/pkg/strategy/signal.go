package strategy

import (
	"encoding/json"
	"strings"
	"time"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
)

type Signal struct {
	Name            string          `json:"name"`
	Conditions      Conditions      `json:"conditions"`
	ConditionGroups ConditionGroups `json:"condition_groups"`
}

type Signals []*Signal

func NewSignalFromBytes(bytes []byte) (*Signal, error) {
	signal := Signal{}
	err := json.Unmarshal(bytes, &signal)
	if err != nil {
		return nil, err
	}
	for _, c := range signal.Conditions {
		if err := c.validate(); err != nil {
			return nil, err
		}
	}
	for _, g := range signal.ConditionGroups {
		if err := g.validate(); err != nil {
			return nil, err
		}
	}
	return &signal, err
}

func (s *Signal) Evaluate(r *runner.Runner, t *tax.Trade) bool {
	if r == nil && t == nil {
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
func (s Signal) Description() string {
	var out []string
	var thisFrame, thatFrame string
	for _, c := range s.Conditions {
		if c.Msg != nil {
			out = append(out, *c.Msg)
			thisFrame = (time.Duration(c.This.TimePeriod) * time.Second).String()
			thatFrame = (time.Duration(c.That.TimePeriod) * time.Second).String()
		}
	}
	for _, g := range s.ConditionGroups {
		for _, c := range g.Conditions {
			if c.Msg != nil {
				out = append(out, *c.Msg)
				thisFrame = (time.Duration(c.This.TimePeriod) * time.Second).String()
				thatFrame = (time.Duration(c.That.TimePeriod) * time.Second).String()
			}
		}
	}
	out = append([]string{s.Name + ": " + thisFrame + ": " + thatFrame}, out...)
	return strings.Join(out, "\n")
}

// IsOnTrade returns if a strategy is valid in term of trade evaluation support.
// A valid trade strategy is the one which has conditions only on `s.Trade` or condition
// groups only on `s.Trade`. Currently the system doesn't support a strategy which
// is a combination of `Candle` and `Trade` or `Indicator` and `Trade`.
func (s Signal) IsOnTrade() bool {
	for _, c := range s.Conditions {
		if err := c.validate(); err != nil {
			return false
		}
		if c.This.Trade == nil || c.That.Trade == nil {
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
