package builder

import (
	"errors"

	"github.com/sdcoffey/big"

	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/util"
)

type Comparable struct {
	Ticker     string `json:"ticker"`
	TimePeriod string `json:"time_period"`
	TimeFrame  int    `json:"time_frame"`
	Candle     *struct {
		Level     CandleLevel `json:"level"`
		Effective float64     `json:"effective"`
	} `json:"candle"`
	Indicator *struct {
		Name      IndicatorName     `json:"name"`
		Config    map[string]string `json:"config"`
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
	if c.Indicator != nil && !util.StringSliceContains(indicatorNames, string(c.Indicator.Name)) {
		return errors.New("invalid indicator name")
	}
	return nil
}

func (c *Comparable) mapDecimal(s *tax.Series) (big.Decimal, bool) {
	d := big.ZERO
	return d, true
}
