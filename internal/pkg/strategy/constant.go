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
	CandleLowHigh   CandleLevel = "LOW_HIGH"
	CandleOpenClose CandleLevel = "OPEN_CLOSE"
	CandleOpenHigh  CandleLevel = "OPEN_HIGH"
	CandleOpenLow   CandleLevel = "OPEN_LOW"
	CandleHighClose CandleLevel = "HIGH_CLOSE"
	CandleLowClose  CandleLevel = "LOW_CLOSE"
)

type TradeLevel string

const (
	TradeFixed     TradeLevel = "FIXED"
	TradePrice     TradeLevel = "PRICE"
	TradeVolume    TradeLevel = "VOLUME"
	TradeUSDVolume TradeLevel = "USD_VOLUME"
)

const (
	OnetimeSignal    = "ONETIME"
	ContinuousSignal = "CONTINUOUS"
)

const (
	AllNotify = "ALL"
	MidNotify = "MID"
	FstNotify = "FIRST"
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
		"FIXED", "OPEN", "CLOSE", "HIGH", "LOW", "VOLUME", "TRADE_NUM",
		"LOW_HIGH", "OPEN_CLOSE", "OPEN_HIGH", "OPEN_LOW", "HIGH_CLOSE", "LOW_CLOSE",
	}
	tradeLevels = []string{
		"USD_VOLUME", "VOLUME", "PRICE", "FIXED",
	}
)
