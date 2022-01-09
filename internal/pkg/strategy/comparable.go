package strategy

import (
	"errors"
	"strconv"
	"time"

	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/util"
)

type ComparableObject struct {
	Name       string             `json:"name"`
	Config     map[string]float64 `json:"config"`
	Multiplier *float64           `json:"multiplier"`
}

func (c *ComparableObject) copy() *ComparableObject {
	if c == nil {
		return nil
	}
	var nc ComparableObject
	nc.Name = c.Name
	nc.Multiplier = c.Multiplier
	nc.Config = c.Config
	return &nc
}

func (c *ComparableObject) parseMultiplier() big.Decimal {
	if c == nil || c.Multiplier != nil {
		return big.NewDecimal(*c.Multiplier)
	}
	return big.ONE
}

type Comparable struct {
	TimePeriod  int               `json:"time_period"` // mustt be in second
	TimeFrame   int               `json:"time_frame"`
	Candle      *ComparableObject `json:"candle,omitempty"`
	Indicator   *ComparableObject `json:"indicator,omitempty"`
	Fundamental *ComparableObject `json:"fundamental,omitempty"`
}

func (c *Comparable) copy() *Comparable {
	if c == nil {
		return nil
	}
	var nc Comparable
	nc.TimePeriod = c.TimePeriod
	nc.TimeFrame = c.TimeFrame
	nc.Candle = c.Candle.copy()
	nc.Indicator = c.Indicator.copy()
	nc.Fundamental = c.Fundamental.copy()
	return &nc
}

func (c *Comparable) convertTimePeriod() time.Duration {
	return time.Duration(c.TimePeriod) * time.Second
}

func (c *Comparable) validate() error {
	if c.Candle == nil && c.Indicator == nil && c.Fundamental == nil { // c.Trade == nil
		return errors.New("missing comparable values")
	}
	if !util.Int64SliceContains(AcceptablePeriods, int64(c.TimePeriod)) {
		return errors.New("unknown time period")
	}
	if c.TimeFrame < 0 {
		return errors.New("invalid time frame")
	}
	if c.Candle != nil && !util.StringSliceContains(candleLevels, string(c.Candle.Name)) {
		return errors.New("invalid candle level")
	}
	if c.Indicator != nil && (!util.StringSliceContains(tax.AvailableIndicators(), string(c.Indicator.Name)) || len(c.Indicator.Config) == 0) {
		return errors.New("invalid indicator name or config")
	}
	if c.Fundamental != nil && (!util.StringSliceContains(fundamentals, string(c.Fundamental.Name))) {
		return errors.New("invalid fundamental name")
	}
	return nil
}

func (c *Comparable) mapDecimal(r *runner.Runner, t *tax.Trade) (string, big.Decimal, bool) {
	minFloatingPoints := 3
	//if c.Trade != nil {
	//	val, ok := c.mapTrade(t)
	//	val = val.Mul(c.Trade.parseMultiplier())
	//	mess := "Trade: " + c.Trade.Name + "@" + val.FormattedString(minFloatingPoints)
	//	return mess, val, ok
	//}
	if r == nil {
		return "", big.ZERO, false
	}
	line, ok := r.GetLines(c.convertTimePeriod())
	if !ok || line == nil {
		return "", big.ZERO, ok
	}
	if line.Candles.LastCandle() == nil {
		return "", big.ZERO, false
	}
	currentPeriod := line.Candles.LastCandle().Period
	if c.Candle != nil {
		val, ok := c.mapCandle(line.CandleByIndex(len(line.Candles.Candles)-1-c.TimeFrame), currentPeriod)
		mess := "Candle: " + c.Candle.Name + "@" + val.FormattedString(minFloatingPoints)
		return mess, val.Mul(c.Candle.parseMultiplier()), ok
	}
	if c.Indicator != nil {
		val, ok := c.mapIndicator(line.IndicatorByIndex(len(line.Indicators.Indicators)-1-c.TimeFrame), currentPeriod)
		mess := "Indicator: " + c.Indicator.Name + "@" + val.FormattedString(minFloatingPoints)
		return mess, val.Mul(c.Indicator.parseMultiplier()), ok
	}
	if c.Fundamental != nil && r != nil {
		val, ok := c.mapFundamental(r)
		mess := "Fundamental: " + c.Fundamental.Name + "@" + val.FormattedString(minFloatingPoints)
		return mess, val.Mul(c.Fundamental.parseMultiplier()), ok
	}
	return "", big.ZERO, false
}

