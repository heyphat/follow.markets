package builder

type CandleLevel string

const (
	CandleOpen      CandleLevel = "OPEN"
	CandleClose     CandleLevel = "CLOSE"
	CandleHigh      CandleLevel = "HIGH"
	CandleLow       CandleLevel = "HIGH"
	CandleVolume    CandleLevel = "VOLUME"
	CandleTrade     CandleLevel = "TRADE_NUM"
	CandleHighLow   CandleLevel = "HIGH_LOW"
	CandleCloseOpen CandleLevel = "CLOSE_OPEN"
	CandleHighOpen  CandleLevel = "HIGH_OPEN"
)

type IndicatorName string

const (
	IndicatorEMA IndicatorName = "EMA"
	IndicatorBB  IndicatorName = "BollingerBand"
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
		"CLOSE", "HIGH", "LOW", "VOLUME", "TRADE_NUM",
		"TRADE_NUM", "HIGH_LOW", "CLOSE_OPEN", "HIGH_OPEN",
	}
	indicatorNames = []string{
		"EMA", "MA",
	}
)
