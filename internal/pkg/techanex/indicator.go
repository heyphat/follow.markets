package techanex

import (
	"fmt"

	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"
)

type IndicatorConfigs struct {
	ATRWindow  int
	BBSigma    float64
	MAWindows  []int
	EMAWindows []int
}

func NewIndicatorDefaultConfigs() *IndicatorConfigs {
	return &IndicatorConfigs{
		ATRWindow:  10,
		BBSigma:    2,
		MAWindows:  []int{50, 200},
		EMAWindows: []int{9, 25, 100},
	}
}

type BB struct {
	MA big.Decimal
	BU big.Decimal
	BL big.Decimal
}

func NewBB() BB {
	return BB{
		MA: big.ZERO,
		BU: big.ZERO,
		BL: big.ZERO,
	}
}

type Indicator struct {
	Period ta.TimePeriod
	BBs    map[int]BB
	ATR    big.Decimal
	EMAs   map[int]big.Decimal
}

func NewIndicator(period ta.TimePeriod, configs *IndicatorConfigs) *Indicator {
	if configs == nil {
		configs = NewIndicatorDefaultConfigs()
	}
	bbs := make(map[int]BB, len(configs.MAWindows))
	for _, wd := range configs.MAWindows {
		bbs[wd] = NewBB()
	}
	emas := make(map[int]big.Decimal, len(configs.EMAWindows))
	for _, wd := range configs.EMAWindows {
		emas[wd] = big.ZERO
	}
	return &Indicator{
		Period: period,
		BBs:    bbs,
		ATR:    big.ZERO,
		EMAs:   emas,
	}
}

func (i *Indicator) Calculate(configs *IndicatorConfigs, candles *ta.TimeSeries, index int) {
	closePrices := ta.NewClosePriceIndicator(candles)
	for window, _ := range i.BBs {
		i.calculateBB(configs, closePrices, window, index)
	}
	for window, _ := range i.EMAs {
		i.EMAs[window] = ta.NewEMAIndicator(closePrices, window).Calculate(index)
	}
	i.ATR = ta.NewAverageTrueRangeIndicator(candles, configs.ATRWindow).Calculate(index)
}

func (i *Indicator) calculateBB(configs *IndicatorConfigs, indicator ta.Indicator, window, index int) {
	ma := ta.NewSimpleMovingAverage(indicator, window)
	bbu := ta.NewBollingerUpperBandIndicator(indicator, window, configs.BBSigma)
	bbl := ta.NewBollingerUpperBandIndicator(indicator, window, configs.BBSigma)
	bb := NewBB()
	bb.MA = ma.Calculate(index)
	bb.BU = bbu.Calculate(index)
	bb.BL = bbl.Calculate(index)
	i.BBs[window] = bb
}

// From this part, the code is for IndicatorSeries, which is a corresponding part of ta.TimeSeries
type IndicatorSeries struct {
	Indicators []*Indicator
	Configs    *IndicatorConfigs
}

func NewIndicators(configs *IndicatorConfigs) *IndicatorSeries {
	if configs == nil {
		configs = NewIndicatorDefaultConfigs()
	}
	is := new(IndicatorSeries)
	is.Indicators = make([]*Indicator, 0)
	is.Configs = configs
	return is
}

func (is *IndicatorSeries) LastIndicator() *Indicator {
	if len(is.Indicators) > 0 {
		return is.Indicators[len(is.Indicators)-1]
	}
	return nil
}

func (is *IndicatorSeries) addIndicator(indicator *Indicator) bool {
	if indicator == nil {
		panic(fmt.Errorf("error adding Indicator: indicator cannot be nil"))
	}
	if is.LastIndicator() == nil || indicator.Period.Since(is.LastIndicator().Period) >= 0 {
		is.Indicators = append(is.Indicators, indicator)
		return true
	}
	return false
}

func (is *IndicatorSeries) newIndicatorsFromCandleSeries(s *ta.TimeSeries) bool {
	if s == nil || len(s.Candles) == 0 {
		return true
	}
	closePrices := ta.NewClosePriceIndicator(s)
	mas := []ta.Indicator{}
	emas := []ta.Indicator{}
	bbus := []ta.Indicator{}
	bbls := []ta.Indicator{}
	for _, window := range is.Configs.MAWindows {
		mas = append(mas, ta.NewSimpleMovingAverage(closePrices, window))
		bbus = append(bbus, ta.NewBollingerUpperBandIndicator(closePrices, window, is.Configs.BBSigma))
		bbls = append(bbls, ta.NewBollingerUpperBandIndicator(closePrices, window, is.Configs.BBSigma))
	}
	for _, window := range is.Configs.EMAWindows {
		emas = append(emas, ta.NewSimpleMovingAverage(closePrices, window))
	}
	atr := ta.NewAverageTrueRangeIndicator(s, is.Configs.ATRWindow)
	for index := 0; index < len(s.Candles); index++ {
		i := NewIndicator(s.Candles[0].Period, is.Configs)
		for j, window := range is.Configs.MAWindows {
			bb := NewBB()
			bb.MA = mas[j].Calculate(index)
			bb.BU = bbus[j].Calculate(index)
			bb.BL = bbls[j].Calculate(index)
			i.BBs[window] = bb
		}
		for j, window := range is.Configs.EMAWindows {
			i.EMAs[window] = emas[j].Calculate(index)
		}
		i.ATR = atr.Calculate(index)
		if !is.addIndicator(i) {
			return false
		}
	}
	return true
}
