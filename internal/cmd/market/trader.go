package market

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/dlclark/regexp2"
	"github.com/sdcoffey/big"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type trader struct {
	sync.Mutex
	connected        bool
	binFutuListenKey string
	binSpotListenKey string

	binSpotOrders   *sync.Map
	binSpotBalances *sync.Map

	// trader configurations
	quoteCurrency   string
	allowedPatterns []*regexp2.Regexp
	allowedMarkets  []runner.MarketType

	minLeverage   big.Decimal
	maxLeverage   big.Decimal
	minBalance    big.Decimal
	maxPositions  big.Decimal
	maxWaitToFill time.Duration

	// risk controlling factors
	profitMargin  big.Decimal
	lossTolerance big.Decimal

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type tdmember struct {
	runner   *runner.Runner
	signal   *strategy.Signal
	channels *streamingChannels

	orderStatus string
}

func newTrader(participants *sharedParticipants, configs *config.Configs) (*trader, error) {
	if configs == nil || participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants or configs")
	}
	t := &trader{
		connected:       false,
		binSpotBalances: &sync.Map{},
		binSpotOrders:   &sync.Map{},

		minBalance:    big.ZERO,
		maxPositions:  big.ZERO,
		minLeverage:   big.NewDecimal(1.0),
		maxLeverage:   big.NewDecimal(1.0),
		maxWaitToFill: time.Second * 60,
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
	if configs.Market.Trader.MaxWaitToFill > 0 {
		t.maxWaitToFill = time.Duration(configs.Market.Trader.MaxWaitToFill) * time.Second
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
			err := t.processEvaluatorRequest(msg)
			if err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
			}
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
	t.binSpotBalances.Store(bl.Asset+t.quoteCurrency, bl)
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

func (t *trader) isOrdering(r *runner.Runner) bool {
	// this request costs 3WI as it hits a binance API
	if r.GetMarketType() == runner.Cash {
		orders, err := t.provider.binSpot.NewListOpenOrdersService().Symbol(r.GetName()).Do(context.Background())
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		return len(orders) > 0 || err != nil
	}
	return true
}

// initialCheck validates if a runner is currently allowed to be traded before placing an order to the markets.
func (t *trader) initialChecks(r *runner.Runner) bool {
	//t.Lock()
	//defer t.Unlock()
	if !t.isAllowedMarkets(r) || !t.isAllowedPatterns(r) {
		return false
	}
	if t.isHolding(r) || t.isOrdering(r) {
		return false
	}
	if val, ok := t.binSpotBalances.Load(t.quoteCurrency + t.quoteCurrency); !ok || !big.NewFromString(val.(bn.Balance).Free).GT(t.minBalance) {
		return false
	}
	return true
}

// shouldStop checks if the current price exceeds the limit given by the loss tolerance
// or the current price surpass the profit margin.
func (t *trader) shouldClose(orderPrice, currentPrice big.Decimal, tradingSide string) (bool, big.Decimal) {
	if currentPrice.LTE(orderPrice) {
		pnl := orderPrice.Sub(currentPrice).Div(currentPrice)
		if strings.ToUpper(tradingSide) == "BUY" {
			return pnl.GTE(t.lossTolerance), pnl.Mul(big.NewFromString("-1"))
		} else {
			return pnl.GTE(t.profitMargin), pnl
		}
	} else {
		pnl := currentPrice.Sub(orderPrice).Div(orderPrice)
		if strings.ToUpper(tradingSide) == "SELL" {
			return pnl.GTE(t.lossTolerance), pnl.Mul(big.NewFromString("-1"))
		} else {
			return pnl.GTE(t.profitMargin), pnl
		}
	}
	return false, big.ZERO
}

// placeMarketOrder places an order on a given runner.
func (t *trader) placeMarketOrder(r *runner.Runner, side, quantity string) error {
	if r.GetMarketType() == runner.Cash {
		_, err := t.provider.binSpot.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bn.SideType(side)).
			Type(bn.OrderTypeMarket).
			Quantity(quantity).
			Do(context.Background())
		return err
	}
	return nil
}

// cancleOpenOrder cancels an outstanding order, given its orderID.
func (t *trader) cancleOpenOrder(r *runner.Runner, oid int64) error {
	if r.GetMarketType() == runner.Cash {
		_, err := t.provider.binSpot.NewCancelOrderService().
			Symbol(r.GetName()).
			OrderID(oid).
			Do(context.Background())
		return err
	}
	return nil
}

// processEvaluatorRequest take care of the request from the evaluator, which will place trades on successfully evaluated signals.
func (t *trader) processEvaluatorRequest(msg *message) error {
	if msg.request.what.runner == nil || msg.request.what.signal == nil {
		return errors.New("missing runner or signal")
	}
	if !t.initialChecks(msg.request.what.runner) {
		return nil
	}
	r, s := msg.request.what.runner, msg.request.what.signal
	mem := &tdmember{runner: r, signal: s}
	switch r.GetMarketType() {
	case runner.Cash:
		// TODO: think of the price and position sizing
		// currently no position sizing as it uses predetermined price of minBalance
		price, ok := s.TradeExcutionPrice(r)
		if !ok {
			t.logger.Warning.Println(t.newLog("cannot find a price to place trade"))
			return nil
		}
		// place a LIMIT order always.
		order, err := t.provider.binSpot.NewCreateOrderService().
			// set the symbol from the runner
			Symbol(r.GetName()).
			// set the trading side from the signal
			Side(bn.SideType(s.OpenTradingSide())).
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
			return err
		}
		orderC := make(chan bn.WsOrderUpdate, 2)
		go func() {
			for msg := range orderC {
				if msg.Id != order.OrderID {
					continue
				}
				mem.orderStatus = msg.Status
				switch msg.Status {
				case "NEW":
					continue
				case "TRADE":
					fallthrough
				case "EXPIRED":
					fallthrough
				case "REJECTED":
					fallthrough
				case "CANCELED":
					t.binSpotOrders.Delete(r.GetName())
					close(orderC)
				default:
					t.logger.Warning.Println(t.newLog(fmt.Sprintf("unknow status: %s", msg.Status)))
					continue
				}
			}
		}()
		t.binSpotOrders.Store(r.GetName(), orderC)
		go t.monitorBinSpotTrade(mem, order)
	case runner.Futures:
		return errors.New("futures market is not currently supported to trade yet")
	default:
		return errors.New("unknow market type")
	}
	return nil
}

