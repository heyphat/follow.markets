package techanex

import (
	"strconv"

	ta "github.com/itsphat/techan"
)

type IndicatorName string

const (
	MA  IndicatorName = "MovingAverge"
	EMA IndicatorName = "ExponentialMovingAverage"
	BBU IndicatorName = "BollingerUpperBand"
	BBL IndicatorName = "BollingerLowerBand"
	ATR IndicatorName = "AverageTrueRage"
)

func (n IndicatorName) getIndicator(indicator ta.Indicator, param interface{}) ta.Indicator {
	switch n {
	case EMA:
		return ta.NewEMAIndicator(indicator, param.(int))
	case MA:
		return ta.NewSimpleMovingAverage(indicator, param.(int))
	case BBU:
		return ta.NewBollingerUpperBandIndicator(indicator, param.(int), 2)
	case BBL:
		return ta.NewBollingerLowerBandIndicator(indicator, param.(int), 2)
	default:
		return indicator
	}
}

func (n IndicatorName) ToKey(i int) string {
	return string(n) + "-" + strconv.Itoa(i)
}

func (n IndicatorName) ToString() string {
	return string(n)
}

func AvailableIndicators() []string {
	return []string{
		MA.ToString(),
		EMA.ToString(),
		BBU.ToString(),
		BBL.ToString(),
		ATR.ToString(),
	}
}
