package market

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/sdcoffey/big"

	"follow.market/internal/pkg/runner"
	"follow.market/internal/pkg/strategy"
	tax "follow.market/internal/pkg/techanex"
	"follow.market/pkg/log"
	ta "github.com/itsphat/techan"
)

type tester struct {
	// shared properties with other market participants
	logger   *log.Logger
	provider *provider
}

func newTester(participants *sharedParticipants) (*tester, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &tester{
		logger:   participants.logger,
		provider: participants.provider,
	}, nil
}

type tmember struct {
	balance  big.Decimal
	runner   *runner.Runner
	record   *ta.TradingRecord
	strategy *strategy.Strategy
}

func (t *tester) test(ticker string, initBalance big.Decimal, stg *strategy.Strategy, start, end time.Time) (tmember, error) {
	if initBalance.LTE(big.ZERO) {
		return tmember{}, errors.New("init balance has to be > 0")
	}
	if stg == nil || stg.EntryRule == nil {
		return tmember{}, errors.New("missing trading strategy or signal")
	}
	r := runner.NewRunner(ticker, &runner.RunnerConfigs{
		LFrames:  stg.EntryRule.Signal.GetPeriods(),
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	})
	mem := tmember{
		runner:   r,
		record:   ta.NewTradingRecord(),
		balance:  initBalance,
		strategy: stg.SetRunner(r),
	}
	candles, err := t.provider.fetchBinanceKlinesV3(ticker, r.SmallestFrame(), &fetchOptions{start: &start, end: &end})
	if err != nil {
		return mem, err
	}
	for i, c := range candles {
		if !mem.runner.SyncCandle(c) {
			t.logger.Warning.Println(t.newLog(ticker, "failed to sync new candle on watching"))
			continue
		}
		if mem.strategy.ShouldEnter(i, mem.record) {
			mem.record.Operate(ta.Order{
				Side:          stg.EntryRule.Signal.Side(ta.BUY),
				Price:         c.ClosePrice,
				Amount:        mem.balance.Div(c.ClosePrice),
				Security:      ticker,
				ExecutionTime: c.Period.Start,
			})
			continue
		}
		if mem.strategy.ShouldExit(i, mem.record) || (mem.record.CurrentPosition().IsOpen() && i == len(candles)-1) {
			mem.record.Operate(ta.Order{
				Side:          stg.EntryRule.Signal.Side(ta.SELL),
				Price:         c.ClosePrice,
				Amount:        mem.record.CurrentPosition().EntranceOrder().Amount,
				Security:      ticker,
				ExecutionTime: c.Period.Start,
			})
		}
	}
	logTrades := ta.LogTradesAnalysis{Writer: bytes.NewBufferString("")}
	_ = logTrades.Analyze(mem.record)
	fmt.Println(logTrades.Writer)
	return mem, nil
}

// returns a log for the tester
func (t *tester) newLog(ticker, message string) string {
	return fmt.Sprintf("[tester] %s: %s", ticker, message)
}
