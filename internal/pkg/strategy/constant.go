package strategy

type CandleLevel string

const (
	CandleFixed     CandleLevel = "FIXED"
	CandleOpen      CandleLevel = "OPEN"
	CandleClose     CandleLevel = "CLOSE"
	CandleHigh      CandleLevel = "HIGH"
	CandleLow       CandleLevel = "LOW"
	CandleVolume    CandleLevel = "VOLUME"
	CandleTrade     CandleLevel = "TRADE_NUM"
	CandleHighLow   CandleLevel = "HIGH_LOW"
	CandleCloseOpen CandleLevel = "CLOSE_OPEN"
	CandleHighOpen  CandleLevel = "HIGH_OPEN"
)

type TradeLevel string

const (
	TradeFixed     TradeLevel = "FIXED"
	TradePrice     TradeLevel = "PRICE"
	TradeVolume    TradeLevel = "VOLUME"
	TradeUSDVolume TradeLevel = "USD_VOLUME"
)

const (
	ComparableMap string = `
	"ticker": "BTCUSDT",
	"time_period": ["1m", "3m", "5m", "10m", "15m", "30m"],
	"time_frame": 0,
	"candle": ["CLOSE", "HIGH", "LOW", "VOLUME", "TRADE_NUM",
						 "TRADE_NUM", "HIGH_LOW", "CLOSE_OPEN", "HIGH_OPEN"],
	"indicator": ["EMA", "MA"]
`
)

var (
	candleLevels = []string{
		"FIXED", "CLOSE", "HIGH", "LOW", "VOLUME", "TRADE_NUM",
		"TRADE_NUM", "HIGH_LOW", "CLOSE_OPEN", "HIGH_OPEN",
	}
	tradeLevels = []string{
		"USD_VOLUME", "VOLUME", "PRICE", "FIXED",
	}
)
