package database

import (
	"testing"
	"time"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func notionTestSuit() (Notion, *Setup, error) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	if err != nil {
		return Notion{}, nil, err
	}
	db := newNotionClient(configs)

	st := &Setup{
		Ticker:         "BTCUSDT",
		Market:         "FUTURES",
		Broker:         "Binance",
		Signal:         "sample",
		OrderID:        1,
		OrderTime:      time.Unix(1645241564, 0),
		OrderSide:      "BUY",
		OrderPrice:     "10",
		OrderQtity:     "20",
		OrderStatus:    "FILLED",
		AccTradingFee:  "1",
		UsedLeverage:   "10",
		TradingFeeAss:  "USDT",
		AvgFilledPrice: "10",
		AccFilledQtity: "20",
		PNL:            "0",
		DollarPNL:      "0",
		Trades: []*Trade{
			&Trade{
				ID:       1,
				Time:     time.Now(),
				Cost:     "1",
				CostAss:  "USDT",
				Price:    "10",
				Quantity: "20",
				Status:   "FILLED",
			},
		},
	}

	return db, st, nil
}

func Test_Notion(t *testing.T) {
	db, _, err := notionTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)
}

func Test_Notion_InsertSetups(t *testing.T) {
	db, st, err := notionTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)

	ok, err := db.InsertSetups([]*Setup{st})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, ok)
}

func Test_Notion_InsertNotifications(t *testing.T) {
	db, _, err := notionTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)
	noti := &Notification{
		Ticker:    "BTCUSDT",
		Market:    "CASH",
		Broker:    "Binance",
		Signal:    "sample",
		CreatedAt: time.Now(),
	}
	ok, err := db.InsertNotifications([]*Notification{noti})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, ok)
}

func Test_Notion_InsertOrUpdateSetups(t *testing.T) {
	db, st, err := notionTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)

	st.OrderQtity = "100"
	ok, err := db.InsertOrUpdateSetups([]*Setup{st})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, ok)
}
