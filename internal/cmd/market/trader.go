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
	binFutuOrders   *sync.Map
	binSpotBalances *sync.Map
	binFutuBalances *sync.Map

	// trader configurations
	quoteCurrency   string
	allowedPatterns []*regexp2.Regexp
	allowedMarkets  []runner.MarketType

	maxLeverage       big.Decimal
	minBalance        big.Decimal
	maxPositions      big.Decimal
	maxWaitToFill     time.Duration
	maxLossPerTrade   big.Decimal
	minProfitPerTrade big.Decimal

	// risk controlling factors
	profitMargin  big.Decimal
	lossTolerance big.Decimal

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

// newTrader returns a trader, meant to be called by the MarketStruct only once.
func newTrader(participants *sharedParticipants, configs *config.Configs) (*trader, error) {
	if configs == nil || participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants or configs")
	}
	t := &trader{
		connected:       false,
		binSpotBalances: &sync.Map{},
		binSpotOrders:   &sync.Map{},
		binFutuBalances: &sync.Map{},
		binFutuOrders:   &sync.Map{},

		minBalance:        big.ZERO,
		maxPositions:      big.ZERO,
		maxLossPerTrade:   big.NewFromString("Inf"),
		minProfitPerTrade: big.NewFromString("Inf"),
		maxLeverage:       big.NewDecimal(1.0),
		maxWaitToFill:     time.Second * 60,
		lossTolerance:     big.NewDecimal(0.01),
		profitMargin:      big.NewDecimal(0.02),

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}
	var err error
	if err = t.updateConfigs(configs); err != nil {
	}
	if t.binSpotBalances, err = t.provider.fetchBinSpotBalances(t.quoteCurrency); err != nil {
		return nil, err
	}
	if t.binFutuBalances, err = t.provider.fetchBinFutuBalances(t.quoteCurrency); err != nil {
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
	if configs.Market.Trader.MaxLeverage > 0 {
		t.maxLeverage = big.NewDecimal(configs.Market.Trader.MaxLeverage)
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
	if configs.Market.Trader.MaxLossPerTrade > 0 {
		t.maxLossPerTrade = big.NewDecimal(configs.Market.Trader.MaxLossPerTrade)
	}
	if configs.Market.Trader.MinProfitPerTrade > 0 {
		t.minProfitPerTrade = big.NewDecimal(configs.Market.Trader.MinProfitPerTrade)
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

// binSpotGetBalances returns assets which are currently available on Binance spot account.
func (t *trader) binSpotGetBalances() []bn.Balance {
	out := []bn.Balance{}
	t.binSpotBalances.Range(func(key, value interface{}) bool {
		out = append(out, value.(bn.Balance))
		return true
	})
	return out
}

// binFutuGetBalances returns assets which are currently available on Binance futures account.
func (t *trader) binFutuGetBalances() []bnf.Balance {
	out := []bnf.Balance{}
	t.binFutuBalances.Range(func(key, value interface{}) bool {
		out = append(out, value.(bnf.Balance))
		return true
	})
	return out
}

// binSpotUpdateBalances removes or adds assset to holdings on trader binSpotBalances.
// It is called on initialization and upon receving data from websocket for UserData event.
func (t *trader) binSpotUpdateBalances(bl bn.Balance) {
	t.Lock()
	defer t.Unlock()
	if big.NewFromString(bl.Free).EQ(big.ZERO) {
		t.binSpotBalances.Delete(bl.Asset + t.quoteCurrency)
		return
	}
	t.binSpotBalances.Store(bl.Asset+t.quoteCurrency, bl)
}

// binFutuUpdateBalances removes or adds assset to holdings on trader binFutuBalances.
// It is called on initialization and upon receving data from websocket for UserData event.
func (t *trader) binFutuUpdateBalances(bl bnf.Balance) {
	t.Lock()
	defer t.Unlock()
	if big.NewFromString(bl.Balance).EQ(big.ZERO) {
		t.binFutuBalances.Delete(bl.Asset + t.quoteCurrency)
		return
	}
	t.binFutuBalances.Store(bl.Asset+t.quoteCurrency, bl)
}

// isAllowedMarkets checks if the given runner is allowed to trade based on its targered markets,
// which are SPOT or FUTURES.
func (t *trader) isAllowedMarkets(r *runner.Runner) bool {
	for _, m := range t.allowedMarkets {
		if m == r.GetMarketType() {
			return true
		}
	}
	return false
}

// isAllowedPatterns checks if the given runner is allowed to trade based on its ticker name.
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

// isHolding checks if the given runner which is intended to be traded is currently being held.
func (t *trader) isHolding(r *runner.Runner) bool {
	valid := false
	switch r.GetMarketType() {
	case runner.Cash:
		t.binSpotBalances.Range(func(key, val interface{}) bool {
			valid = (strings.ToUpper(val.(bn.Balance).Asset+t.quoteCurrency) == strings.ToUpper(r.GetName()))
			return !valid
		})
		return valid
	case runner.Futures:
		t.binFutuBalances.Range(func(key, val interface{}) bool {
			valid = strings.ToUpper(val.(bnf.Balance).Asset+t.quoteCurrency) == strings.ToUpper(r.GetName())
			return !valid
		})
		return valid
	default:
		return false
	}
}

// isOrdering checks if there are outstanding orders on the runner.
func (t *trader) isOrdering(r *runner.Runner) bool {
	switch r.GetMarketType() {
	case runner.Cash:
		orders, err := t.provider.binSpot.NewListOpenOrdersService().Symbol(r.GetName()).Do(context.Background())
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		return len(orders) > 0 || err != nil
	case runner.Futures:
		orders, err := t.provider.binFutu.NewListOpenOrdersService().Symbol(r.GetName()).Do(context.Background())
		if err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		return len(orders) > 0 || err != nil
	default:
		return true
	}
}

// isEnoughBalance checks if balance is enough to open a trade
// based on the minimum balance requirement.
func (t *trader) isEnoughBalance(r *runner.Runner) bool {
	quote := t.quoteCurrency + t.quoteCurrency
	switch r.GetMarketType() {
	case runner.Cash:
		if val, ok := t.binSpotBalances.Load(quote); ok && big.NewFromString(val.(bn.Balance).Free).GT(t.minBalance) {
			return true
		}
	case runner.Futures:
		if val, ok := t.binFutuBalances.Load(quote); ok && big.NewFromString(val.(bnf.Balance).Balance).GT(t.minBalance) {
			return true
		}
	default:
		return false
	}
	return false
}

// initialCheck validates if a runner is currently allowed to be traded before placing an order to the markets.
// the checks include isAllowedMarkets, isAllowedPatterns, not isHolding, not isOrdering and the account has
// enough balance to open a trade.
func (t *trader) initialChecks(r *runner.Runner) bool {
	if !t.isAllowedMarkets(r) || !t.isAllowedPatterns(r) {
		return false
	}
	if t.isHolding(r) || t.isOrdering(r) {
		return false
	}
	if !t.isEnoughBalance(r) {
		return false
	}
	return true
}

// shouldClose checks if the current price exceeds the limit given by the loss tolerance
// or the current price surpasses the profit margin. It also returns current PNL and PNL in dollar.
func (t *trader) shouldClose(st *setup, currentPrice big.Decimal) (bool, big.Decimal, big.Decimal) {
	if currentPrice.EQ(big.ZERO) || st.avgFilledPrice.EQ(big.ZERO) {
		return true, big.ZERO, big.ZERO
	}
	isFutures := st.runner.GetMarketType() == runner.Futures
	isCash := st.runner.GetMarketType() == runner.Cash
	if currentPrice.LTE(st.avgFilledPrice) {
		pnl := st.avgFilledPrice.Sub(currentPrice).Div(currentPrice)
		pnlDollar := pnl.Mul(st.accFilledQtity.Mul(st.avgFilledPrice))
		if isFutures {
			pnlDollar = pnlDollar.Mul(t.maxLeverage)
		}
		if strings.ToUpper(st.orderSide) == "BUY" {
			return (isCash && pnl.GTE(t.lossTolerance)) || (isFutures && pnlDollar.GTE(t.maxLossPerTrade)), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
		} else {
			return (isCash && pnl.GTE(t.profitMargin)) || (isFutures && pnlDollar.GTE(t.minProfitPerTrade)), pnl, pnlDollar
		}
	} else {
		pnl := currentPrice.Sub(st.avgFilledPrice).Div(st.avgFilledPrice)
		pnlDollar := pnl.Mul(st.accFilledQtity.Mul(st.avgFilledPrice))
		if isFutures {
			pnlDollar = pnlDollar.Mul(t.maxLeverage)
		}
		if strings.ToUpper(st.orderSide) == "SELL" {
			return (isCash && pnl.GTE(t.lossTolerance)) || (isFutures && pnlDollar.GTE(t.maxLossPerTrade)), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
		} else {
			return (isCash && pnl.GTE(t.profitMargin)) || (isFutures && pnlDollar.GTE(t.maxLossPerTrade)), pnl, pnlDollar
		}
	}
	return false, big.ZERO, big.ZERO
}

// placeMarketOrder places a market order on a given runner.
// this can be used in different scenarios, such as close positions.
func (t *trader) placeMarketOrder(r *runner.Runner, side, quantity string) error {
	switch r.GetMarketType() {
	case runner.Cash:
		_, err := t.provider.binSpot.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bn.SideType(side)).
			Type(bn.OrderTypeMarket).
			Quantity(quantity).
			Do(context.Background())
		return err
	case runner.Futures:
		_, err := t.provider.binFutu.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bnf.SideType(side)).
			Type(bnf.OrderTypeMarket).
			Quantity(quantity).
			ReduceOnly(true).
			Do(context.Background())
		return err
	default:
		return errors.New("unknow market")
	}
}

// cancleOpenOrder cancels an outstanding order, given the orderID.
// the bot has to manage all orders it's initialized. In some case,
// it needs to cancel outstanding orders before completing trades.
func (t *trader) cancleOpenOrder(r *runner.Runner, oid int64) error {
	switch r.GetMarketType() {
	case runner.Cash:
		_, err := t.provider.binSpot.NewCancelOrderService().
			Symbol(r.GetName()).
			OrderID(oid).
			Do(context.Background())
		return err
	case runner.Futures:
		_, err := t.provider.binFutu.NewCancelOrderService().
			Symbol(r.GetName()).
			OrderID(oid).
			Do(context.Background())
		return err
	default:
		return nil
	}
}

// processEvaluatorRequest take care of requests from the evaluator,
// which places trades if the given runner passed the initialChecks method and
// the given signal gives a valid limit price.
func (t *trader) processEvaluatorRequest(msg *message) error {
	if msg.request.what.runner == nil || msg.request.what.signal == nil {
		return errors.New("missing runner or signal")
	}
	if !t.initialChecks(msg.request.what.runner) {
		return nil
	}
	r, s := msg.request.what.runner, msg.request.what.signal
	price, ok := s.TradeExecutionPrice(r)
	if !ok {
		t.logger.Warning.Println(t.newLog("cannot find a price to place trade"))
		return nil
	}
	switch r.GetMarketType() {
	case runner.Cash:
		pricePrecision, quantityPrecision, err := t.provider.fetchBinSpotExchangeInfo(r.GetName())
		if err != nil {
			return err
		}
		o, err := t.provider.binSpot.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bn.SideType(s.OpenTradingSide())).
			Type(bn.OrderTypeLimit).
			TimeInForce(bn.TimeInForceTypeGTC).
			Price(price.FormattedString(pricePrecision)).
			Quantity(t.minBalance.Div(price).FormattedString(quantityPrecision)).
			Do(context.Background())
		if err != nil {
			return err
		}
		st := newSetup(r, s, big.ONE, o)
		t.binSpotOrders.Store(r.GetName(), st)
		go t.monitorBinSpotTrade(st)
	case runner.Futures:
		pricePrecision, quantityPrecision, err := t.provider.fetchBinFutuExchangeInfo(r.GetName())
		if err != nil {
			return err
		}
		o, err := t.provider.binFutu.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bnf.SideType(s.OpenTradingSide())).
			Type(bnf.OrderTypeLimit).
			TimeInForce(bnf.TimeInForceTypeGTC).
			Price(price.FormattedString(pricePrecision)).
			Quantity(t.minBalance.Mul(t.maxLeverage).Div(price).FormattedString(quantityPrecision)).
			Do(context.Background())
		if err != nil {
			return err
		}
		st := newSetup(r, s, t.maxLeverage, o)
		t.binFutuOrders.Store(r.GetName(), st)
		go t.monitorBinFutuTrade(st)
	default:
		return nil
	}
	return nil
}

