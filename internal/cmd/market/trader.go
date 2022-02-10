package market

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/dlclark/regexp2"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type trader struct {
	sync.Mutex
	connected        bool
	binSpotListenKey string
	binSpotBalances  *sync.Map
	binFutuListenKey string

	// trader configuration
	quoteCurrency   string
	allowedPatterns []*regexp2.Regexp
	allowedMarkets  []runner.MarketType

	minLeverage   big.Decimal
	maxLeverage   big.Decimal
	minBalance    big.Decimal
	maxPositions  big.Decimal
	profitMargin  big.Decimal
	lossTolerance big.Decimal

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type tdmember struct {
	runner   *runner.Runner
	channels *streamingChannels
}

func newTrader(participants *sharedParticipants, configs *config.Configs) (*trader, error) {
	if configs == nil || participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants or configs")
	}
	t := &trader{
		connected:       false,
		binSpotBalances: &sync.Map{},

		minBalance:    big.ZERO,
		maxPositions:  big.ZERO,
		minLeverage:   big.NewDecimal(1.0),
		maxLeverage:   big.NewDecimal(1.0),
		lossTolerance: big.NewDecimal(0.01),
		profitMargin:  big.NewDecimal(0.02),

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}
	var err error
	if err = t.updateConfigs(configs); err != nil {
	}
	if t.binSpotBalances, err = t.provider.fetchBinSpotBalances(); err != nil {
		return nil, err
	}
	if t.binSpotListenKey, t.binFutuListenKey, err = t.provider.fetchBinUserDataListenKey(); err != nil {
		return nil, err
	}
	go t.binSpotUserDataStreaming()
	go t.binFutuUserDataStreaming()
	return t, nil
}

// updateConfigs updates basic trading configuration for the trader. It overrides the current configuration.
func (t *trader) updateConfigs(configs *config.Configs) error {
	var err error
	reges := make([]*regexp2.Regexp, len(configs.Market.Trader.AllowedPatterns))
	for i, p := range configs.Market.Trader.AllowedPatterns {
		if reges[i], err = regexp2.Compile(p, 0); err != nil {
			return err
		}
	}
	t.allowedPatterns = reges
	mkts := make([]runner.MarketType, len(configs.Market.Trader.AllowedMarkets))
	for i, m := range configs.Market.Trader.AllowedMarkets {
		if mt, ok := runner.ValidateMarket(m); ok {
			mkts[i] = mt
		}
	}
	t.allowedMarkets = mkts
	t.maxPositions = big.NewDecimal(configs.Market.Trader.MaxPositions)
	t.minBalance = big.NewDecimal(configs.Market.Trader.MinBalance)
	t.lossTolerance, t.profitMargin = big.NewDecimal(0.01), big.NewDecimal(0.02)
	if configs.Market.Trader.MinLeverage > 0 {
		t.minLeverage = big.NewDecimal(configs.Market.Trader.MinLeverage)
	}
	if configs.Market.Trader.MaxLeverage > 0 {
		t.minLeverage = big.NewDecimal(configs.Market.Trader.MaxLeverage)
	}
	if configs.Market.Trader.LossTolerance > 0 {
		t.lossTolerance = big.NewDecimal(configs.Market.Trader.LossTolerance)
	}
	if configs.Market.Trader.ProfitMargin > 0 {
		t.profitMargin = big.NewDecimal(configs.Market.Trader.ProfitMargin)
	}
	t.quoteCurrency = configs.Market.Base.Crypto.QuoteCurrency
	return nil
}

// isConnected returns true when the trader is connected to other market participants, false otherwise.
func (t *trader) isConnected() bool { return t.connected }

// connect connects the trader to other market participants py listening to
// decicated channels for communication. It is called only once on ititialization.
func (t *trader) connect() {
	t.Lock()
	defer t.Unlock()
	if t.connected {
		return
	}
	go func() {
		for msg := range t.communicator.evaluator2Trader {
			go t.processEvaluatorRequest(msg)
		}
	}()
	t.connected = true
}

// getBinSpotBalances returns assets holding on Binance spot account.
func (t *trader) getBinSpotBalances() []bn.Balance {
	out := []bn.Balance{}
	t.binSpotBalances.Range(func(key, value interface{}) bool {
		out = append(out, value.(bn.Balance))
		return true
	})
	return out
}

// updatebinSpotBalances removes or adds assset to holdings on trader binSpotBalances.
// It is called once on initialization and upon receving data from websocket for UserData event.
func (t *trader) updatebinSpotBalances(bl bn.Balance) {
	t.Lock()
	defer t.Unlock()
	if big.NewFromString(bl.Free).EQ(big.ZERO) {
		t.binSpotBalances.Delete(bl.Asset)
		return
	}
	t.binSpotBalances.Store(bl.Asset, bl)
}

// isAllowedMarkets checks if the runner is allowed to trade based on its targered markets, which are SPOT or FUTURES.
func (t *trader) isAllowedMarkets(r *runner.Runner) bool {
	for _, m := range t.allowedMarkets {
		if m == r.GetMarketType() {
			return true
		}
	}
	return false
}

// isAllowedPatterns checks if the runner is allowed to trade based on its ticker name.
func (t *trader) isAllowedPatterns(r *runner.Runner) bool {
	for _, p := range t.allowedPatterns {
		isMatched, err := p.MatchString(r.GetName())
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
			continue
		}
		if isMatched {
			return true
		}
	}
	return false
}

