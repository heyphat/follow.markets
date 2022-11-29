package techanex

import (
	ta "github.com/heyphat/techan"
	"github.com/sdcoffey/big"
)

type candleLowHighChange struct {
	*ta.TimeSeries
}

// NewCandleHighLowChangeIndicator returns an indicator which returns the change in high low of a candle for a given index.
func NewCandleLowHighChangeIndicator(series *ta.TimeSeries) ta.Indicator {
	return candleLowHighChange{series}
}

func (hl candleLowHighChange) Calculate(index int) big.Decimal {
	return change(hl.Candles[index].MinPrice, hl.Candles[index].MaxPrice)
}

type candleOpenCloseAbsoluteChange struct {
	*ta.TimeSeries
}

// NewCandleOpenCloseAbsoluteChange returns an indicator which returns the change in abs(close->open) of a candle for a given index.
func NewCandleOpenCloseAbsoluteChange(series *ta.TimeSeries) ta.Indicator {
	return candleOpenCloseAbsoluteChange{series}
}

func (oc candleOpenCloseAbsoluteChange) Calculate(index int) big.Decimal {
	if oc.Candles[index].OpenPrice.GTE(oc.Candles[index].ClosePrice) {
		return change(oc.Candles[index].ClosePrice, oc.Candles[index].OpenPrice)
	}
	return change(oc.Candles[index].OpenPrice, oc.Candles[index].ClosePrice)
}
