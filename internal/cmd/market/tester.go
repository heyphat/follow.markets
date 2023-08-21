package market

import (
	"errors"
	"fmt"
	"math"

	ta "github.com/heyphat/techan"

	db "follow.markets/internal/pkg/database"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type tester struct {
	savePath string

	// shared properties with other market participants
	logger   *log.Logger
	provider *provider
}

func newTester(participants *sharedParticipants, configs *config.Configs) (*tester, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &tester{
		savePath: configs.Market.Tester.SavePath,
		logger:   participants.logger,
		provider: participants.provider,
	}, nil
}

func (t *tester) execute(id int64) error {
	status := db.BacktestStatusAccepted
	statusPointer := &status
	defer t.provider.dbClient.UpdateBacktestStatus(id, statusPointer, false)
	backtests, err := t.initTest(id)
	if err != nil {
		status = db.BacktestStatusError
		return err
	}
	*statusPointer = db.BacktestStatusProcessing
	go t.provider.dbClient.UpdateBacktestStatus(id, statusPointer, false)
	for _, backtest := range backtests {
		_, err := t.runTest(backtest.bt.ID, backtest)
		if err != nil {
			status = db.BacktestStatusError
			return err
		}
	}
	*statusPointer = db.BacktestStatusCompleted
	return nil
}

func (t *tester) initTest(id int64) ([]*backtest, error) {
	data, err := t.provider.dbClient.GetBacktest(id)
	if err != nil {
		return nil, err
	}
	tickers := []string{}
	if data.NRunners > 0 {
		gainers, err := t.provider.fetchRunners(true, int(data.NRunners))
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, gainers...)
	} else {
		losers, err := t.provider.fetchRunners(false, int(math.Abs(float64(data.NRunners))))
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, losers...)
	}
	var out []*backtest
	for _, ticker := range tickers {
		backtest := newBacktest(data.Copy(&ticker))
		backtestResultID, err := t.provider.dbClient.CreateBacktestResultItem(backtest.bt)
		if err != nil {
			return nil, err
		}
		backtest.bt.ID = backtestResultID
		out = append(out, backtest)
	}
	return out, nil
}

func (t *tester) runTest(id int64, bt *backtest) (*backtest, error) {
	// if the smallest frame is 3 minute, cannot use 5 minute in the signal criterion
	status := db.BacktestStatusProcessing
	go t.provider.dbClient.UpdateBacktestStatus(id, &status, true)
	newStatus := &bt.bt.Status
	defer t.provider.dbClient.UpdateBacktestStatus(id, newStatus, true)
	candles, err := t.provider.fetchBinanceSpotKlinesV3(bt.r.GetName(), bt.r.SmallestFrame(), &fetchOptions{start: &bt.bt.Start, end: &bt.bt.End, limit: 499})
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
			price, ok := bt.bt.Signal.TradeExecutionPrice(bt.r)
			if !ok {
				price = c.ClosePrice
			}
			bt.rcs.Operate(ta.Order{
				Side:          ta.OrderSideFromString(bt.s.EntryRule.Signal.BacktestSide("BUY")),
				Price:         price,
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
