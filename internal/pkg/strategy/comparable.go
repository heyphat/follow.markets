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
	Ticker     string            `json:"ticker"`
	TimePeriod int               `json:"time_period"` // mustt be in second
	TimeFrame  int               `json:"time_frame"`
	Trade      *ComparableObject `json:"trade",omitempty`
	Candle     *ComparableObject `json:"candle,omitempty"`
	Indicator  *ComparableObject `json:"indicator,omitempty"`
}

type ComparableObject struct {
	Name       string             `json:"name"`
	Config     map[string]float64 `json:"config"`
	Multiplier float64            `json:"multiplier"`
}

func (c *Comparable) validate() error {
	if c.Candle == nil && c.Indicator == nil {
		return errors.New("missing comparable values")
	}
	if c.Candle != nil && !util.StringSliceContains(candleLevels, string(c.Candle.Name)) {
		return errors.New("invalid candle level")
	}
	if c.Indicator != nil && (!util.StringSliceContains(tax.AvailableIndicators(), string(c.Indicator.Name)) || len(c.Indicator.Config) == 0) {
		return errors.New("invalid indicator name or config")
	}
	if c.Trade != nil && !(!util.StringSliceContains(tradeLevels, c.Trade.Name)) {
		return errors.New("invalid trade name")
	}
	return nil
}

func (c *Comparable) convertTimePeriod() time.Duration {
	return time.Duration(c.TimePeriod) * time.Second
}

func (c *Comparable) mapDecimal(r *runner.Runner, t *tax.Trade) (string, big.Decimal, bool) {
	line, ok := r.GetLines(c.convertTimePeriod())
	if !ok {
		return "", big.ZERO, ok
	}
	if c.Candle != nil {
		val, ok := c.mapCandle(line.CandleByIndex(len(line.Candles.Candles) - c.TimeFrame))
		mess := c.Candle.Name + " candle level @" + val.FormattedString(2)
		return mess, val, ok
	}
	if c.Indicator != nil {
		val, ok := c.mapIndicator(line.IndicatorByIndex(len(line.Indicators.Indicators) - c.TimeFrame))
		mess := c.Indicator.Name + " indicator level @" + val.FormattedString(2)
		return mess, val, ok
	}
	if c.Trade != nil {
		val, ok := c.mapTrade(t)
		mess := c.Trade.Name + " trade level @" + val.FormattedString(2)
		return mess, val, ok
	}
	return "", big.ZERO, false
}

func (c *Comparable) mapCandle(cd *ta.Candle) (big.Decimal, bool) {
	if cd == nil {
		return big.ZERO, false
	}
	switch CandleLevel(c.Candle.Name) {
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
	case CandleFixed:
		value, ok := c.Candle.Config["level"]
		if !ok {
			return big.ZERO, false
		}
		return big.NewDecimal(value), true
	default:
		return big.ZERO, false
	}
}

func (c *Comparable) mapTrade(td *tax.Trade) (big.Decimal, bool) {
	if td == nil {
		return big.ZERO, false
	}
	switch TradeLevel(c.Trade.Name) {
	case TradeAmount:
		return td.Quantity, true
	case TradePrice:
		return td.Price, true
	case TradeFixed:
		value, ok := c.Trade.Config["amount"]
		if !ok {
			return big.ZERO, false
		}
		return big.NewDecimal(value), true
	default:
		return big.ZERO, false
	}
}

func (c *Comparable) mapIndicator(id *tax.Indicator) (big.Decimal, bool) {
	if id == nil {
		return big.ZERO, false
	}
	if tax.IndicatorName(c.Indicator.Name) == tax.LEVL {
		value, ok := c.Indicator.Config["level"]
		if !ok {
			return big.ZERO, false
		}
		return big.NewDecimal(value), true
	}
	window, ok := c.Indicator.Config["window"]
	if !ok {
		return big.ZERO, false
	}
	if v, ok := id.IndiMap[c.Indicator.Name+"-"+strconv.Itoa(int(window))]; ok {
		return v, ok
	}
	return big.ZERO, false
}
