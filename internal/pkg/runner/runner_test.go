package runner

import (
	"testing"
	"time"

	bn "github.com/adshao/go-binance/v2"
	ta "github.com/heyphat/techan"
	"github.com/stretchr/testify/assert"

	tax "follow.markets/internal/pkg/techanex"
)

func Test_NewRunner(t *testing.T) {
	configs := &RunnerConfigs{
		LFrames: []time.Duration{
			time.Minute,
			10 * time.Minute,
		},
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
	runner := NewRunner("BTCUSDT", configs)
	returnedConfigs := runner.GetConfigs()
	assert.EqualValues(t, time.Minute, returnedConfigs.LFrames[0])
}

func Test_SyncCandle(t *testing.T) {
	configs := &RunnerConfigs{
		LFrames: []time.Duration{
			time.Minute,
			5 * time.Minute,
			15 * time.Minute,
		},
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
	runner := NewRunner("BTCUSDT", configs)

	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle1 := tax.ConvertBinanceKline(kline, nil)

	ok := runner.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	for d, line := range runner.lines {
		assert.EqualValues(t, 0.0, line.Candles.LastCandle().OpenPrice.Float())
		assert.EqualValues(t, 0.01, line.Candles.LastCandle().MinPrice.Float())

		assert.EqualValues(t, kline.OpenTime, line.Candles.LastCandle().Period.Start.Unix()*1000)
		assert.EqualValues(t, kline.OpenTime+d.Milliseconds(), line.Candles.LastCandle().Period.End.Unix()*1000)
	}

	candle2 := tax.ConvertBinanceKline(kline, nil)
	candle2.Period.Start = candle2.Period.End
	candle2.Period.End = candle2.Period.Start.Add(time.Minute)

	ok = runner.SyncCandle(candle2)
	assert.EqualValues(t, true, ok)

	for d, line := range runner.lines {
		assert.EqualValues(t, 0.0, line.Candles.LastCandle().OpenPrice.Float())
		assert.EqualValues(t, 0.01, line.Candles.LastCandle().MinPrice.Float())
		switch d {
		case time.Minute:
			assert.EqualValues(t, 2, len(line.Candles.Candles))
			assert.EqualValues(t, 2, len(line.Indicators.Indicators))
			assert.EqualValues(t, 308, line.Candles.LastCandle().TradeCount)
		case 5 * time.Minute:
			assert.EqualValues(t, 1, len(line.Candles.Candles))
			assert.EqualValues(t, 1, len(line.Indicators.Indicators))
			assert.EqualValues(t, 616, line.Candles.LastCandle().TradeCount)
		case 15 * time.Minute:
			assert.EqualValues(t, 1, len(line.Candles.Candles))
			assert.EqualValues(t, 1, len(line.Indicators.Indicators))
			assert.EqualValues(t, 616, line.Candles.LastCandle().TradeCount)
		}
	}
}

func Test_AddNewLine(t *testing.T) {
	configs := &RunnerConfigs{
		LFrames: []time.Duration{
			time.Minute,
			5 * time.Minute,
		},
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
	runner := NewRunner("BTCUSDT", configs)

	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle1 := tax.ConvertBinanceKline(kline, nil)

	ok := runner.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	candle2 := tax.ConvertBinanceKline(kline, nil)
	candle2.Period.Start = candle2.Period.End
	candle2.Period.End = candle2.Period.Start.Add(time.Minute)

	ok = runner.SyncCandle(candle2)
	assert.EqualValues(t, true, ok)

	//newFrame := time.Minute
	//ok = runner.AddNewLine(&newFrame, nil)
	//assert.EqualValues(t, true, ok)

	for d, line := range runner.lines {
		assert.EqualValues(t, 0.0, line.Candles.LastCandle().OpenPrice.Float())
		assert.EqualValues(t, 0.01, line.Candles.LastCandle().MinPrice.Float())
		switch d {
		case time.Minute:
			assert.EqualValues(t, 2, len(line.Candles.Candles))
			assert.EqualValues(t, 2, len(line.Indicators.Indicators))
			assert.EqualValues(t, 308, line.Candles.LastCandle().TradeCount)
		case 5 * time.Minute:
			assert.EqualValues(t, 1, len(line.Candles.Candles))
			assert.EqualValues(t, 1, len(line.Indicators.Indicators))
			assert.EqualValues(t, 616, line.Candles.LastCandle().TradeCount)
		case 10 * time.Minute:
			assert.EqualValues(t, 1, len(line.Candles.Candles))
			assert.EqualValues(t, 1, len(line.Indicators.Indicators))
			assert.EqualValues(t, 616, line.Candles.LastCandle().TradeCount)
		}
	}
}

func Test_AddNewLineWithNewTimeSeries(t *testing.T) {
	configs := &RunnerConfigs{
		LFrames: []time.Duration{
			time.Minute,
			5 * time.Minute,
			10 * time.Minute,
		},
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
	runner := NewRunner("BTCUSDT", configs)

	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle1 := tax.ConvertBinanceKline(kline, nil)

	candle2 := tax.ConvertBinanceKline(kline, nil)
	candle2.Period = candle2.Period.Advance(1)

	series := ta.TimeSeries{
		Candles: []*ta.Candle{candle1, candle2},
	}
	for _, frame := range runner.GetConfigs().LFrames {
		ok := runner.Initialize(&series, &frame)
		assert.EqualValues(t, true, ok)
	}
	for d, line := range runner.lines {
		if d == time.Minute || d == time.Minute*5 {
			continue
		}
		assert.EqualValues(t, 0.0, line.Candles.LastCandle().OpenPrice.Float())
		assert.EqualValues(t, 0.01, line.Candles.LastCandle().MinPrice.Float())
		if d == time.Minute {
			assert.EqualValues(t, 2, len(line.Candles.Candles))
			assert.EqualValues(t, 2, len(line.Indicators.Indicators))
			assert.EqualValues(t, 308, line.Candles.LastCandle().TradeCount)
		} else {
			assert.EqualValues(t, 1, len(line.Candles.Candles))
			assert.EqualValues(t, 1, len(line.Indicators.Indicators))
			assert.EqualValues(t, 616, line.Candles.LastCandle().TradeCount)
		}
	}
}

func Test_Fundamental(t *testing.T) {
	configs := &RunnerConfigs{
		LFrames: []time.Duration{
			time.Minute,
			5 * time.Minute,
		},
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
	runner := NewRunner("BTCUSDT", configs)

	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle1 := tax.ConvertBinanceKline(kline, nil)

	ok := runner.SyncCandle(candle1)
	assert.EqualValues(t, true, ok)

	candle2 := tax.ConvertBinanceKline(kline, nil)
	candle2.Period.Start = candle2.Period.End
	candle2.Period.End = candle2.Period.Start.Add(time.Minute)

	ok = runner.SyncCandle(candle2)
	assert.EqualValues(t, true, ok)

	fundamental := Fundamental{
		MaxSupply:         10,
		TotalSupply:       5,
		CirculatingSupply: 2,
	}

	mcap := runner.GetCap()
	assert.EqualValues(t, "0.0", mcap.FormattedString(1))

	runner.SetFundamental(fundamental)

	mcap = runner.GetCap()
	assert.EqualValues(t, "1.0", mcap.FormattedString(1))

	float := runner.GetFloat()
	assert.EqualValues(t, "2.0", float.FormattedString(1))
}
