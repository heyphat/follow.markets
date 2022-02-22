package market

import "fmt"

type Agent string

const (
	TRADER    Agent = "trader"
	WATCHER   Agent = "watcher"
	STREAMER  Agent = "streamer"
	NOTIFIER  Agent = "notifier"
	EVALUATOR Agent = "evaluator"

	SimpleDateTimeFormat = "01/02/2006T15:04:05"
	SimpleDateFormat     = "01/02/2006"

	SimpleTimeFormat   = "15:04:05"
	SimpleDateFormatV2 = "2006-01-02"
)

var (
	simpleLayout = fmt.Sprint(SimpleDateFormatV2, "T", SimpleTimeFormat)
)

const (
	TRADER_MESSAGE_IS_TRADE_ENABLED        = "🤔 IS TRADE ENABLED?"
	TRADER_MESSAGE_DISABLE_TRADE           = "❌ DISABLE TRADE"
	TRADER_MESSAGE_ENABLE_TRADE            = "✅ ENABLE TRADE"
	TRADER_MESSAGE_DISABLE_TRADE_COMPLETED = " ➡️  TRADE DISABLED."
	TRADER_MESSAGE_ENABLE_TRADE_COMPLETED  = " ➡️  TRADE ENABLED."

	TRADER_MESSAGE_UPDATE_BALANCES = "💰 FORCE UPDATE BALANCES"

	TRADER_MESSAGE_SPOT_BALANCES = "SPOT BALANCES"
	TRADER_MESSAGE_FUTU_BALANCES = "FUTU BALANCES"
)
