package strategy

import (
	"errors"
	"strconv"
	"time"

	ta "github.com/itsphat/techan"

	"github.com/sdcoffey/big"

	"follow.market/internal/pkg/runner"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/util"
)

type Comparable struct {
	Ticker     string `json:"ticker"`
	TimePeriod int    `json:"time_period"` // mustt be in second
	TimeFrame  int    `json:"time_frame"`
	Candle     *struct {
		Level     CandleLevel `json:"level"`
		Effective float64     `json:"effective"`
	} `json:"candle"`
	Indicator *struct {
		Name      tax.IndicatorName `json:"name"`
		Config    map[string]int    `json:"config"`
		Effective float64           `json:"effective"`
	} `json:"indicator"`
}

func (c *Comparable) validate() error {
	if c.Candle == nil && c.Indicator == nil {
		return errors.New("missing comparable values")
	}
	if c.Candle != nil && !util.StringSliceContains(candleLevels, string(c.Candle.Level)) {
		return errors.New("invalid candle level")
	}
	if c.Indicator != nil && (!util.StringSliceContains(tax.AvailableIndicators(), string(c.Indicator.Name)) || len(c.Indicator.Config) == 0) {
		return errors.New("invalid indicator name or config")
	}
	return nil
}

func (c *Comparable) convertTimePeriod() time.Duration {
	return time.Duration(c.TimePeriod) * time.Second
}

func (c *Comparable) mapDecimal(r *runner.Runner) (big.Decimal, bool) {
	line, ok := r.GetLines(c.convertTimePeriod())
	if !ok {
		return big.ZERO, ok
	}
	if c.Candle != nil {
		return c.mapCandle(line.CandleByIndex(len(line.Candles.Candles) - c.TimeFrame))
	}
	if c.Indicator != nil {
		return c.mapIndicator(line.IndicatorByIndex(len(line.Indicators.Indicators) - c.TimeFrame))
	}
	return big.ZERO, false
}

func (c *Comparable) mapCandle(cd *ta.Candle) (big.Decimal, bool) {
	if cd == nil {
		return big.ZERO, false
	}
	switch c.Candle.Level {
	case CandleOpen:
		return cd.OpenPrice, true
	case CandleClose:
		return cd.ClosePrice, true
	case CandleHigh:
		return cd.MaxPrice, true
	case CandleLow:
		return cd.MinPrice, true
	case CandleVolume:
		return cd.Volume, true
	case CandleTrade:
		return big.NewFromInt(int(cd.TradeCount)), true
	default:
		return big.ZERO, false
	}
}

func (c *Comparable) mapIndicator(id *tax.Indicator) (big.Decimal, bool) {
	if id == nil {
		return big.ZERO, false
	}
	window, ok := c.Indicator.Config["window"]
	if !ok {
		return big.ZERO, false
	}
	if v, ok := id.IndiMap[c.Indicator.Name.ToString()+"-"+strconv.Itoa(window)]; ok {
		return v, ok
	}
	return big.ZERO, false
}
