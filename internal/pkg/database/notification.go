package database

import (
	"time"

	notion "github.com/heyphat/notionapi"
)

type Notification struct {
	Ticker    string    `bson:"ticker" json:"ticker"`
	Market    string    `json:"market" json:"market"`
	Broker    string    `bson:"broker" json:"broker"`
	Signal    string    `bson:"signal" json:"signal"`
	ClientID  *string   `bson:"client_id,omitempty" json:"client_id,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	URL       string    `bson:"url", json:"url"`
}

func (n Notification) convertNotion(ps map[string]notion.PropertyConfig) map[string]notion.Property {
	out := make(map[string]notion.Property, len(ps))
	timeT := notion.Date(n.CreatedAt)
	for k, _ := range ps {
		switch k {
		case "Ticker":
			out[k] = notion.TitleProperty{Title: []notion.RichText{notion.RichText{Text: notion.Text{Content: n.Ticker}}}}
		case "Broker":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: n.Broker}}
		case "Market":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: n.Market}}
		case "Signal":
			out[k] = notion.SelectProperty{Select: notion.Option{Name: n.Signal}}
		case "URL":
			out[k] = notion.URLProperty{URL: n.URL}
		case "ClientID":
			if n.ClientID != nil {
				out[k] = notion.TitleProperty{Title: []notion.RichText{notion.RichText{Text: notion.Text{Content: *n.ClientID}}}}
			}
		case "CreatedAt":
			out[k] = notion.DateProperty{Date: notion.DateObject{Start: &timeT}}
		}
	}
	return out
}
