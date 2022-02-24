package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type MongoDB struct {
	client  *mongo.Client
	configs *config.MongoDB

	logger *log.Logger

	isInitialized bool
}

func newMongDBClient(configs *config.Configs) MongoDB {
	db := &MongoDB{logger: log.NewLogger(), configs: configs.Database.MongoDB}
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(configs.Database.MongoDB.URI).
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	if db.client, err = mongo.Connect(ctx, clientOptions); err != nil {
		db.logger.Error.Println(db.newLog(err.Error()))
		return MongoDB{}
	}
	db.isInitialized = true
	return *db
}

func (db MongoDB) Disconnect() {}

func (db MongoDB) IsInitialized() bool { return db.isInitialized }

func (db MongoDB) InsertSetups(ss []*Setup) (bool, error) {
	if !db.isInitialized {
		return false, nil
	}
	iss := make([]interface{}, len(ss))
	for i, s := range ss {
		iss[i] = s
	}
	_, err := db.client.Database(db.configs.DBName).
		Collection(db.configs.SetUpCol).
		InsertMany(context.Background(), iss)
	return true, err
}

func (db MongoDB) InsertOrUpdateSetups(ss []*Setup) (bool, error) {
	if !db.isInitialized {
		return false, nil
	}
	iss := make([]interface{}, len(ss))
	for i, s := range ss {
		iss[i] = s
	}
	opts := options.FindOneAndReplace().SetUpsert(true)
	for i, s := range ss {
		filters := bson.M{
			"ticker":     bson.D{{"$eq", s.Ticker}},
			"order_id":   bson.D{{"$eq", s.OrderID}},
			"market":     bson.D{{"$eq", s.Market}},
			"broker":     bson.D{{"$eq", s.Broker}},
			"order_time": bson.D{{"$eq", s.OrderTime}},
		}
		var st Setup
		if err := db.client.Database(db.configs.DBName).
			Collection(db.configs.SetUpCol).
			FindOneAndReplace(context.TODO(), filters, iss[i], opts).
			Decode(&st); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (db MongoDB) GetSetups(r *runner.Runner, opts *QueryOptions) ([]*Setup, error) {
	if !db.isInitialized {
		return nil, nil
	}
	return nil, nil
}

func (db MongoDB) findSetup(s *Setup) (*Setup, error) {
	// this method is internal, assumming the caller checked if the
	// db is initialized already.
	filters := bson.M{
		"ticker":     bson.D{{"$eq", s.Ticker}},
		"order_id":   bson.D{{"$eq", s.OrderID}},
		"market":     bson.D{{"$eq", s.Market}},
		"broker":     bson.D{{"$eq", s.Broker}},
		"order_time": bson.D{{"$eq", s.OrderTime}},
	}
	var st Setup
	if err := db.client.Database(db.configs.DBName).
		Collection(db.configs.SetUpCol).
		FindOne(context.TODO(), filters).Decode(&st); err != nil {
		db.logger.Error.Println(db.newLog(err.Error()))
		return nil, err
	}
	return &st, nil
}

func (db MongoDB) InsertNotifications(ns []*Notification) (bool, error) {
	if !db.isInitialized {
		return false, nil
	}
	ins := make([]interface{}, len(ns))
	for i, n := range ns {
		ins[i] = n
	}
	if _, err := db.client.Database(db.configs.DBName).
		Collection(db.configs.NotiCol).
		InsertMany(context.Background(), ins); err != nil {
		db.logger.Error.Println(err)
		return false, err
	}
	return true, nil
}

func (db MongoDB) GetBacktest(id int64) (*Backtest, error) {
	return nil, errors.New("not support yet")
}

func (db MongoDB) UpdateBacktestStatus(id int64, st *BacktestStatus) error {
	return errors.New("not support yet")
}

func (db MongoDB) UpdateBacktestResults(id int64, rs *BacktestResult) error {
	return errors.New("not support yet")
}

func (db MongoDB) newLog(msg string) string {
	return fmt.Sprintf("[mongodb]: %s", msg)
}
