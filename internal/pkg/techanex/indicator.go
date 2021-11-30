package techanex

import (
	"fmt"
	"strings"

	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"
)

type IndicatorConfigs map[IndicatorName][]int

func NewDefaultIndicatorConfigs() IndicatorConfigs {
	configs := make(map[IndicatorName][]int, 4)
	configs[EMA] = []int{9, 25, 50}
	configs[MA] = []int{99, 200}
	configs[BBU] = []int{25}
	configs[BBL] = []int{25}
	configs[ATR] = []int{10}
	//configs[LEVL] = []float64{}
	return configs
}

type Indicator struct {
	Period  ta.TimePeriod
	IndiMap map[string]big.Decimal
}

func NewIndicator(period ta.TimePeriod, configs IndicatorConfigs) *Indicator {
	inds := make(map[string]big.Decimal)
	for k, v := range configs {
		if len(v) == 0 {
			inds[k.ToString()] = big.ZERO
		}
		for _, window := range v {
			inds[k.ToKey(window)] = big.ZERO
		}
	}
	return &Indicator{
		Period:  period,
		IndiMap: inds,
	}
}

func (i *Indicator) Calculate(configs IndicatorConfigs, candles *ta.TimeSeries, index int) {
	closePrices := ta.NewClosePriceIndicator(candles)
	for k, v := range configs {
		var ind ta.Indicator
		if len(v) == 0 {
			ind = k.getIndicator(closePrices, 0)
		}
		for _, window := range v {
			ind = k.getIndicator(closePrices, window)
			i.IndiMap[k.ToKey(window)] = ind.Calculate(index)
		}
	}
}

type IndicatorSeries struct {
	Indicators []*Indicator
	Configs    IndicatorConfigs
}

func NewIndicatorSeries(configs IndicatorConfigs) *IndicatorSeries {
	if configs == nil {
		configs = NewDefaultIndicatorConfigs()
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
	inds := map[string]ta.Indicator{}
	for k, v := range is.Configs {
		if len(v) == 0 {
			inds[k.ToString()] = k.getIndicator(closePrices, 0)
		}
		for _, window := range v {
			inds[k.ToKey(window)] = k.getIndicator(closePrices, window)
		}
	}
	for index := 0; index < len(s.Candles); index++ {
		i := NewIndicator(s.Candles[index].Period, is.Configs)
		for k, v := range is.Configs {
			if len(v) == 0 {
				i.IndiMap[k.ToString()] = inds[k.ToString()].Calculate(index)
			}
			for _, window := range v {
				i.IndiMap[k.ToKey(window)] = inds[k.ToKey(window)].Calculate(index)
			}
		}
		if !is.addIndicator(i) {
			return false
		}
	}
	return true
}

func (i *Indicator) String() string {
	vs := []string{}
	for k, v := range i.IndiMap {
		vs = append(vs, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.TrimSpace(fmt.Sprintf(
		`
 Time:	%s
 %s
	`,
		i.Period,
		strings.Join(vs, "\n"),
	))
}
