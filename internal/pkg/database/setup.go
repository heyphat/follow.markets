package database

import (
	"time"
)

type QueryOptions struct {
	Start time.Time
	End   time.Time
}

type Setup struct {
	Ticker         string    `bson:"ticker" json:"ticker"`
	Market         string    `json:"market" json:"market"`
	Broker         string    `bson:"broker" json:"broker"`
	Signal         string    `bson:"signal" json:"signal"`
	OrderID        int64     `bson:"order_id" json:"order_id"`
	OrderTime      time.Time `bson:"order_time" json:"order_time"`
	OrderSide      string    `bson:"order_side" json:"order_side"`
	OrderPrice     string    `bson:"order_price" json:"order_price"`
	OrderQtity     string    `bson:"order_quantity" json:"order_quantity"`
	OrderStatus    string    `bson:"order_status" json:"order_status"`
	AccTradingFee  string    `bson:"commission" json:"commission"`
	UsedLeverage   string    `bson:"leverage" json:"leverage"`
	TradingFeeAss  string    `bson:"commission_asset" json:"commission_asset"`
	AvgFilledPrice string    `bson:"avg_filled_price" json:"avg_filled_price"`
	AccFilledQtity string    `bson:"acc_filled_quantity" json:"acc_filled_quantity"`
	PNL            string    `bson:"pnl" json:"pnl"`
	Trades         []*Trade  `bson:"trades" json:"trades"`
}

type Trade struct {
	ID       int64     `bson:"trade_id" json:"trade_id"`
	Time     time.Time `bson:"trade_time" json:"trade_time"`
	Cost     string    `bson:"commission" json:"commission"`
	CostAss  string    `bson:"commission_asset" json:"commission_asset"`
	Price    string    `bson:"price" json:"price"`
	Status   string    `bson:"status" json:"status"`
	Quantity string    `bson:"quantity" json:"quantity"`
}
