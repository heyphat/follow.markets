package builder

import (
	"encoding/json"

	tax "follow.market/internal/pkg/techanex"
)

type Strategy struct {
	Name       string     `json:"name"`
	Conditions Conditions `json:"conditions"`
	Series     *tax.Series
}

func NewStrategy(bytes []byte) (*Strategy, error) {
	stg := Strategy{}
	err := json.Unmarshal(bytes, &stg)
	return &stg, err
}

func (s *Strategy) SetSeries(series *tax.Series) *Strategy {
	s.Series = series
	return s
}

func (s *Strategy) Evaluate() bool {
	if s.Series == nil {
		return false
	}
	for _, c := range s.Conditions {
		if !c.evaluate(s.Series) {
			return false
		}
	}
	return true
}
