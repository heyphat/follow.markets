package config

type MongoDB struct {
	URI      string `json:"uri"`
	DBName   string `json:"db_name"`
	SetUpCol string `json:"setup_col_name"`
}
