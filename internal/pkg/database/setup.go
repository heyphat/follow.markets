package database

import (
	"time"

	notion "github.com/jomei/notionapi"
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
	UsedLeverage   string    `bson:"leverage" json:"leverage"`
	AccTradingFee  string    `bson:"commission" json:"commission"`
	TradingFeeAss  string    `bson:"commission_asset" json:"commission_asset"`
	AvgFilledPrice string    `bson:"avg_filled_price" json:"avg_filled_price"`
	AccFilledQtity string    `bson:"acc_filled_quantity" json:"acc_filled_quantity"`
	PNL            string    `bson:"pnl" json:"pnl"`
	DollarPNL      string    `bson:"dollar_pnl" json:"dollar_pnl"`
	LastUpdatedAt  time.Time `bson:"last_updated_at" json:"last_updated_at"`
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

func (s *Setup) convertNotion(ps map[string]notion.PropertyConfig) map[string]notion.Property {
	out := make(map[string]notion.Property, len(ps))
	orderT := notion.Date(s.OrderTime)
	lastT := notion.Date(s.LastUpdatedAt)
	for k, _ := range ps {
		switch k {
		case "Ticker":
			out[k] = notion.TitleProperty{Title: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.Ticker}}}}
		case "Broker":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.Broker}}
		case "Market":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.Market}}
		case "Signal":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.Signal}}
		case "OrderID":
			out[k] = notion.NumberProperty{Number: float64(s.OrderID)}
		case "OrderTime":
			out[k] = notion.DateProperty{Date: notion.DateObject{Start: &orderT}}
		case "OrderSide":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.OrderSide}}
		case "OrderPrice":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.OrderPrice}}}}
		case "OrderQuantity":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.OrderQtity}}}}
		case "OrderStatus":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.OrderStatus}}
		case "Commission":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.AccTradingFee}}}}
		case "CommissionAsset":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: s.TradingFeeAss}}
		case "Leverage":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.UsedLeverage}}}}
		case "FilledPrice":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.AvgFilledPrice}}}}
		case "FilledQuantity":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.AccFilledQtity}}}}
		case "PNL":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.PNL}}}}
		case "DollarPNL":
			out[k] = notion.RichTextProperty{RichText: []notion.RichText{notion.RichText{Text: notion.Text{Content: s.DollarPNL}}}}
		case "NTrades":
			out[k] = notion.NumberProperty{Number: float64(len(s.Trades))}
		case "LastUpdatedAt":
			out[k] = notion.DateProperty{Date: notion.DateObject{Start: &lastT}}
		}
	}
	return out
}
