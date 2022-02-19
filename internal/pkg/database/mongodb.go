package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
)

type MongoDB struct {
	client  *mongo.Client
	configs *config.MongoDB

	isInitialized bool
}

func newMongDBClient(configs *config.Configs) MongoDB {
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(configs.Database.MongoDB.URI).
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return MongoDB{}
	}
	return MongoDB{
		isInitialized: true,
		configs:       configs.Database.MongoDB,
		client:        client,
	}
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
		return nil, err
	}
	return &st, nil
}
