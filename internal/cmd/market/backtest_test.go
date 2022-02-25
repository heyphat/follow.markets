package market

import (
	"io/ioutil"
	"testing"
	"time"

	db "follow.markets/internal/pkg/database"
	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Backtest_Summary(t *testing.T) {
	consigPath := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&consigPath)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, "development", configs.Stage)

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)

	signal, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	btdb := db.Backtest{
		ID:            1,
		Name:          "sample",
		Balance:       10,
		Market:        runner.Cash,
		LossTolerance: 0.01,
		ProfitMargin:  0.02,
		Signal:        signal,
		Start:         time.Now().Add(-time.Minute * 10),
		End:           time.Now(),
		Status:        db.BacktestStatusUnknown,
	}
	bt := newBacktest(&btdb)

	sm := bt.summary("")
	assert.EqualValues(t, 0, sm["Profit"])
	assert.EqualValues(t, 0, sm["PctGain"])
}
