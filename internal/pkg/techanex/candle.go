package techanex

import (
	"fmt"

	ta "github.com/heyphat/techan"
	"github.com/sdcoffey/big"
)

const (
	SimpleDateTimeFormat = "01/02/2006T15:04:05"
	SimpleDateFormat     = "01/02/2006"

	SimpleTimeFormat   = "15:04:05"
	SimpleDateFormatV2 = "2006-01-02"
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

func MidPoint(this, that big.Decimal) big.Decimal {
	return this.Add(that).Div(big.NewDecimal(2.0))
}

type CandleJSON struct {
	StartTime string `json:"st"`
	EndTime   string `json:"et"`
	Open      string `json:"o"`
	High      string `json:"h"`
	Low       string `json:"l"`
	Close     string `json:"c"`
	Volume    string `json:"v"`
	Trade     uint   `json:"tc"`
}

type CandlesJSON []CandleJSON

func Candle2JSON(c *ta.Candle) *CandleJSON {
	if c == nil {
		return nil
	}
	layout := fmt.Sprint(SimpleDateFormatV2, "T", SimpleTimeFormat)
	return &CandleJSON{
		StartTime: c.Period.Start.Format(layout),
		EndTime:   c.Period.End.Format(layout),
		Open:      c.OpenPrice.FormattedString(2),
		High:      c.MaxPrice.FormattedString(2),
		Low:       c.MinPrice.FormattedString(2),
		Close:     c.ClosePrice.FormattedString(2),
		Volume:    c.Volume.FormattedString(2),
		Trade:     c.TradeCount,
	}
}
