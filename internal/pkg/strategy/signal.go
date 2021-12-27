package strategy

import (
	"encoding/json"
	"strings"
	"time"

	"follow.market/pkg/util"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
	ta "github.com/itsphat/techan"
)

type Signal struct {
	Name            string          `json:"name"`
	Conditions      Conditions      `json:"conditions"`
	ConditionGroups ConditionGroups `json:"condition_groups"`
	TimePeriod      time.Duration   `json:"primary_period"`

	NotifyType string `json:"notify_type"`
	TrackType  string `json:"track_type"`
	SignalType string `json:"signal_type"`
}

type Signals []*Signal

// A new signal entity will be created mostly from a post request body.
// This method will convert bytes data from a json to a singal and
// validate all the conditions.
func NewSignalFromBytes(bytes []byte) (*Signal, error) {
	signal := Signal{}
	err := json.Unmarshal(bytes, &signal)
	if err != nil {
		return nil, err
	}
	for i, c := range signal.Conditions {
		if err := c.validate(); err != nil {
			return nil, err
		}
		// TODO: need to find a better structure to parse the primary duration
		if i == 0 {
			signal.TimePeriod = time.Second * time.Duration(c.This.TimePeriod)
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

func (s *Signal) copy() *Signal {
	var ns Signal
	ns.Name = s.Name
	ns.Conditions = s.Conditions.copy()
	ns.ConditionGroups = s.ConditionGroups.copy()
	ns.SignalType = s.SignalType
	ns.TrackType = s.TrackType
	ns.NotifyType = s.NotifyType
	ns.TimePeriod = s.TimePeriod
	return &ns
}

func (ss Signals) Copy() Signals {
	var out Signals
	for _, s := range ss {
		out = append(out, s.copy())
	}
	return out
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

// IsOnetime returns true if the signal is valid for only one time check.
func (s Signal) IsOnetime() bool {
	return strings.ToLower(s.TrackType) == strings.ToLower(OnetimeTrack)
}

// IsBullish return true if the signal is bullish, false otherwise.
func (s Signal) IsBullish() bool {
	return strings.ToLower(s.SignalType) == strings.ToLower(BullishSignal)
}

// IsBearish return true if the signal is bearish, false otherwise.
func (s Signal) IsBearish() bool {
	return strings.ToLower(s.SignalType) == strings.ToLower(BearishSignal)
}

// Side returns BUY or SELL side of the signal depending on the given postion.
func (s Signal) Side(side ta.OrderSide) ta.OrderSide {
	if s.IsBullish() && side == ta.BUY {
		return ta.BUY
	} else if s.IsBullish() && side == ta.SELL {
		return ta.SELL
	} else if s.IsBearish() && side == ta.BUY {
		return ta.SELL
	} else if s.IsBearish() && side == ta.SELL {
		return ta.BUY
	} else {
		panic("unknown signal type")
	}
}

func (s Signal) GetPeriods() []time.Duration {
	var periods []time.Duration
	for _, c := range s.Conditions {
		if !util.DurationSliceContains(periods, time.Duration(c.This.TimePeriod)*time.Second) {
			periods = append(periods, time.Duration(c.This.TimePeriod)*time.Second)
		}
		if !util.DurationSliceContains(periods, time.Duration(c.That.TimePeriod)*time.Second) {
			periods = append(periods, time.Duration(c.This.TimePeriod)*time.Second)
		}
	}
	for _, g := range s.ConditionGroups {
		if g == nil {
			continue
		}
		for _, c := range g.Conditions {
			if !util.DurationSliceContains(periods, time.Duration(c.This.TimePeriod)*time.Second) {
				periods = append(periods, time.Duration(c.This.TimePeriod)*time.Second)
			}
			if !util.DurationSliceContains(periods, time.Duration(c.That.TimePeriod)*time.Second) {
				periods = append(periods, time.Duration(c.This.TimePeriod)*time.Second)
			}
		}
	}
	return periods
}

// encodeNotify returns float64 ranging from -1 to 1 depends on signal notification options.
// the meaning values only from 0 to 1, -1 means given data is wrong and won't be accepted.
// 0 means only send once.
// 1 means send all the time the signal is valid.
// 0 -> 1 means some where between the given time period.
func (s Signal) encodeNotify() float64 {
	switch s.NotifyType {
	case AllNotify:
		return 1.0
	case FstNotify:
		return 0.0
	case MidNotify:
		return 0.5
	default:
		return -1.0
	}
}

// ShouldSend return true if the triggered time satisfied the sending policy defined
// on the signal. The ShoudSend method should be called only when the signal is continuous
// and all signal's conditions pass validation and evaluation. If it is onetime signal,
// the method returns false.
func (s Signal) ShouldSend(lastSent time.Time) bool {
	// TimePeriod == 0 means the duration is unknown, hence return false.
	if s.IsOnetime() || s.TimePeriod == 0 || lastSent.Unix() == 0 {
		return false
	}
	nw := time.Now().Add(-time.Minute)
	beginFrame := nw.Truncate(s.TimePeriod)
	if s.encodeNotify() == 1.0 {
		// 1.0 means always send the notification
		return true
	} else if s.encodeNotify() != 1.0 && !beginFrame.Equal(lastSent.Truncate(s.TimePeriod)) {
		// sends when timeframe hasn't had any notis and is the first and satisfied mid policy.
		return nw.Sub(beginFrame) >= time.Duration(s.TimePeriod.Seconds()*s.encodeNotify())*time.Second
	}
	return false
}
