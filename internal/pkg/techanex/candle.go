package techanex

import (
	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"
)

func change(purchased, sold big.Decimal) big.Decimal {
	if purchased.EQ(big.ZERO) {
		return big.ZERO
	}
	return sold.Sub(purchased).Div(purchased).Mul(big.NewDecimal(100.0))
}

func LowHigh(low, high big.Decimal) big.Decimal {
	return change(low, high)
}

func OpenClose(open, closeP big.Decimal) big.Decimal {
	return change(open, closeP)
}

func OpenHigh(open, high big.Decimal) big.Decimal {
	return change(open, high)
}

func OpenLow(open, low big.Decimal) big.Decimal {
	return change(open, low)
}

func LowClose(low, closeP big.Decimal) big.Decimal {
	return change(low, closeP)
}

func HighClose(high, closeP big.Decimal) big.Decimal {
	return change(high, closeP)
}

type CandleJSON struct {
	Time   int64  `json:"t"`
	Open   string `json:"o"`
	High   string `json:"h"`
	Low    string `json:"l"`
	Close  string `json:"c"`
	Volume string `json:"v"`
	Trade  uint   `json:"tc"`
}

type CandlesJSON []CandleJSON

func Candle2JSON(c *ta.Candle) CandleJSON {
	return CandleJSON{
		Time:  c.Period.Start.Unix(),
		Open:  c.OpenPrice.FormattedString(2),
		High:  c.MaxPrice.FormattedString(2),
		Low:   c.MinPrice.FormattedString(2),
		Close: c.ClosePrice.FormattedString(2),
		Trade: c.TradeCount,
	}
}
