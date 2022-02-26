package market

import (
	"errors"
	"fmt"

	ta "github.com/itsphat/techan"

	db "follow.markets/internal/pkg/database"
	"follow.markets/pkg/log"
)

type tester struct {
	savePath string

	// shared properties with other market participants
	logger   *log.Logger
	provider *provider
}

func newTester(participants *sharedParticipants) (*tester, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &tester{
		savePath: "./test_result",
		logger:   participants.logger,
		provider: participants.provider,
	}, nil
}

func (t *tester) test(id int64) (*backtest, error) {
	status := db.BacktestStatusProcessing
	go t.provider.dbClient.UpdateBacktestStatus(id, &status)
	data, err := t.provider.dbClient.GetBacktest(id)
	if err != nil {
		status = db.BacktestStatusError
		t.provider.dbClient.UpdateBacktestStatus(id, &status)
		return nil, err
	}
	bt := newBacktest(data)
	newStatus := &bt.bt.Status
	defer t.provider.dbClient.UpdateBacktestStatus(id, newStatus)
	candles, err := t.provider.fetchBinanceSpotKlinesV3(bt.r.GetName(), bt.r.SmallestFrame(), &fetchOptions{start: &bt.bt.Start, end: &bt.bt.End})
	if err != nil {
		bt.bt.UpdateStatus(db.BacktestStatusError)
		return nil, err
	}
	for i, c := range candles {
		if !bt.r.SyncCandle(c) {
			t.logger.Warning.Println(t.newLog(bt.r.GetName(), "failed to sync new candle on watching"))
			continue
		}
		if bt.s.ShouldEnter(i, bt.rcs) {
			bt.rcs.Operate(ta.Order{
				Side:          ta.OrderSideFromString(bt.s.EntryRule.Signal.BacktestSide("BUY")),
				Price:         c.ClosePrice,
				Amount:        bt.balance.Div(c.ClosePrice),
				Security:      bt.r.GetName(),
				ExecutionTime: c.Period.Start,
			})
			continue
		}
		if bt.s.ShouldExit(i, bt.rcs) || (bt.rcs.CurrentPosition().IsOpen() && i == len(candles)-1) {
			bt.rcs.Operate(ta.Order{
				Side:          ta.OrderSideFromString(bt.s.EntryRule.Signal.BacktestSide("SELL")),
				Price:         c.ClosePrice,
				Amount:        bt.rcs.CurrentPosition().EntranceOrder().Amount,
				Security:      bt.r.GetName(),
				ExecutionTime: c.Period.Start,
			})
		}
	}
	if err != t.provider.dbClient.UpdateBacktestResult(bt.bt.ID, bt.summary(t.savePath), bt.rcs.Trades...) {
		bt.bt.UpdateStatus(db.BacktestStatusError)
	}
	bt.bt.UpdateStatus(db.BacktestStatusCompleted)
	return bt, nil
}

// returns a log for the tester
func (t *tester) newLog(ticker, message string) string {
	return fmt.Sprintf("[tester] %s: %s", ticker, message)
}
