package strategy

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/util"
	"github.com/sdcoffey/big"
)

type Signal struct {
	// the signal basic information
	Name       string `json:"name"`
	OwnerID    *int64 `json:"owner_id"`
	NotifyType string `json:"notify_type"`
	TrackType  string `json:"track_type"`
	SignalType string `json:"signal_type"`

	// The conditions of the signal
	Groups     Groups        `json:"groups"`
	TimePeriod time.Duration `json:"primary_period"`

	// The trading information for the signal
	Trade struct {
		Price         *Comparable `json:"price"`
		MaxWaitToFill *int64      `json:"max_wait_to_fill"` // in second
	} `json:"trade"`
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
	if len(signal.Groups) == 0 {
		return nil, errors.New("not a valid signal")
	}
	for _, g := range signal.Groups {
		if err := g.validate(); err != nil {
			return nil, err
		}
	}
	periods := signal.GetPeriods()
	signal.TimePeriod = periods[0]
	return &signal, err
}

// Evaluate evaluates signal against the current status of the runner.
func (s *Signal) Evaluate(r *runner.Runner, t *tax.Trade) bool {
	if r == nil && t == nil {
		return false
	}
	for _, g := range s.Groups {
		if !g.evaluate(r, t) {
			return false
		}
	}
	return true
}

// copy does a deep copy on the signal.
func (s *Signal) copy() *Signal {
	var ns Signal
	ns.Name = s.Name
	ns.Groups = s.Groups.copy()
	ns.OwnerID = s.OwnerID
	ns.SignalType = s.SignalType
	ns.TrackType = s.TrackType
	ns.NotifyType = s.NotifyType
	ns.TimePeriod = s.TimePeriod
	ns.Trade.Price = s.Trade.Price.copy()
	ns.Trade.MaxWaitToFill = s.Trade.MaxWaitToFill
	return &ns
}

// Copy operates on a list of signal.
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
	for _, cg := range s.Groups {
		for _, g := range cg.Groups {
			for _, c := range g.Conditions {
				if c.Msg != nil {
					out = append(out, *c.Msg)
					thisFrame = (time.Duration(c.This.TimePeriod) * time.Second).String()
					thatFrame = (time.Duration(c.That.TimePeriod) * time.Second).String()
				}
			}
		}
	}
	out = append([]string{s.Name + ": " + thisFrame + ": " + thatFrame}, out...)
	return strings.Join(out, "\n")
}

// IsOnetime returns true if the signal is valid for only one time check.
func (s Signal) IsOnetime() bool {
	return strings.ToLower(s.TrackType) == strings.ToLower(OnetimeTrack)
}

// OpenTradingSide returns generic string for trading, either BUY or SELL.
func (s Signal) OpenTradingSide() string {
	if s.IsBullish() {
		return "BUY"
	}
	return "SELL"
}

// CloseTradingSide returns generic string for trading, either BUY or SELL.
func (s Signal) CloseTradingSide() string {
	if s.IsBullish() {
		return "SELL"
	}
	return "BUY"
}

// TradeExecutionPrice returns a price level for trade, ideally after the signal is successfully evaluated.
func (s Signal) TradeExecutionPrice(r *runner.Runner) (big.Decimal, bool) {
	if s.Trade.Price == nil {
		return big.ZERO, false
	}
	if err := s.Trade.Price.validate(); err != nil {
		return big.ZERO, false
	}
	if s.Trade.Price.Fundamental != nil && s.Trade.Price.Candle == nil && s.Trade.Price.Indicator == nil {
		return big.ZERO, false
	}
	_, price, ok := s.Trade.Price.mapDecimal(r, nil)
	return price, ok
}

// MaxWaitToFill returns the duration in second the trader should wait for an open order to be filled
// once a signal is triggered.
func (s Signal) GetMaxWaitToFill() (time.Duration, bool) {
	if s.Trade.MaxWaitToFill == nil {
		return time.Second, false
	}
	return time.Duration(*s.Trade.MaxWaitToFill) * time.Second, true
}

// IsBullish return true if the signal is bullish, false otherwise.
func (s Signal) IsBullish() bool {
	return strings.ToLower(s.SignalType) == strings.ToLower(BullishSignal)
}

// IsBearish return true if the signal is bearish, false otherwise.
func (s Signal) IsBearish() bool {
	return strings.ToLower(s.SignalType) == strings.ToLower(BearishSignal)
}

// Side returns BUY or SELL side of the signal depending on the given postion. This is only for tester to know whether to in or out a postion.
func (s Signal) BacktestSide(side string) string {
	if strings.ToUpper(side) != "BUY" || strings.ToUpper(side) != "SELL" {
		return side
	}
	isBuy := strings.ToUpper(side) == "BUY"
	if s.IsBullish() && isBuy {
		return "BUY"
	} else if s.IsBullish() && !isBuy {
		return "SELL"
	} else if s.IsBearish() && isBuy {
		return "SELL"
	} else if s.IsBearish() && !isBuy {
		return "BUY"
	} else {
		return side
		//panic("unknown signal type")
	}
}

func (s Signal) GetPeriods() []time.Duration {
	var periods []time.Duration
	for _, gs := range s.Groups {
		for _, g := range gs.Groups {
			if g == nil {
				continue
			}
			for _, c := range g.Conditions {
				if !util.DurationSliceContains(periods, time.Duration(c.This.TimePeriod)*time.Second) {
					periods = append(periods, time.Duration(c.This.TimePeriod)*time.Second)
				}
				if !util.DurationSliceContains(periods, time.Duration(c.That.TimePeriod)*time.Second) {
					periods = append(periods, time.Duration(c.That.TimePeriod)*time.Second)
				}
			}
		}
	}
	sort.Slice(periods, func(i, j int) bool {
		return periods[i] < periods[j]
	})
	return periods
}

// encodeNotify returns float64 ranging from -1 to 1 depends on signal notification option.
// the valid values ranges from 0 to 1,
// -1 means given data is wrong and won't be accepted.
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
