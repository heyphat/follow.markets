package builder

import (
	"encoding/json"

	tax "follow.market/internal/pkg/techanex"
)

type Strategy struct {
	Name       string     `json:"name"`
	Conditions Conditions `json:"conditions"`
}

func NewStrategy(bytes []byte) (*Strategy, error) {
	stg := Strategy{}
	err := json.Unmarshal(bytes, &stg)
	return &stg, err
}

func (s *Strategy) Evaluate(series *tax.Series) bool {
	if series == nil {
		return false
	}
	for _, c := range s.Conditions {
		if !c.evaluate(series) {
			return false
		}
	}
	return true
}
