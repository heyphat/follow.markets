package strategy

import (
	"encoding/json"

	"follow.market/internal/pkg/runner"
)

type Strategy struct {
	Name            string          `json:"name"`
	Conditions      Conditions      `json:"conditions"`
	ConditionGroups ConditionGroups `json:"condition_groups"`
}

type Strategies []*Strategy

func NewStrategy(bytes []byte) (*Strategy, error) {
	stg := Strategy{}
	err := json.Unmarshal(bytes, &stg)
	return &stg, err
}

func (s *Strategy) Evaluate(r *runner.Runner) bool {
	if r == nil {
		return false
	}
	for _, c := range s.Conditions {
		if !c.evaluate(r) {
			return false
		}
	}
	for _, g := range s.ConditionGroups {
		if !g.evaluate(r) {
			return false
		}
	}
	return true
}