// monitorBinSpotTrade monitors the trade after an order is placed successfully.
func (t *trader) monitorBinSpotTrade(st *setup) {
	defer t.report(st)

	// Waiting for the order to be (partially) filled
	nw := time.Now()
	for st.orderStatus != "FILLED" && st.orderStatus != "PARTIALLY_FILLED" {
		time.Sleep(time.Millisecond * 100)
		if st.orderStatus == "CANCELED" ||
			st.orderStatus == "REJECTED" ||
			st.orderStatus == "EXPIRED" ||
			st.orderStatus == "PENDING_CANCEL" {
			return
		}
		if time.Now().Sub(nw) > t.maxWaitToFill {
			if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
			}
		}
	}

	// Upto this point, the order has been filled,
	// you basically could place a stop order or an OCO order to control your risk and return
	// or listen to streaming channels, depth or trade event, to manage the trade yourself.
	for st.avgFilledPrice.EQ(big.ZERO) {
		time.Sleep(time.Second)
	}
	st.channels = &streamingChannels{depth: make(chan interface{}, 20)}
	for !t.registerStreamingChannel(*st) {
		t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to register streaming service", st.runner.GetName())))
	}
	for msg := range st.channels.depth {
		bestPrice := tax.BinanceSpotBestBidAskFromDepth(msg.(*bn.WsPartialDepthEvent)).L1ForClosingTrade(st.orderSide)
		ok, pnl, _ := t.shouldClose(st, bestPrice.Price)
		if !ok {
			continue
		}
		st.pnl = pnl
		val, ok := t.binSpotBalances.Load(st.runner.GetName())
		if !ok {
			t.logger.Error.Println(t.newLog("there is no asset to trade"))
			break
		}
		if err := t.placeMarketOrder(st.runner, st.signal.CloseTradingSide(), val.(bn.Balance).Free); err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
			break
		}
	}
	for !t.registerStreamingChannel(*st) {
		t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to deregister streaming service", st.runner.GetName())))
	}

	// Upto this point, the trade should be close, and converted back to
	// the quote currency, which should be in USDT, and report PNLs.
	// The outstanding portion or the order should be canceled.
	if !t.isOrdering(st.runner) {
		return
	}
	if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
		t.logger.Error.Println(t.newLog(err.Error()))
	}
}

