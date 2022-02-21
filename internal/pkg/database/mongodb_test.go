package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"follow.markets/pkg/config"
)

func mongoDBTestSuit() (MongoDB, *Setup, error) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	if err != nil {
		return MongoDB{}, nil, err
	}
	db := newMongDBClient(configs)

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

func Test_MongoDB(t *testing.T) {
	db, _, err := mongoDBTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)
}

func Test_MongoDB_InsertSetups(t *testing.T) {
	db, st, err := mongoDBTestSuit()
	defer db.Disconnect()
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, db.isInitialized)

	ok, err := db.InsertSetups([]*Setup{st})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, ok)
}

func Test_MongoDB_FindSetup(t *testing.T) {
	db, st, err := mongoDBTestSuit()
	assert.EqualValues(t, nil, err)

	nst, err := db.findSetup(st)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1645241564, nst.OrderTime.Unix())
}

func Test_MongoDB_InsertOrUpdateSetup(t *testing.T) {
	db, st, err := mongoDBTestSuit()
	assert.EqualValues(t, nil, err)

	st.AvgFilledPrice = "60"
	ok, err := db.InsertOrUpdateSetups([]*Setup{st})
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, true, ok)
}

func Test_MongoDB_InsertNotifications(t *testing.T) {
	db, _, err := mongoDBTestSuit()
	assert.EqualValues(t, nil, err)

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
