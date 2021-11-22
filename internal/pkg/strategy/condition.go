package strategy

import (
	"errors"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
)

type Condition struct {
	This *Comparable `json:"this"`
	That *Comparable `json:"that"`
	Opt  *Operator   `json:"opt"`
	Msg  *string     `json:"message"`
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

func (c *Condition) evaluate(r *runner.Runner, t *tax.Trade) bool {
	if r == nil && t == nil {
		return false
	}
	thisM, thisD, ok := c.This.mapDecimal(r, t)
	if !ok {
		return ok
	}
	thatM, thatD, ok := c.That.mapDecimal(r, t)
	if !ok {
		return ok
	}
	mess := thisM + " " + c.Opt.toString() + " " + thatM
	valid := false
	switch Operator(*c.Opt) {
	case Less:
		valid = thisD.LT(thatD)
	case More:
		valid = thisD.GT(thatD)
	case LessEqual:
		valid = thisD.LTE(thatD)
	case MoreEqual:
		valid = thisD.GTE(thatD)
	case Equal:
		valid = thisD.GTE(thatD)
	}
	if valid {
		c.Msg = &mess
	}
	return valid
}

type ConditionGroup struct {
	Conditions Conditions `json:"conditions"`
	Opt        *Operator  `json:"opt"`
}

type ConditionGroups []*ConditionGroup

func (g *ConditionGroup) validate() error {
	if g.Opt == nil {
		return errors.New("missing group operator")
	}
	if *g.Opt != Or && *g.Opt != And {
		return errors.New("invalid group condition")
	}
	for _, c := range g.Conditions {
		if err := c.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (g *ConditionGroup) evaluate(r *runner.Runner, t *tax.Trade) bool {
	if r == nil {
		return false
	}
	switch *g.Opt {
	case And:
		for _, c := range g.Conditions {
			if !c.evaluate(r, t) {
				return false
			}
		}
	case Or:
		for _, c := range g.Conditions {
			if c.evaluate(r, t) {
				return true
			}
		}
	default:
		return false
	}
	return true
}
