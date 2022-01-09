package strategy

import (
	"testing"
	"time"

	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	bn "github.com/adshao/go-binance/v2"
	"github.com/stretchr/testify/assert"
)

func Test_MapDecimal(t *testing.T) {
	candleComparable := ComparableObject{
		Name:       "OPEN",
		Multiplier: nil,
	}
	comparable := Comparable{
		TimePeriod: 300,
		TimeFrame:  0,
		Candle:     &candleComparable}

	r := runner.NewRunner("BTCUSDT", nil)

	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	duration := time.Duration(comparable.TimePeriod) * time.Second
	candle1 := tax.ConvertBinanceKline(kline, &duration)
	ok := r.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	candle2 := tax.ConvertBinanceKline(kline, &duration)
	candle2.Period.Start = candle1.Period.End
	candle2.Period.End = candle2.Period.Start.Add(duration)
	ok = r.SyncCandle(candle2)
	assert.EqualValues(t, true, ok)

	// map candle test
	_, val, ok := comparable.mapDecimal(r, nil)
	assert.EqualValues(t, true, ok)
	assert.EqualValues(t, "0.0", val.FormattedString(1))

	comparable = Comparable{
		TimePeriod: 300,
		TimeFrame:  1,
		Candle:     &candleComparable,
	}

	_, val, ok = comparable.mapDecimal(r, nil)
	assert.EqualValues(t, true, ok)
	assert.EqualValues(t, "0.0", val.FormattedString(1))

	/// map fundamental test
	fundamental := runner.Fundamental{
		MaxSupply:         10,
		TotalSupply:       5,
		CirculatingSupply: 2,
	}
	r.SetFundamental(&fundamental)

	fundamentalComparable := ComparableObject{
		Name:       "MARKET_CAP",
		Multiplier: nil,
	}
	comparable = Comparable{
		TimePeriod:  300,
		TimeFrame:   1,
		Fundamental: &fundamentalComparable,
	}
	_, val, ok = comparable.mapDecimal(r, nil)
	assert.EqualValues(t, true, ok)
	assert.EqualValues(t, "1.0", val.FormattedString(1))

}
