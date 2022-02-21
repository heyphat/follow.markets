package database

import (
	"context"
	"fmt"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
	notion "github.com/jomei/notionapi"
)

type Notion struct {
	logger        *log.Logger
	isInitialized bool

	client  *notion.Client
	configs *config.Notion

	setupDB *notion.Database
	notisDB *notion.Database
}

func newNotionClient(configs *config.Configs) Notion {
	n := &Notion{configs: configs.Database.Notion, logger: log.NewLogger()}
	n.client = notion.NewClient(notion.Token(configs.Database.Notion.Token))
	var err error
	if n.setupDB, err = n.client.Database.Get(context.Background(), notion.DatabaseID(configs.Database.Notion.SetDBID)); err != nil {
		n.logger.Error.Println(n.newLog(err.Error()))
		return Notion{}
	}
	if n.notisDB, err = n.client.Database.Get(context.Background(), notion.DatabaseID(configs.Database.Notion.NotiDBID)); err != nil {
		n.logger.Error.Println(n.newLog(err.Error()))
		return Notion{}
	}
	n.isInitialized = true
	return *n
}

func (n Notion) newPageRequest(id notion.DatabaseID, ps map[string]notion.Property) *notion.PageCreateRequest {
	return &notion.PageCreateRequest{
		Parent: notion.Parent{
			Type:       notion.ParentTypeDatabaseID,
			DatabaseID: id,
		},
		Properties: ps,
	}
}

func (n Notion) Disconnect() {}

func (n Notion) IsInitialized() bool { return n.isInitialized }

func (n Notion) InsertSetups(ss []*Setup) (bool, error) {
	if !n.isInitialized {
		return false, nil
	}
	for _, s := range ss {
		if _, err := n.client.Page.Create(context.Background(), n.newPageRequest(notion.DatabaseID(n.configs.SetDBID), s.convertNotion(n.setupDB.Properties))); err != nil {
			n.logger.Error.Println(n.newLog(err.Error()))
			return false, err
		}
	}
	return true, nil
}

func (n Notion) InsertOrUpdateSetups(ss []*Setup) (bool, error) {
	if !n.isInitialized {
		return false, nil
	}

	filters := make(map[notion.FilterOperator][]notion.PropertyFilter, 1)
	for _, s := range ss {
		ot := notion.Date(s.OrderTime)
		tickerFilter := notion.PropertyFilter{Property: "Ticker", Text: &notion.TextFilterCondition{Equals: s.Ticker}}
		marketFilter := notion.PropertyFilter{Property: "Market", Select: &notion.SelectFilterCondition{Equals: s.Market}}
		brokerFilter := notion.PropertyFilter{Property: "Broker", Select: &notion.SelectFilterCondition{Equals: s.Broker}}
		orderIDFilter := notion.PropertyFilter{Property: "OrderID", Number: &notion.NumberFilterCondition{Equals: float64(s.OrderID)}}
		orderTimeFilter := notion.PropertyFilter{Property: "OrderTime", Date: &notion.DateFilterCondition{Equals: &ot}}
		filters[notion.FilterOperatorAND] = []notion.PropertyFilter{tickerFilter, marketFilter, brokerFilter, orderIDFilter, orderTimeFilter}
		comp := notion.CompoundFilter(filters)
		rsp, err := n.client.Database.Query(context.Background(), notion.DatabaseID(n.configs.SetDBID), &notion.DatabaseQueryRequest{CompoundFilter: &comp})
		if err != nil {
			return false, err
		}
		if len(rsp.Results) > 0 {
			if _, err := n.client.Page.Update(context.Background(),
				notion.PageID(rsp.Results[0].ID.String()),
				&notion.PageUpdateRequest{Properties: s.convertNotion(n.setupDB.Properties)}); err != nil {
				return false, err
			}
		} else {
			if _, err := n.InsertSetups([]*Setup{s}); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (n Notion) InsertNotifications(ns []*Notification) (bool, error) {
	if !n.isInitialized {
		return false, nil
	}
	for _, nt := range ns {
		if _, err := n.client.Page.Create(context.Background(), n.newPageRequest(notion.DatabaseID(n.configs.NotiDBID), nt.convertNotion(n.notisDB.Properties))); err != nil {
			n.logger.Error.Println(n.newLog(err.Error()))
			return false, err
		}
	}
	return true, nil
}

func (n Notion) GetSetups(r *runner.Runner, opts *QueryOptions) ([]*Setup, error) {
	return nil, nil
}

func (n Notion) newLog(msg string) string {
	return fmt.Sprintf("[notion]: %s", msg)
}
