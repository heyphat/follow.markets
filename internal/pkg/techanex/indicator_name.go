package techanex

import (
	"sort"
	"strconv"

	ta "github.com/heyphat/techan"
)

func AvailableIndicators() []string {
	return []string{
		MA.ToString(),
		VMA.ToString(),
		LHMA.ToString(),
		OCAMA.ToString(),
		EMA.ToString(),
		BBU.ToString(),
		BBL.ToString(),
		ATR.ToString(),
		RSI.ToString(),
		STO.ToString(),
		MACD.ToString(),
		HMACD.ToString(),
	}
}

type IndicatorName string

const (
	MA    IndicatorName = "MovingAverge"
	VMA   IndicatorName = "VolumeMovingAverage"
	LHMA  IndicatorName = "LowHighChangeMovingAverage"
	OCAMA IndicatorName = "OpenCloseAbsoluteChangeMovingAverage"
	EMA   IndicatorName = "ExponentialMovingAverage"
	BBU   IndicatorName = "BollingerUpperBand"
	BBL   IndicatorName = "BollingerLowerBand"
	ATR   IndicatorName = "AverageTrueRage"
	RSI   IndicatorName = "RelativeStrengthIndex"
	STO   IndicatorName = "Stochastic"
	MACD  IndicatorName = "MACD"
	HMACD IndicatorName = "MACDHistogram"
)

func (n IndicatorName) getIndicator(ts *ta.TimeSeries, param interface{}) ta.Indicator {
	switch n {
	case EMA:
		return ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), param.(int))
	case MA:
		return ta.NewSimpleMovingAverage(ta.NewClosePriceIndicator(ts), param.(int))
	case VMA:
		return ta.NewSimpleMovingAverage(ta.NewVolumeIndicator(ts), param.(int))
	case LHMA:
		return ta.NewSimpleMovingAverage(NewCandleLowHighChangeIndicator(ts), param.(int))
	case OCAMA:
		return ta.NewSimpleMovingAverage(NewCandleOpenCloseAbsoluteChange(ts), param.(int))
	case BBU:
		return ta.NewBollingerUpperBandIndicator(ta.NewClosePriceIndicator(ts), param.(int), 2)
	case BBL:
		return ta.NewBollingerLowerBandIndicator(ta.NewClosePriceIndicator(ts), param.(int), 2)
	case ATR:
		return ta.NewAverageTrueRangeIndicator(ts, param.(int))
	case RSI:
		return ta.NewRelativeStrengthIndexIndicator(ta.NewClosePriceIndicator(ts), param.(int))
	case STO:
		return ta.NewFastStochasticIndicator(ts, param.(int))
	case MACD:
		windows := param.([]int)
		if len(windows) < 2 {
			return ta.NewConstantIndicator(float64(0))
		}
		sort.Slice(windows, func(i, j int) bool {
			return windows[i] < windows[j]
		})
		return ta.NewDifferenceIndicator(ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), windows[0]), ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), windows[1]))
	case HMACD:
		windows := param.([]int)
		if len(windows) < 3 {
			return ta.NewConstantIndicator(float64(0))
		}
		sort.Slice(windows, func(i, j int) bool {
			return windows[i] < windows[j]
		})
		macd := ta.NewDifferenceIndicator(ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), windows[0]), ta.NewEMAIndicator(ta.NewClosePriceIndicator(ts), windows[2]))
		return ta.NewDifferenceIndicator(macd, ta.NewEMAIndicator(macd, windows[1]))
	default:
		return ta.NewConstantIndicator(float64(0))
	}
}

func (n IndicatorName) ToKey(i ...int) string {
	if len(i) == 0 {
		return string(n)
	}
	out := string(n)
	for _, j := range i {
		out = out + "-" + strconv.Itoa(j)
	}
	return out
}

func (n IndicatorName) ToString() string {
	return string(n)
}
