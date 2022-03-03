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

	db "follow.markets/internal/pkg/database"
	"follow.markets/internal/pkg/runner"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
)

type trader struct {
	sync.Mutex
	connected       bool
	isTradeDisabled bool

	binFutuListenKey string
	binSpotListenKey string
	binTrades        *sync.Map
	binSpotBalances  *sync.Map
	binFutuBalances  *sync.Map

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
		isTradeDisabled: true,

		binTrades:       &sync.Map{},
		binSpotBalances: &sync.Map{},
		binFutuBalances: &sync.Map{},

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
	if err = t.binSpotUpdateBalances(); err != nil {
		return nil, err
	}
	if err = t.binFutuUpdateBalances(); err != nil {
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
	t.Lock()
	defer t.Unlock()

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
	t.isTradeDisabled = !configs.Market.Trader.Allowed
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
		for msg := range t.communicator.notifier2Trader {
			go t.processNotifierRequest(msg)
		}
	}()
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

// this method processes request from notifier. it mainly handles
// request from user via the notifier.
func (t *trader) processNotifierRequest(msg *message) {
	var rs string
	balances := make(map[string]string)
	switch msg.request.what.dynamic.(string) {
	case TRADER_MESSAGE_IS_TRADE_ENABLED:
		rs = TRADER_MESSAGE_IS_TRADE_ENABLED + " ➡️  NO."
		if !t.isTradeDisabled {
			rs = TRADER_MESSAGE_IS_TRADE_ENABLED + " ➡️  YES."
		}
	case TRADER_MESSAGE_ENABLE_TRADE:
		t.isTradeDisabled = false
		rs = TRADER_MESSAGE_ENABLE_TRADE + TRADER_MESSAGE_ENABLE_TRADE_COMPLETED
	case TRADER_MESSAGE_DISABLE_TRADE:
		t.isTradeDisabled = true
		rs = TRADER_MESSAGE_DISABLE_TRADE + TRADER_MESSAGE_DISABLE_TRADE_COMPLETED
	case TRADER_MESSAGE_SPOT_BALANCES:
		t.binSpotBalances.Range(func(key, val interface{}) bool {
			balances[val.(bn.Balance).Asset] = val.(bn.Balance).Free
			return true
		})
		rs = TRADER_MESSAGE_SPOT_BALANCES + fmt.Sprintf(" ➡️  %+v.", balances)
	case TRADER_MESSAGE_FUTU_BALANCES:
		t.binFutuBalances.Range(func(key, val interface{}) bool {
			balances[val.(bnf.Balance).Asset] = val.(bnf.Balance).Balance
			return true
		})
		rs = TRADER_MESSAGE_FUTU_BALANCES + fmt.Sprintf(" ➡️  %+v.", balances)
	case TRADER_MESSAGE_UPDATE_BALANCES:
		rs = TRADER_MESSAGE_UPDATE_BALANCES + " ⬇️ \n"
		if err := t.binSpotUpdateBalances(); err != nil {
			rs += "FAILED TO UPDATE SPOT BALANCE,"
		} else {
			rs += "SPOT BALANCE UPDATED,"
		}
		if err := t.binFutuUpdateBalances(); err != nil {
			rs += "\nFAILED TO UPDATE FUTIRES BALANCE."
		} else {
			rs += "\nFUTURES BALANCE UPDATED."
		}
	default:
		rs = "UNKNOWN REQUEST"
	}
	if msg.response != nil {
		msg.response <- t.communicator.newPayload(nil, nil, nil, rs).addRequestID(&msg.request.requestID).addResponseID()
		close(msg.response)
	}
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
func (t *trader) binSpotUpdateBalances() error {
	t.Lock()
	defer t.Unlock()
	var err error
	t.binSpotBalances, err = t.provider.fetchBinSpotBalances(t.quoteCurrency)
	return err
}

// binFutuUpdateBalances removes or adds assset to holdings on trader binFutuBalances.
// It is called on initialization and upon receving data from websocket for UserData event.
func (t *trader) binFutuUpdateBalances() error {
	t.Lock()
	defer t.Unlock()
	var err error
	t.binFutuBalances, err = t.provider.fetchBinFutuBalances(t.quoteCurrency)
	return err
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
	if t.isTradeDisabled {
		return false
	}
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
	//isFutures := st.runner.GetMarketType() == runner.Futures
	//isCash := st.runner.GetMarketType() == runner.Cash
	isBuy := strings.ToUpper(st.orderSide) == "BUY"
	if currentPrice.LTE(st.avgFilledPrice) {
		pnl := st.avgFilledPrice.Sub(currentPrice).Div(currentPrice)
		pnlDollar := pnl.Mul(st.accFilledQtity.Mul(st.avgFilledPrice)).Mul(st.usedLeverage)
		if isBuy {
			//return (isCash && pnl.GTE(t.lossTolerance)) || (isFutures && pnlDollar.GTE(t.maxLossPerTrade)), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
			return pnlDollar.GTE(t.maxLossPerTrade), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
		}
		//return (isCash && pnl.GTE(t.profitMargin)) || (isFutures && pnlDollar.GTE(t.minProfitPerTrade)), pnl, pnlDollar
		return pnlDollar.GTE(t.minProfitPerTrade), pnl, pnlDollar
	} else {
		pnl := currentPrice.Sub(st.avgFilledPrice).Div(st.avgFilledPrice)
		pnlDollar := pnl.Mul(st.accFilledQtity.Mul(st.avgFilledPrice)).Mul(st.usedLeverage)
		if !isBuy {
			//return (isCash && pnl.GTE(t.lossTolerance)) || (isFutures && pnlDollar.GTE(t.maxLossPerTrade)), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
			return pnlDollar.GTE(t.maxLossPerTrade), pnl.Mul(big.NewFromString("-1")), pnlDollar.Mul(big.NewFromString("-1"))
		}
		//return (isCash && pnl.GTE(t.profitMargin)) || (isFutures && pnlDollar.GTE(t.minProfitPerTrade)), pnl, pnlDollar
		return pnlDollar.GTE(t.minProfitPerTrade), pnl, pnlDollar
	}
	return false, big.ZERO, big.ZERO
}

// placeMarketOrder places a market order on a given runner.
// this can be used in different scenarios, such as close positions.
func (t *trader) placeMarketOrder(r *runner.Runner, side, quantity string) error {
	switch r.GetMarketType() {
	case runner.Cash:
		_, quantityPrecision, err := t.provider.fetchBinSpotExchangeInfo(r.GetName())
		if err != nil {
			return err
		}
		_, err = t.provider.binSpot.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bn.SideType(side)).
			Type(bn.OrderTypeMarket).
			Quantity(big.NewFromString(quantity).FormattedString(quantityPrecision)).
			Do(context.Background())
		return err
	case runner.Futures:
		_, quantityPrecision, err := t.provider.fetchBinFutuExchangeInfo(r.GetName())
		if err != nil {
			return err
		}
		_, err = t.provider.binFutu.NewCreateOrderService().
			Symbol(r.GetName()).
			Side(bnf.SideType(side)).
			Type(bnf.OrderTypeMarket).
			Quantity(big.NewFromString(quantity).FormattedString(quantityPrecision)).
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
// which will place trades if the given runner passes the initialChecks method and
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
		t.binTrades.Store(r.GetUniqueName(), st)
		go t.monitorBinTrade(st)
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
		t.binTrades.Store(r.GetUniqueName(), st)
		go t.monitorBinTrade(st)
	default:
		return nil
	}
	return nil
}

// monitorBinSpotTrade monitors the trade after an order is placed successfully.
func (t *trader) monitorBinTrade(st *setup) {
	defer t.report(st)

	// Waiting for the order to be (partially) filled
	nw := time.Now()
	maxWait := t.maxWaitToFill
	if val, ok := st.signal.GetMaxWaitToFill(); ok {
		maxWait = val
	}
	for st.orderStatus != "FILLED" && st.orderStatus != "PARTIALLY_FILLED" {
		time.Sleep(time.Millisecond * 100)
		if st.orderStatus == "CANCELED" ||
			st.orderStatus == "REJECTED" ||
			st.orderStatus == "EXPIRED" ||
			st.orderStatus == "PENDING_CANCEL" {
			return
		}
		if time.Now().Sub(nw) > maxWait {
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
	t.registerStreamingChannel(st)
	bestPrice, isCash := tax.NewPriceLevel(), st.runner.GetMarketType() == runner.Cash
	for msg := range st.channels.depth {
		if isCash {
			bestPrice = tax.BinanceSpotBestBidAskFromDepth(msg.(*bn.WsPartialDepthEvent)).L1ForClosingTrade(st.orderSide)
		} else {
			bestPrice = tax.BinanceFutuBestBidAskFromDepth(msg.(*bnf.WsDepthEvent)).L1ForClosingTrade(st.orderSide)
		}
		ok, pnl, _ := t.shouldClose(st, bestPrice.Price)
		if !ok {
			continue
		}
		st.pnl = pnl
		var balance string
		if isCash {
			val, ok := t.binSpotBalances.Load(st.runner.GetName())
			if !ok {
				st.isClose = t.registerStreamingChannel(st)
				continue
			}
			balance = val.(bn.Balance).Free
		} else {
			val, ok := t.binFutuBalances.Load(st.runner.GetName())
			if !ok {
				st.isClose = t.registerStreamingChannel(st)
				continue
			}
			balance = val.(bnf.Balance).Balance
		}
		if err := t.placeMarketOrder(st.runner, st.signal.CloseTradingSide(), balance); err != nil {
			t.logger.Error.Println(t.newLog(err.Error()))
		}
		st.lastUpdatedAt = time.Now().Unix() * 1000
		st.isClose = t.registerStreamingChannel(st)
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
//func (t *trader) monitorBinFutuTrade(st *setup) {
//	defer t.report(st)
//
//	// Waiting for the order to be (partially) filled
//	nw := time.Now()
//	for st.orderStatus != "FILLED" && st.orderStatus != "PARTIALLY_FILLED" {
//		time.Sleep(time.Millisecond * 500)
//		if st.orderStatus == "CANCELED" ||
//			st.orderStatus == "REJECTED" ||
//			st.orderStatus == "EXPIRED" ||
//			st.orderStatus == "PENDING_CANCEL" {
//			return
//		}
//		if time.Now().Sub(nw) > t.maxWaitToFill {
//			if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
//				t.logger.Error.Println(t.newLog(err.Error()))
//			}
//		}
//	}
//
//	// Upto this point, the order has been filled,
//	// you basically could place a stop order or an OCO order to control your risk and return
//	// or listen to streaming channels, depth or trade event, to manage the trade yourself.
//	for st.avgFilledPrice.EQ(big.ZERO) {
//		time.Sleep(time.Second)
//	}
//	st.channels = &streamingChannels{depth: make(chan interface{}, 20)}
//	t.registerStreamingChannel(st)
//	for msg := range st.channels.depth {
//		bestPrice := tax.BinanceFutuBestBidAskFromDepth(msg.(*bnf.WsDepthEvent)).L1ForClosingTrade(st.orderSide)
//		ok, pnl, _ := t.shouldClose(st, bestPrice.Price)
//		if !ok {
//			continue
//		}
//		st.pnl = pnl
//		val, ok := t.binFutuBalances.Load(st.runner.GetName())
//		if !ok {
//			st.isClose = t.registerStreamingChannel(st)
//			continue
//		}
//		if err := t.placeMarketOrder(st.runner, st.signal.CloseTradingSide(), val.(bnf.Balance).Balance); err != nil {
//			t.logger.Error.Println(t.newLog(err.Error()))
//		}
//		st.isClose = t.registerStreamingChannel(st)
//		st.lastUpdatedAt = time.Now().Unix() * 1000
//	}
//	//// Upto this point, the trade should be close, and converted back to
//	//// the quote currency, which should be in USDT, and report PNLs.
//	//// The outstanding portion or the order should be canceled.
//	if !t.isOrdering(st.runner) {
//		return
//	}
//	if err := t.cancleOpenOrder(st.runner, st.orderID); err != nil {
//		t.logger.Error.Println(t.newLog(err.Error()))
//	}
//}

// report reports a trade to user after closing it. It also store the trade
// to some persistent storage for performance evaluation later.
func (t *trader) report(st *setup) {
	t.provider.dbClient.InsertOrUpdateSetups([]*db.Setup{st.convertDB()})
	t.logger.Info.Println(t.newLog(st.description()))
	t.communicator.trader2Notifier <- t.communicator.newMessage(st.runner, st.signal, nil, st.description(), nil)
}

// binSpotUserDataStreaming manages all account changing events from trading activities on cash account.
func (t *trader) binSpotUserDataStreaming() {
	isError, isInit := false, true
	dataHandler := func(e *bn.WsUserDataEvent) {
		switch e.Event {
		case bn.UserDataEventTypeBalanceUpdate:
			fallthrough
		case bn.UserDataEventTypeOutboundAccountPosition:
			if err := t.binSpotUpdateBalances(); err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
			}
		case bn.UserDataEventTypeExecutionReport:
			if val, ok := t.binTrades.Load(e.OrderUpdate.Symbol); ok && e.OrderUpdate.Id == val.(*setup).orderID {
				val.(*setup).binSpotUpdateTrade(e.OrderUpdate)
				t.provider.dbClient.InsertOrUpdateSetups([]*db.Setup{val.(*setup).convertDB()})
			}
		case bn.UserDataEventTypeListStatus:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %s, %+v", string(e.Event), *e)))
		default:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("cash %s, %+v", "unknown event", *e)))
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
			if err := t.binFutuUpdateBalances(); err != nil {
				t.logger.Error.Println(t.newLog(err.Error()))
			}
		case bnf.UserDataEventTypeOrderTradeUpdate:
			if val, ok := t.binTrades.Load(e.OrderTradeUpdate.Symbol + "PERP"); ok && e.OrderTradeUpdate.ID == val.(*setup).orderID {
				val.(*setup).binFutuUpdateTrade(e.OrderTradeUpdate)
				t.provider.dbClient.InsertOrUpdateSetups([]*db.Setup{val.(*setup).convertDB()})
			}
		case bnf.UserDataEventTypeMarginCall:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, %s, %+v", string(e.Event), *e)))
		default:
			t.logger.Info.Println(t.newLog(fmt.Sprintf("futu, %s, %+v", "unknown event", *e)))
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
// receive candle or depth broadcasted by data provider.
func (t *trader) registerStreamingChannel(st *setup) bool {
	if st.isClose {
		return st.isClose
	}
	done := false
	var maxTries int
	for !done && maxTries <= 3 {
		resC := make(chan *payload)
		t.communicator.trader2Streamer <- t.communicator.newMessage(st.runner, nil, st.channels, nil, resC)
		done = (<-resC).what.dynamic.(bool)
		maxTries++
	}
	time.Sleep(time.Second)
	return done
}

// generates a new log with the format for the watcher
func (t *trader) newLog(message string) string {
	return fmt.Sprintf("[trader] %s", message)
}
