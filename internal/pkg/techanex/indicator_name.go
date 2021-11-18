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
	BBD IndicatorName = "BollingerLowerBand"
	ATR IndicatorName = "AverageTrueRage"
)

func (n IndicatorName) getIndicator(indicator ta.Indicator, window int) ta.Indicator {
	switch n {
	case EMA:
		return ta.NewEMAIndicator(indicator, window)
	case MA:
		return ta.NewSimpleMovingAverage(indicator, window)
	case BBU:
		return ta.NewBollingerUpperBandIndicator(indicator, window, 2)
	case BBD:
		return ta.NewBollingerLowerBandIndicator(indicator, window, 2)
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
		BBD.ToString(),
		ATR.ToString()}
}
