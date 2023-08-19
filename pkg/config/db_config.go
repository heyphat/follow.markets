package config

type MongoDB struct {
	URI      string `json:"uri"`
	DBName   string `json:"db_name"`
	SetUpCol string `json:"setup_col_name"`
	NotiCol  string `json:"notification_col_name"`
}

type Notion struct {
	Token               string `json:"integration_token"`
	SetDBID             string `json:"setup_db_id"`
	NotiDBID            string `json:"notification_db_id"`
	BacktestDBID        string `json:"backtest_db_id"`
	GeneralBacktestDBID string `json:"general_backtest_db_id"`
}
