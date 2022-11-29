package techanex

import (
	"fmt"
	"strings"

	ta "github.com/heyphat/techan"
	"github.com/sdcoffey/big"
)

type IndicatorConfigs map[IndicatorName][]int

func NewDefaultIndicatorConfigs() IndicatorConfigs {
	configs := make(map[IndicatorName][]int, 5)
	configs[EMA] = []int{9, 26, 50}
	configs[VMA] = []int{200}
	configs[LHMA] = []int{200}
	configs[OCAMA] = []int{200}
	configs[MA] = []int{99, 200}
	configs[BBU] = []int{26, 50}
	configs[BBL] = []int{26, 50}
	configs[ATR] = []int{10}
	configs[RSI] = []int{14}
	configs[STO] = []int{14}
	configs[MACD] = []int{9, 26}
	configs[HMACD] = []int{9, 12, 26}
	return configs
}

type Indicator struct {
	Period  ta.TimePeriod
	IndiMap map[string]big.Decimal
}

func NewIndicator(period ta.TimePeriod, configs IndicatorConfigs) *Indicator {
	inds := make(map[string]big.Decimal)
	for k, v := range configs {
		if len(v) == 0 || k == MACD || k == HMACD {
			inds[k.ToKey(v...)] = big.ZERO
			continue
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
	for k, v := range configs {
		var ind ta.Indicator
		if len(v) == 0 || k == MACD || k == HMACD {
			ind = k.getIndicator(candles, v)
			i.IndiMap[k.ToKey(v...)] = ind.Calculate(index)
			continue
		}
		for _, window := range v {
			ind = k.getIndicator(candles, window)
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
	inds := map[string]ta.Indicator{}
	for k, v := range is.Configs {
		if len(v) == 0 || k == MACD || k == HMACD {
			inds[k.ToKey(v...)] = k.getIndicator(s, v)
			continue
		}
		for _, window := range v {
			inds[k.ToKey(window)] = k.getIndicator(s, window)
		}
	}
	for index := 0; index < len(s.Candles); index++ {
		i := NewIndicator(s.Candles[index].Period, is.Configs)
		for k, v := range is.Configs {
			if len(v) == 0 || k == MACD || k == HMACD {
				i.IndiMap[k.ToKey(v...)] = inds[k.ToKey(v...)].Calculate(index)
				continue
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

type IndicatorJSON struct {
	StartTime string            `json:"st"`
	EndTime   string            `json:"et"`
	IndiMap   map[string]string `json:"indicators"`
}

type IndicatorsJSON []IndicatorJSON

func (id *Indicator) Indicator2JSON() *IndicatorJSON {
	if id == nil {
		return nil
	}
	m := make(map[string]string, len(id.IndiMap))
	for k, v := range id.IndiMap {
		m[k] = v.FormattedString(2)
	}
	layout := fmt.Sprint(SimpleDateFormatV2, "T", SimpleTimeFormat)
	return &IndicatorJSON{
		StartTime: id.Period.Start.Format(layout),
		EndTime:   id.Period.End.Format(layout),
		IndiMap:   m,
	}
}
