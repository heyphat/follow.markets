package techanex

import (
	"fmt"
	"time"

	ta "github.com/itsphat/techan"
)

type Series struct {
	Candles    *ta.TimeSeries
	Indicators *IndicatorSeries
}

func NewSeries(configs IndicatorConfigs) *Series {
	return &Series{
		Candles:    ta.NewTimeSeries(),
		Indicators: NewIndicatorSeries(configs),
	}
}

// SyncCandel is a combination of AddCandle and UpdateCandle where it aggregates a given
// candle to the series, the time period need to be given in order to perform the operation
func (s *Series) SyncCandle(candle *ta.Candle, d *time.Duration) bool {
	if candle == nil {
		panic(fmt.Errorf("error syncing candle: cannle cannot be nil"))
	}
	indicator := NewIndicator(syncPeriod(candle.Period, d), s.Indicators.Configs)
	if s.Candles.LastCandle() == nil || candle.Period.Since(s.Candles.LastCandle().Period) >= 0 {
		if !s.Candles.AddCandle(NewCandleFromCandle(candle, d)) {
			return false
		}
		indicator.Calculate(s.Indicators.Configs, s.Candles, len(s.Candles.Candles)-1)
		if !s.Indicators.addIndicator(indicator) {
			return false
		}
		return true
	}
	s.Candles.LastCandle().UpdateCandle(candle)
	s.Indicators.LastIndicator().Calculate(s.Indicators.Configs, s.Candles, len(s.Candles.Candles)-1)
	return true
}

// SyncCandles is a handy function to perform updating a series of candles to
// an exiting series, expecially when the frame of new time series is shorter
// than the original tim series
func (s *Series) SyncCandles(candles *ta.TimeSeries, d *time.Duration) bool {
	if candles == nil {
		return false
	}
	if len(candles.Candles) == 0 {
		return true
	}
	for _, c := range candles.Candles {
		if s.Candles.LastCandle() == nil || c.Period.Since(s.Candles.LastCandle().Period) >= 0 {
			if !s.Candles.AddCandle(NewCandleFromCandle(c, d)) {
				return false
			}
			continue
		}
		s.Candles.LastCandle().UpdateCandle(c)
	}
	return s.Indicators.newIndicatorsFromCandleSeries(s.Candles)
}

// AddCandle append the given candle to the series.Candles. It also create a new corresponding indicator
// and append it to the series.Indicators.
func (s *Series) AddCandle(candle *ta.Candle) bool {
	if candle == nil {
		panic(fmt.Errorf("error adding candle: cannle cannot be nil"))
	}
	if !s.Candles.AddCandle(candle) {
		return false
	}
	indicator := NewIndicator(candle.Period, s.Indicators.Configs)
	indicator.Calculate(s.Indicators.Configs, s.Candles, len(s.Candles.Candles)-1)
	if ok := s.Indicators.addIndicator(indicator); !ok {
		return ok
	}
	return true
}

// UpdateCandle aggregaates the given candle to the last candle on the Series.Candle. It also updates
// the indicator on the last candle. UpdateCandle is called when the period of the given candle is
// the sub-period of the last candle. UpdateCandle doesn't check if the sub-period condition is satisfied,
// it only perform the task.
func (s *Series) UpdateCandle(candle *ta.Candle) bool {
	if candle == nil {
		panic(fmt.Errorf("error aggregating Candle: cannle cannot be nil"))
	}
	if s.Candles.LastCandle() == nil {
		return false
	}
	s.Candles.LastCandle().UpdateCandle(candle)
	indicator := NewIndicator(s.Candles.LastCandle().Period, s.Indicators.Configs)
	indicator.Calculate(s.Indicators.Configs, s.Candles, len(s.Candles.Candles)-1)
	s.Indicators.Indicators[len(s.Indicators.Indicators)-1] = indicator
	return true
}

func (ts *Series) CandleByIndex(index int) *ta.Candle {
	if len(ts.Candles.Candles) == 0 || index < 0 {
		return nil
	}
	if len(ts.Candles.Candles) > index {
		return ts.Candles.Candles[index]
	}
	return nil
}

func (ts *Series) IndicatorByIndex(index int) *Indicator {
	if len(ts.Indicators.Indicators) == 0 || index < 0 {
		return nil
	}
	if len(ts.Indicators.Indicators) > index {
		return ts.Indicators.Indicators[index]
	}
	return nil
}

func (ts *Series) Shrink(size int) {
	if len(ts.Candles.Candles) != len(ts.Indicators.Indicators) {
		return
	}
	currentSize := len(ts.Candles.Candles)
	if currentSize+100 <= size {
		return
	}
	_, ts.Candles.Candles = ts.Candles.Candles[:currentSize-size-1], ts.Candles.Candles[currentSize-size-1:currentSize-1]
	_, ts.Indicators.Indicators = ts.Indicators.Indicators[:currentSize-size-1], ts.Indicators.Indicators[currentSize-size-1:currentSize-1]
}
