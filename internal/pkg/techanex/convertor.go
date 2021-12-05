package techanex

import (
	"time"

	bn "github.com/adshao/go-binance/v2"
	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"
)

func ConvertBinanceKline(kline *bn.Kline, duration *time.Duration) *ta.Candle {
	d := time.Minute
	if duration != nil {
		d = *duration
	}
	period := ta.NewTimePeriod(time.Unix(kline.OpenTime/1000, 0), d)
	candle := ta.NewCandle(period)
	candle.OpenPrice = big.NewFromString(kline.Open)
	candle.ClosePrice = big.NewFromString(kline.Close)
	candle.MaxPrice = big.NewFromString(kline.High)
	candle.MinPrice = big.NewFromString(kline.Low)
	candle.Volume = big.NewFromString(kline.Volume)
	candle.TradeCount = uint(kline.TradeNum)
	return candle
}

func ConvertBinanceStreamingKline(kline *bn.WsKlineEvent, duration *time.Duration) *ta.Candle {
	d := time.Minute
	if duration != nil {
		d = *duration
	}
	period := ta.NewTimePeriod(time.Unix(kline.Kline.StartTime/1000, 0), d)
	candle := ta.NewCandle(period)
	candle.OpenPrice = big.NewFromString(kline.Kline.Open)
	candle.ClosePrice = big.NewFromString(kline.Kline.Close)
	candle.MaxPrice = big.NewFromString(kline.Kline.High)
	candle.MinPrice = big.NewFromString(kline.Kline.Low)
	candle.Volume = big.NewFromString(kline.Kline.Volume)
	candle.TradeCount = uint(kline.Kline.TradeNum)
	return candle
}

func ConvertBinanceStreamingTrade(t *bn.WsTradeEvent) *Trade {
	trade := NewTrade()
	trade.Price = big.NewFromString(t.Price)
	trade.Quantity = big.NewFromString(t.Quantity)
	trade.TradeTime = t.TradeTime
	trade.IsBuyerMaker = t.IsBuyerMaker
	return trade
}

func ConvertBinanceStreamingAggTrade(t *bn.WsAggTradeEvent) *Trade {
	trade := NewTrade()
	trade.Price = big.NewFromString(t.Price)
	trade.Quantity = big.NewFromString(t.Quantity)
	trade.TradeTime = t.TradeTime
	trade.IsBuyerMaker = t.IsBuyerMaker
	return trade
}

func NewCandleFromCandle(candle *ta.Candle, duration *time.Duration) *ta.Candle {
	d := time.Minute
	if duration != nil {
		d = *duration
	}
	period := candle.Period
	period.Start = period.Start.Truncate(d)
	period.End = period.Start.Add(d)
	newCandle := ta.NewCandle(period)
	newCandle.OpenPrice = candle.OpenPrice
	newCandle.ClosePrice = candle.ClosePrice
	newCandle.MaxPrice = candle.MaxPrice
	newCandle.MinPrice = candle.MinPrice
	newCandle.Volume = candle.Volume
	newCandle.TradeCount = candle.TradeCount
	return newCandle
}
