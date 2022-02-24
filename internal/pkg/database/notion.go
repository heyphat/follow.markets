package database

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
	"follow.markets/pkg/util"
	notion "github.com/itsphat/notionapi"
)

type Notion struct {
	logger        *log.Logger
	isInitialized bool

	client  *notion.Client
	configs *config.Notion

	setupDB *notion.Database
	notisDB *notion.Database
	btestDB *notion.Database
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
	if n.btestDB, err = n.client.Database.Get(context.Background(), notion.DatabaseID(configs.Database.Notion.BacktestDBID)); err != nil {
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

func (n Notion) getBacktestPage(id int64) (*notion.Page, error) {
	// the db has be isInitialized already.
	rsp, err := n.client.Database.Query(context.Background(),
		notion.DatabaseID(n.configs.BacktestDBID),
		&notion.DatabaseQueryRequest{PropertyFilter: &notion.PropertyFilter{
			Property: "ID",
			Formula: &notion.FormulaFilterCondition{
				Number: &notion.NumberFilterCondition{
					Equals: float64(id)}},
		},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(rsp.Results) == 0 {
		return nil, errors.New("cannot find backtest")
	}
	return &rsp.Results[0], nil
}

func (n Notion) UpdateBacktestStatus(id int64, st *BacktestStatus) error {
	if !n.isInitialized {
		return errors.New("DB hasn't been initialized.")
	}
	page, err := n.getBacktestPage(id)
	if err != nil {
		return err
	}
	p := make(map[string]notion.Property, 1)
	p["Status"] = notion.SelectProperty{Select: notion.Option{Name: string(*st)}}
	if _, err := n.client.Page.Update(context.Background(),
		notion.PageID(page.ID.String()),
		&notion.PageUpdateRequest{Properties: p}); err != nil {
		return err
	}
	return nil
}

func (n Notion) UpdateBacktestResults(id int64, rs *BacktestResult) error {
	return nil
}

func (n Notion) GetBacktest(id int64) (*Backtest, error) {
	if !n.isInitialized {
		return nil, errors.New("DB hasn't been isInitialized.")
	}
	page, err := n.getBacktestPage(id)
	if err != nil {
		return nil, err
	}
	bt := &Backtest{
		ID:            id,
		CreatedAt:     util.ConvertUnixMillisecond2Time(id),
		Balance:       10000,
		LossTolerance: 0.01,
		ProfitMargin:  0.02,
	}
	for k, v := range page.Properties {
		switch k {
		case "Name":
			if v.GetType() == notion.PropertyTypeTitle {
				p := v.(*notion.TitleProperty)
				if len(p.Title) == 0 {
					continue
				}
				bt.Name = p.Title[0].Text.Content
			}
		case "Ticker":
			if v.GetType() == notion.PropertyTypeRichText {
				p := v.(*notion.RichTextProperty)
				if len(p.RichText) == 0 {
					return bt, errors.New("missing ticker")
				}
				bt.Ticker = v.(*notion.RichTextProperty).RichText[0].Text.Content
			}
		case "Balance":
			if v.GetType() == notion.PropertyTypeNumber {
				p := v.(*notion.NumberProperty)
				if p.Number > 0 {
					bt.Balance = int64(p.Number)
				}
			}
		case "Market":
			if v.GetType() == notion.PropertyTypeSelect {
				p := v.(*notion.SelectProperty)
				bt.Market, _ = runner.ValidateMarket(p.Select.Name)
			}
		case "Status":
			if v.GetType() == notion.PropertyTypeSelect {
				p := v.(*notion.SelectProperty)
				bt.Status, _ = ValidateBacktestStatus(p.Select.Name)
			}
		case "LossTolerance":
			if v.GetType() == notion.PropertyTypeNumber {
				p := v.(*notion.NumberProperty)
				if p.Number > 0 {
					bt.LossTolerance = p.Number
				}
			}
		case "ProfitMargin":
			if v.GetType() == notion.PropertyTypeNumber {
				p := v.(*notion.NumberProperty)
				if p.Number > 0 {
					bt.LossTolerance = p.Number
				}
			}
		case "Start":
			if v.GetType() == notion.PropertyTypeDate {
				p := v.(*notion.DateProperty)
				if p.Date.Start == nil {
					return nil, errors.New("missing start date")
				}
				bt.Start = time.Time(*p.Date.Start)
			}
		case "End":
			if v.GetType() == notion.PropertyTypeDate {
				p := v.(*notion.DateProperty)
				if p.Date.Start == nil {
					return nil, errors.New("missing end date")
				}
				bt.End = time.Time(*p.Date.Start)
			}
		case "Signal":
			if v.GetType() == notion.PropertyTypeFiles {
				files := v.(*notion.FilesProperty).Files
				if len(files) == 0 {
					return nil, errors.New("no signal file")
				}
				if files[0].File == nil {
					return nil, errors.New("no signal url to download")
				}
				resp, err := http.Get(files[0].File.URL)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}
				s, err := strategy.NewSignalFromBytes(body)
				if err != nil {
					return nil, err
				}
				bt.Signal = s
			}
		}
	}
	return bt, nil
}

func (n Notion) newLog(msg string) string {
	return fmt.Sprintf("[notion]: %s", msg)
}