// monitorBinFutuTrade monitors the trade after an order is placed successfully.
func (t *trader) monitorBinFutuTrade(st *setup) {
	defer t.report(st)

	// Waiting for the order to be (partially) filled
	nw := time.Now()
	for st.orderStatus != "FILLED" && st.orderStatus != "PARTIALLY_FILLED" {
		time.Sleep(time.Millisecond * 100)
		if st.orderStatus == "CANCELED" ||
			st.orderStatus == "REJECTED" ||
			st.orderStatus == "EXPIRED" ||
			st.orderStatus == "PENDING_CANCEL" {
			return
		}
		if time.Now().Sub(nw) > t.maxWaitToFill {
			if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
			}
		}
	}

	// Upto this point, the order has been filled,
	// you basically could place a stop order or an OCO order to control your risk and return
	// or listen to streaming channels, depth or trade event, to manage the trade yourself.
	for st.avgFilledPrice.EQ(big.ZERO) {
		time.Sleep(time.Second)
	}
	st.channels = &streamingChannels{depth: make(chan interface{}, 20)}
	for !t.registerStreamingChannel(*st) {
		t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to register streaming service", st.runner.GetName())))
	}
	for msg := range st.channels.depth {
		bestPrice := tax.BinanceFutuBestBidAskFromDepth(msg.(*bnf.WsDepthEvent)).L1ForClosingTrade(st.orderSide)
		ok, pnl, _ := t.shouldClose(st, bestPrice.Price)
		if !ok {
			continue
		}
		st.pnl = pnl
		val, ok := t.binFutuBalances.Load(st.runner.GetName())
		if !ok {
			t.logger.Error.Println(t.newLog("there is no asset to trade"))
			break
		}
		if err := t.placeMarketOrder(st.runner, st.signal.CloseTradingSide(), val.(bnf.Balance).Balance); err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
			break
		}
	}
	for !t.registerStreamingChannel(*st) {
		t.logger.Error.Println(t.newLog(fmt.Sprintf("%+s, failed to deregister streaming service", st.runner.GetName())))
	}

	//// Upto this point, the trade should be close, and converted back to
	//// the quote currency, which should be in USDT, and report PNLs.
	//// The outstanding portion or the order should be canceled.
	if !t.isOrdering(st.runner) {
		return
	}
	if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
		t.logger.Error.Println(t.newLog(err.Error()))
	}
}

