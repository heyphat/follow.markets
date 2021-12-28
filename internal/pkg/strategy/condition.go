package strategy

import (
	"errors"
	"strings"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
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

func (c *Condition) copy() *Condition {
	var nc Condition
	nc.This = c.This.copy()
	nc.That = c.That.copy()
	nc.Opt = c.Opt.copy()
	nc.Msg = nil
	return &nc
}

func (cs Conditions) copy() Conditions {
	var ncs Conditions
	for _, c := range cs {
		ncs = append(ncs, c.copy())
	}
	return ncs
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

func (g *ConditionGroup) copy() *ConditionGroup {
	var ng ConditionGroup
	ng.Conditions = g.Conditions.copy()
	ng.Opt = g.Opt.copy()
	return &ng
}

func (gs ConditionGroups) copy() ConditionGroups {
	var ngs ConditionGroups
	for _, g := range gs {
		ngs = append(ngs, g.copy())
	}
	return ngs
}

func (g *ConditionGroup) validate() error {
	if g.Opt == nil {
		return errors.New("missing group operator")
	}
	if strings.ToLower(string(*g.Opt)) != strings.ToLower(string(Or)) && strings.ToLower(string(*g.Opt)) != strings.ToLower(string(And)) {
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
