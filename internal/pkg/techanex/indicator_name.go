package techanex

import (
	"strconv"

	ta "github.com/itsphat/techan"
)

func AvailableIndicators() []string {
	return []string{
		MA.ToString(),
		VMA.ToString(),
		EMA.ToString(),
		BBU.ToString(),
		BBL.ToString(),
		ATR.ToString(),
	}
}

type IndicatorName string

const (
	MA  IndicatorName = "MovingAverge"
	VMA IndicatorName = "VolumeMovingAverage"
	EMA IndicatorName = "ExponentialMovingAverage"
	BBU IndicatorName = "BollingerUpperBand"
	BBL IndicatorName = "BollingerLowerBand"
	ATR IndicatorName = "AverageTrueRage"
)

func (n IndicatorName) getIndicator(ts *ta.TimeSeries, param interface{}) ta.Indicator {
	switch n {
	case EMA:
		return ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), param.(int))
	case MA:
		return ta.NewSimpleMovingAverage(ta.NewClosePriceIndicator(ts), param.(int))
	case VMA:
		return ta.NewSimpleMovingAverage(ta.NewVolumeIndicator(ts), param.(int))
	case BBU:
		return ta.NewBollingerUpperBandIndicator(ta.NewClosePriceIndicator(ts), param.(int), 2)
	case BBL:
		return ta.NewBollingerLowerBandIndicator(ta.NewClosePriceIndicator(ts), param.(int), 2)
	case ATR:
		return ta.NewAverageTrueRangeIndicator(ts, param.(int))
	default:
		return ta.NewClosePriceIndicator(ts)
	}
}

func (n IndicatorName) ToKey(i int) string {
	return string(n) + "-" + strconv.Itoa(i)
}

func (n IndicatorName) ToString() string {
	return string(n)
}