// report reports a trade to user after closing it. It could also store the trade
// to some persistent storage for later evaluation.
func (t *trader) report(st *setup) {
	t.logger.Info.Println(t.newLog(st.description()))
	t.communicator.trader2Notifier <- t.communicator.newMessage(st.runner, st.signal, nil, st.description(), nil)
}

// binSpotUserDataStreaming manages all account changing events from trading activities on cash account.
func (t *trader) binSpotUserDataStreaming() {
	isError, isInit := false, true
	dataHandler := func(e *bn.WsUserDataEvent) {
		switch e.Event {
		case bn.UserDataEventTypeOutboundAccountPosition:
			for _, a := range e.AccountUpdate {
				t.binSpotUpdateBalances(bn.Balance{
					Asset:  a.Asset,
					Free:   a.Free,
					Locked: a.Locked,
				})
			}
		case bn.UserDataEventTypeBalanceUpdate:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash, status name: %s, %+v", string(bn.UserDataEventTypeBalanceUpdate), *e)))
		case bn.UserDataEventTypeExecutionReport:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash, status name: %s, %+v", string(bn.UserDataEventTypeExecutionReport), *e)))
			if val, ok := t.binSpotOrders.Load(e.OrderUpdate.Symbol); ok && e.OrderUpdate.Id == val.(*setup).orderID {
				val.(*setup).binSpotUpdateTrade(e.OrderUpdate)
			}
		case bn.UserDataEventTypeListStatus:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash, status name: %s, %+v", string(bn.UserDataEventTypeListStatus), *e)))
		default:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash, status name: %s, %+v", "unknow", *e)))
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
	dataHandler := func(e *bnf.WsUserDataEvent) {
		switch e.Event {
		case bnf.UserDataEventTypeAccountUpdate:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, status name: %s, %+v", string(bnf.UserDataEventTypeAccountUpdate), *e)))
			for _, b := range e.AccountUpdate.Balances {
				t.binFutuUpdateBalances(bnf.Balance{
					Asset:              b.Asset,
					Balance:            b.Balance,
					CrossWalletBalance: b.CrossWalletBalance,
				})
			}
		case bnf.UserDataEventTypeOrderTradeUpdate:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, status name: %s, %+v", string(bnf.UserDataEventTypeOrderTradeUpdate), *e)))
			if val, ok := t.binFutuOrders.Load(e.OrderTradeUpdate.Symbol); ok && e.OrderTradeUpdate.ID == val.(*setup).orderID {
				val.(*setup).binFutuUpdateTrade(e.OrderTradeUpdate)
			}
		case bnf.UserDataEventTypeMarginCall:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, status name: %s, %+v", string(bnf.UserDataEventTypeMarginCall), *e)))
		default:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, status name: %s, %+v", "unknow", *e)))
		}
	}
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

// registerStreamingChannel registers or deregisters a runner to the streamer in order to
// receive candle or depth broadcasted by data providor.
func (t *trader) registerStreamingChannel(m setup) bool {
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