// monitorBinSpotTrade monitors the trade after an order is placed.
func (t *trader) monitorBinSpotTrade(m *tdmember, o *bn.CreateOrderResponse) {
	nw := time.Now()
	// Waiting for the order to be (partially) filled
	for m.orderStatus != "TRADE" {
		time.Sleep(time.Second)
		if m.orderStatus == "CANCELED" {
			return
		}
		if time.Now().Sub(nw) > t.maxWaitToFill {
			err := t.cancleOpenOrder(m.runner, o.OrderID)
			if err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
				continue
			}
			return
		}
	}
	// Upto this point, the order has been filled,
	// you basically could place a stop order or an OCO order to control your risk and return
	// or listen to streaming channels, depth or trade event, to manage the trade yourself.
	m.channels = &streamingChannels{depth: make(chan interface{}, 20)}
	for !t.registerStreamingChannel(*m) {
		t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to register streaming service", m.runner.GetName())))
	}
	for msg := range m.channels.depth {
		bestPrice := tax.BinanceSpotBestBidAskFromDepth(msg.(bn.WsPartialDepthEvent)).L1ForClosingTrade(string(o.Side))
		if ok, _ := t.shouldClose(big.NewFromString(o.Price), bestPrice.Price, string(o.Side)); !ok {
			continue
		}
		val, ok := t.binSpotBalances.Load(m.runner.GetName())
		if !ok {
			t.logger.Error.Println(t.newLog("there is no asset to trade"))
			for !t.registerStreamingChannel(*m) {
				t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to deregister streaming service", m.runner.GetName())))
			}
		}
		if err := t.placeMarketOrder(m.runner, m.signal.CloseTradingSide(), val.(bn.Balance).Free); err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
			for !t.registerStreamingChannel(*m) {
				t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to deregister streaming service", m.runner.GetName())))
			}
		}
	}
	// Upto this point, the trade should be close, and converted back to
	// the quote currency, which should be in USDT, and report PNLs.
	// The outstanding portion or the order should be canceled.
	if !t.isOrdering(m.runner) {
		return
	}
	if err := t.cancleOpenOrder(m.runner, o.OrderID); err != nil {
		t.logger.Error.Println(t.newLog(err.Error()))
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
			if val, ok := t.binSpotOrders.Load(e.OrderUpdate.Symbol); ok {
				val.(chan bn.WsOrderUpdate) <- e.OrderUpdate
			}
		case bn.UserDataEventTypeListStatus:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %+v", *e)))
		default:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %+v", *e)))
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
func (t *trader) registerStreamingChannel(m tdmember) bool {
	doneStreamingRegister := false
	var maxTries int
	for !doneStreamingRegister && maxTries <= 3 {
		resC := make(chan *payload)
		t.communicator.trader2Streamer <- t.communicator.newMessage(m.runner, nil, m.channels, nil, resC)
		doneStreamingRegister = (<-resC).what.dynamic.(bool)
		maxTries++
	}
	return doneStreamingRegister
}

// generates a new log with the format for the watcher
func (t *trader) newLog(message string) string {
	return fmt.Sprintf("[trader] %s", message)
}
