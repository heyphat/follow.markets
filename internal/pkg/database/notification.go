package database

import "time"

type Notification struct {
	Ticker    string    `bson:"ticker" json:"ticker"`
	Market    string    `json:"market" json:"market"`
	Broker    string    `bson:"broker" json:"broker"`
	Signal    string    `bson:"signal" json:"signal"`
	ClientID  *string   `bson:"client_id,omitempty" json:"client_id,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