// isHolding checks if the intended trading ticker is currently beign held.
func (t *trader) isHolding(r *runner.Runner) bool {
	valid := false
	if r.GetMarketType() == runner.Cash {
		t.binSpotBalances.Range(func(key, value interface{}) bool {
			valid = strings.ToUpper(key.(string)+t.quoteCurrency) == strings.ToUpper(r.GetName())
			return !valid
		})
	}
	return valid
}

// initialCheck validates if a runner is currently allowed to be traded before placing an order to the markets.
func (t *trader) initialChecks(r *runner.Runner) bool {
	t.Lock()
	defer t.Unlock()

	if !t.isAllowedMarkets(r) {
		return false
	}
	if !t.isAllowedPatterns(r) {
		return false
	}
	if t.isHolding(r) {
		return false
	}
	if val, ok := t.binSpotBalances.Load(t.quoteCurrency); !ok || !big.NewFromString(val.(bn.Balance).Free).GT(t.minBalance) {
		return false
	}
	return true
}

// stopLoss calculates the stop loss price given a price.
func (t *trader) stopLoss(price, tradingSide string) string {
	quantity := t.minBalance.Div(big.NewFromString(price))
	if strings.ToUpper(tradingSide) == "BUY" {
		return t.minBalance.Mul(big.NewDecimal(1.0).Sub(t.lossTolerance)).Div(quantity).FormattedString(8)
	}
	return t.minBalance.Mul(big.NewDecimal(1.0).Add(t.lossTolerance)).Div(quantity).FormattedString(8)
}

// processEvaluatorRequest take care of the request from the evaluator, which will place trades on successfully evaluated signals.
func (t *trader) processEvaluatorRequest(msg *message) {
	if msg.request.what.runner == nil || msg.request.what.signal == nil {
		return
	}
	if !t.initialChecks(msg.request.what.runner) {
		return
	}
	r, s := msg.request.what.runner, msg.request.what.signal
	if r.GetMarketType() == runner.Cash {
		// TODO: think of the price and position sizing
		// currently no position sizing as it uses predetermined price of minBalance
		price := "30000"

		// place a LIMIT order always.
		order, err := t.provider.binSpot.NewCreateOrderService().
			// set the symbol from the runner
			Symbol(r.GetName()).
			// set the trading side from the signal
			Side(bn.SideType(s.TradingSide())).
			// order type is always stop loss
			Type(bn.OrderTypeLimit).
			// timeInFore is always good-to-candle
			TimeInForce(bn.TimeInForceTypeGTC).
			// set the limit price
			Price(price).
			// quantity deduced from price and minimum trading balance for a position
			Quantity(t.minBalance.Div(big.NewFromString(price)).FormattedString(8)).
			// place the order
			Do(context.Background())
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
			return
		}
		// after placing an order, the trader goes on to monitor the trade
		fmt.Println(fmt.Sprintf("%+v", *order))
	}
}

// binSpotUserDataStreaming manages all account changing events from trading activities on cash account.
func (t *trader) binSpotUserDataStreaming() {
	isError, isInit := false, true
	dataHandler := func(e *bn.WsUserDataEvent) {
		switch e.Event {
		case bn.UserDataEventTypeOutboundAccountPosition:
			for _, a := range e.AccountUpdate {
				t.updatebinSpotBalances(bn.Balance{
					Asset:  a.Asset,
					Free:   a.Free,
					Locked: a.Locked,
				})
			}
		case bn.UserDataEventTypeBalanceUpdate:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %+v", *e)))
		case bn.UserDataEventTypeExecutionReport:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %+v", *e)))
		case bn.UserDataEventTypeListStatus:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %+v", *e)))
		default:
		}
	}
	errorHandler := func(err error) { t.logger.Error.Println(t.newLog(err.Error())); isError = true }
	for isInit || isError {
		done, _, err := bn.WsUserDataServe(t.binSpotListenKey, dataHandler, errorHandler)
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		isError, isInit = false, false
		<-done
	}
}

// binFutuUserDataStreaming manages all account changing events from trading activities on futures account.
func (t *trader) binFutuUserDataStreaming() {
	isError, isInit := false, true
	dataHandler := func(e *bnf.WsUserDataEvent) { t.logger.Info.Println(t.newLog(fmt.Sprintf("futu %+v", *e))) }
	errorHandler := func(err error) { t.logger.Error.Println(t.newLog(err.Error())); isError = true }
	for isInit || isError {
		done, _, err := bnf.WsUserDataServe(t.binFutuListenKey, dataHandler, errorHandler)
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		isError, isInit = false, false
		<-done
	}
}

// registerStreamingChannel registers the runners with the streamer in order to
// recevie and consume candles broadcasted by data providor.
func (t *trader) registerStreamingChannel(mem tdmember) bool {
	doneStreamingRegister := false
	var maxTries int
	for !doneStreamingRegister && maxTries <= 3 {
		resC := make(chan *payload)
		t.communicator.trader2Streamer <- t.communicator.newMessage(mem.runner, nil, mem.channels, nil, resC)
		doneStreamingRegister = (<-resC).what.dynamic.(bool)
		maxTries++
	}
	return doneStreamingRegister
}

// generates a new log with the format for the watcher
func (t *trader) newLog(message string) string {
	return fmt.Sprintf("[trader] %s", message)
}
