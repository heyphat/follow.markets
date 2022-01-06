package strategy

type CandleLevel string

const (
	CandleOpenTime     CandleLevel = "OPEN_TIME"
	CandleCloseTime    CandleLevel = "CLOSE_TIME"
	CandleFixed        CandleLevel = "FIXED"
	CandleOpen         CandleLevel = "OPEN"
	CandleClose        CandleLevel = "CLOSE"
	CandleHigh         CandleLevel = "HIGH"
	CandleLow          CandleLevel = "LOW"
	CandleVolume       CandleLevel = "VOLUME"
	CandleTrade        CandleLevel = "TRADE_COUNT"
	CandleMidOpenClose CandleLevel = "MID_OPEN_CLOSE"
	CandleMidLowHigh   CandleLevel = "MID_LOW_HIGH"
	CandleLowHigh      CandleLevel = "LOW_HIGH"
	CandleOpenClose    CandleLevel = "OPEN_CLOSE"
	CandleOpenHigh     CandleLevel = "OPEN_HIGH"
	CandleOpenLow      CandleLevel = "OPEN_LOW"
	CandleHighClose    CandleLevel = "HIGH_CLOSE"
	CandleLowClose     CandleLevel = "LOW_CLOSE"
)

type TradeLevel string

const (
	TradeFixed     TradeLevel = "FIXED"
	TradePrice     TradeLevel = "PRICE"
	TradeVolume    TradeLevel = "VOLUME"
	TradeUSDVolume TradeLevel = "USD_VOLUME"
)

const (
	OnetimeTrack    = "ONETIME"
	ContinuousTrack = "CONTINUOUS"

	AllNotify = "ALL"
	MidNotify = "MID"
	FstNotify = "FIRST"

	BullishSignal = "BULLISH"
	BearishSignal = "BEARISH"
)

var (
	// indicator levels are defined in the techanex package in /internal/pkg/strategy/constant.go
	candleLevels = []string{
		"OPEN_TIME", "CLOSE_TIME", "FIXED", "OPEN", "CLOSE", "HIGH", "LOW", "VOLUME", "TRADE_COUNT", "MID_LOW_HIGH", "MID_OPEN_CLOSE",
		"LOW_HIGH", "OPEN_CLOSE", "OPEN_HIGH", "OPEN_LOW", "HIGH_CLOSE", "LOW_CLOSE",
	}

	tradeLevels = []string{
		"USD_VOLUME", "VOLUME", "PRICE", "FIXED",
	}

	AcceptablePeriods = []int64{60, 180, 300, 900, 1800, 3600, 7200, 14400, 86400}
)
