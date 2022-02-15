package market

import "fmt"

const (
	TRADER    = "trader"
	WATCHER   = "watcher"
	MANAGER   = "manager"
	STREAMER  = "streamer"
	EVALUATOR = "evaluator"

	SimpleDateTimeFormat = "01/02/2006T15:04:05"
	SimpleDateFormat     = "01/02/2006"

	SimpleTimeFormat   = "15:04:05"
	SimpleDateFormatV2 = "2006-01-02"
)

var (
	simpleLayout = fmt.Sprint(SimpleDateFormatV2, "T", SimpleTimeFormat)
)