func (c *Comparable) validatePeriod(referencePeriod, currentPeriod ta.TimePeriod) bool {
	return referencePeriod.Advance(c.TimeFrame).Start.Equal(currentPeriod.Start)
}

func (c *Comparable) mapCandle(cd *ta.Candle, currentPeriod ta.TimePeriod) (big.Decimal, bool) {
	if cd == nil {
		return big.ZERO, false
	}
	if !c.validatePeriod(cd.Period, currentPeriod) {
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
	case CandleUSDVolume:
		return cd.Volume.Mul(cd.ClosePrice), true
	case CandleTrade:
		return big.NewFromInt(int(cd.TradeCount)), true
	case CandleLowHigh:
		return tax.LowHigh(cd.MinPrice, cd.MaxPrice), true
	case CandleOpenClose:
		return tax.OpenClose(cd.OpenPrice, cd.ClosePrice), true
	case CandleOpenHigh:
		return tax.OpenClose(cd.OpenPrice, cd.MaxPrice), true
	case CandleOpenLow:
		return tax.OpenLow(cd.OpenPrice, cd.MinPrice), true
	case CandleHighClose:
		return tax.HighClose(cd.MaxPrice, cd.ClosePrice), true
	case CandleLowClose:
		return tax.LowClose(cd.MinPrice, cd.ClosePrice), true
	case CandleMidLowHigh:
		return tax.MidPoint(cd.MinPrice, cd.MaxPrice), true
	case CandleMidOpenClose:
		return tax.MidPoint(cd.OpenPrice, cd.ClosePrice), true
	case CandleOpenTime:
		return big.NewFromInt(int(cd.Period.Start.Unix())), true
	case CandleCloseTime:
		return big.NewFromInt(int(cd.Period.End.Unix())), true
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

func (c *Comparable) mapIndicator(id *tax.Indicator, currentPeriod ta.TimePeriod) (big.Decimal, bool) {
	if id == nil {
		return big.ZERO, false
	}
	if !c.validatePeriod(id.Period, currentPeriod) {
		return big.ZERO, false
	}
	var indiName string
	if window, ok := c.Indicator.Config["window"]; !ok {
		indiName = c.Indicator.Name
	} else {
		indiName = c.Indicator.Name + "-" + strconv.Itoa(int(window))
	}
	if v, ok := id.IndiMap[indiName]; ok {
		return v, ok
	}
	return big.ZERO, false
}

func (c *Comparable) mapFundamental(r *runner.Runner) (big.Decimal, bool) {
	if c == nil || r == nil {
		return big.ZERO, false
	}
	switch Fundamental(c.Fundamental.Name) {
	case FundMarketCap:
		return r.GetCap(), true
	case FundMaxSupply:
		return r.GetMaxSupply(), true
	case FundTotalSupply:
		return r.GetTotalSupply(), true
	case FundCirculatingSupply:
		return r.GetFloat(), true
	default:
		return big.ZERO, false
	}
}

//func (c *Comparable) mapTrade(td *tax.Trade) (big.Decimal, bool) {
//	if td == nil {
//		return big.ZERO, false
//	}
//	switch TradeLevel(c.Trade.Name) {
//	case TradeVolume:
//		return td.Quantity, true
//	case TradePrice:
//		return td.Price, true
//	case TradeFixed:
//		value, ok := c.Trade.Config["level"]
//		if !ok {
//			return big.ZERO, false
//		}
//		return big.NewDecimal(value), true
//	case TradeUSDVolume:
//		return td.Quantity.Mul(td.Price), true
//	default:
//		return big.ZERO, false
//	}
//}
