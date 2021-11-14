package builder

import (
	"errors"

	tax "follow.market/internal/pkg/techanex"
)

type Condition struct {
	This *Comparable `json:"this"`
	That *Comparable `json:"that"`
	Opt  *operator   `json:"opt"`
}

type Conditions []*Condition

func (c *Condition) validate() error {
	if err := c.This.validate(); err != nil {
		return err
	}
	if err := c.That.validate(); err != nil {
		return err
	}
	if c.Opt == nil {
		return errors.New("missing operator")
	}
	return nil
}

func (c *Condition) evaluate(s *tax.Series) bool {
	if s == nil {
		return false
	}
	thisD, ok := c.This.mapDecimal(s)
	if !ok {
		return ok
	}
	thatD, ok := c.That.mapDecimal(s)
	if !ok {
		return ok
	}
	switch operator(*c.Opt) {
	case Less:
		return thisD.LT(thatD)
	case More:
		return thisD.GT(thatD)
	case LessEqual:
		return thisD.LTE(thatD)
	case MoreEqual:
		return thisD.GTE(thatD)
	case Equal:
		return thisD.EQ(thatD)
	default:
		return false
	}
}
