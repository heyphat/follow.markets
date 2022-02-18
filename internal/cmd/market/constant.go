package market

import "fmt"

type Agent string

const (
	TRADER    Agent = "trader"
	WATCHER   Agent = "watcher"
	MANAGER   Agent = "manager"
	STREAMER  Agent = "streamer"
	EVALUATOR Agent = "evaluator"

	SimpleDateTimeFormat = "01/02/2006T15:04:05"
	SimpleDateFormat     = "01/02/2006"

	SimpleTimeFormat   = "15:04:05"
	SimpleDateFormatV2 = "2006-01-02"
)

var (
	simpleLayout = fmt.Sprint(SimpleDateFormatV2, "T", SimpleTimeFormat)
)
