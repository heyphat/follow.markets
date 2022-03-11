package market

import (
	"io/ioutil"
	"testing"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	tax "follow.markets/internal/pkg/techanex"
	"follow.markets/pkg/config"
	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"
	"github.com/stretchr/testify/assert"
)

func testSuit(ticker string) (*trader, *runner.Runner, error) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	if err != nil {
		return nil, nil, err
	}
	trader, err := newTrader(initSharedParticipants(configs), configs)
	if err != nil {
		return nil, nil, err
	}
	r := runner.NewRunner(ticker, runner.NewRunnerDefaultConfigs())
	return trader, r, nil
}

func Test_Trader(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	trader, err := newTrader(initSharedParticipants(configs), configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	mem := setup{
		runner: runner.NewRunner("BTCUSDT", runner.NewRunnerDefaultConfigs()),
		channels: &streamingChannels{
			bar:   nil,
			trade: nil,
			depth: make(chan interface{}, 10),
		},
	}

	assert.EqualValues(t, "BTCUSDT", mem.runner.GetName())
}

func Test_Trader_Evaluator(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	common := initSharedParticipants(configs)
	trader, err := newTrader(common, configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	trader.connect()
	assert.EqualValues(t, true, trader.isConnected())

	notifier, err := newNotifier(common, configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, notifier.isConnected())

	notifier.connect()
	assert.EqualValues(t, true, notifier.isConnected())

	streamer, err := newStreamer(common)
	assert.EqualValues(t, nil, err)

	streamer.connect()
	assert.EqualValues(t, true, notifier.isConnected())

	ticker := "ETHUSDT"
	rConfigs := runner.NewRunnerDefaultConfigs()
	rConfigs.Market = runner.Futures
	r := runner.NewRunner(ticker, rConfigs)
	assert.EqualValues(t, ticker, r.GetName())
	kline := &bn.Kline{
		OpenTime: 1499040000000,
		Open:     "0.0",
		High:     "0.8",
		Low:      "0.01",
		Close:    "0.2",
		Volume:   "148976.1",
		TradeNum: 308,
	}

	candle := tax.ConvertBinanceKline(kline, nil)
	ok := r.SyncCandle(candle)
	assert.EqualValues(t, true, ok)

	signalPath := "./../../../configs/signals/signal.json"
	raw, err := ioutil.ReadFile(signalPath)
	assert.EqualValues(t, nil, err)
	s, err := strategy.NewSignalFromBytes(raw)
	assert.EqualValues(t, nil, err)

	common.communicator.evaluator2Trader <- common.communicator.newMessage(r, s, nil, nil, nil)

	time.Sleep(time.Minute * 1)
}

func Test_Trader_ShouldClose(t *testing.T) {
	ticker := "BTCUSDT"
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)
	configs.Market.Trader.MaxLeverage = 10
	configs.Market.Trader.MaxLossPerTrade = 5
	configs.Market.Trader.MinProfitPerTrade = 10

	trader, err := newTrader(initSharedParticipants(configs), configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())

	rConfigs := runner.NewRunnerDefaultConfigs()
	rConfigs.Market = runner.Futures
	isCash := rConfigs.Market == runner.Cash
	r := runner.NewRunner(ticker, rConfigs)

	trader.connect()
	assert.EqualValues(t, true, trader.isConnected())

	st := &setup{runner: r, avgFilledPrice: big.NewFromString("10.0"), accFilledQtity: big.NewFromString("1000")}
	if r.GetMarketType() == runner.Cash {
		st.usedLeverage = big.ONE
	} else {
		st.usedLeverage = big.NewFromString("10")
	}

	st.orderSide = "BUY"
	currentPrice := big.NewFromString("10.3")
	shouldStop, pnl, dl := trader.shouldClose(st, currentPrice)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "0.03", pnl.FormattedString(2))
	if isCash {
		assert.EqualValues(t, "300.00", dl.FormattedString(2))
	} else {
		assert.EqualValues(t, "3000.00", dl.FormattedString(2))
	}

	currentPrice = big.NewFromString("9.5")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "-0.05", pnl.FormattedString(2))
	if isCash {
		assert.EqualValues(t, "-526.32", dl.FormattedString(2))
	} else {
		assert.EqualValues(t, "-5263.16", dl.FormattedString(2))
	}

	currentPrice = big.NewFromString("10.05")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	if isCash {
		assert.EqualValues(t, false, shouldStop)
	} else {
		assert.EqualValues(t, true, shouldStop)
	}
	assert.EqualValues(t, "0.005", pnl.FormattedString(3))

	currentPrice = big.NewFromString("9.99")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	if isCash {
		assert.EqualValues(t, false, shouldStop)
	} else {
		assert.EqualValues(t, true, shouldStop)
	}
	assert.EqualValues(t, "-0.001", pnl.FormattedString(3))

	st.orderSide = "SELL"
	currentPrice = big.NewFromString("10.5")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "-0.05", pnl.FormattedString(2))

	currentPrice = big.NewFromString("9.5")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	assert.EqualValues(t, true, shouldStop)
	assert.EqualValues(t, "0.05", pnl.FormattedString(2))

	currentPrice = big.NewFromString("10.01")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	if isCash {
		assert.EqualValues(t, false, shouldStop)
	} else {
		assert.EqualValues(t, true, shouldStop)
	}
	assert.EqualValues(t, "-0.001", pnl.FormattedString(3))

	currentPrice = big.NewFromString("9.99")
	shouldStop, pnl, dl = trader.shouldClose(st, currentPrice)
	if isCash {
		assert.EqualValues(t, false, shouldStop)
	} else {
		assert.EqualValues(t, true, shouldStop)
	}
	assert.EqualValues(t, "0.001", pnl.FormattedString(3))
}

func Test_Trader_IsOrdering(t *testing.T) {
	trader, runner, err := testSuit("BTCUSDT")
	assert.EqualValues(t, nil, err)

	ok := trader.isOrdering(runner)
	assert.EqualValues(t, false, ok)
}

func Test_Trader_IsHolding(t *testing.T) {
	trader, runner, err := testSuit("SHIBUSDT")
	assert.EqualValues(t, nil, err)

	coin := bn.Balance{Asset: "SHIB", Free: "30", Locked: "0"}
	trader.binSpotBalances.Store("SHIBUSDT", coin)
	ok := trader.isHolding(runner)
	assert.EqualValues(t, true, ok)

	_, newRunner, err := testSuit("THETAUSDT")
	assert.EqualValues(t, nil, err)
	ok = trader.isHolding(newRunner)
	assert.EqualValues(t, false, ok)
}

func Test_Trader_GetAndUpdateSpotBalances(t *testing.T) {
	trader, _, err := testSuit("BTCUSDT")
	assert.EqualValues(t, nil, err)

	bls := []bn.Balance{bn.Balance{Asset: "BNB", Free: "10", Locked: "0"}}
	for _, b := range bls {
		trader.binSpotBalances.Store(b.Asset+trader.quoteCurrency, b)
	}

	nbls := trader.binSpotGetBalances()
	assert.EqualValues(t, true, len(nbls) >= 1)

	// balances are queried directly from broker.
	// test add new balance to BNB
	//bnb := bn.Balance{Asset: "BNB", Free: "30", Locked: "0"}
	//trader.binSpotUpdateBalances(bnb)
	//nbls = trader.binSpotGetBalances()
	//for _, b := range nbls {
	//	if b.Asset == "BNB" {
	//		assert.EqualValues(t, "30", b.Free)
	//	}
	//}

	//// test remove balance from the holdings
	//bnb = bn.Balance{Asset: "BNB", Free: "0", Locked: "0"}
	//trader.binSpotUpdateBalances(bnb)
	//nbls = trader.binSpotGetBalances()
	//for _, b := range nbls {
	//	assert.EqualValues(t, true, b.Asset != "BNB")
	//}
}

func Test_Trader_GetAndUpdateFutuBalances(t *testing.T) {
	trader, _, err := testSuit("BTCUSDT")
	assert.EqualValues(t, nil, err)

	bls := []bnf.Balance{bnf.Balance{Asset: "BNB", Balance: "10", AvailableBalance: "10"}}
	for _, b := range bls {
		trader.binFutuBalances.Store(b.Asset+trader.quoteCurrency, b)
	}

	nbls := trader.binFutuGetBalances()
	assert.EqualValues(t, true, len(nbls) >= 1)

	// test add new balance to BNB
	//bnb := bnf.Balance{Asset: "BNB", Balance: "30", AvailableBalance: "30"}
	//trader.binFutuUpdateBalances(bnb)
	//nbls = trader.binFutuGetBalances()
	//for _, b := range nbls {
	//	if b.Asset == "BNB" {
	//		assert.EqualValues(t, "30", b.Balance)
	//	}
	//}

	//// test remove balance from the holdings
	//bnb = bnf.Balance{Asset: "BNB", Balance: "0", AvailableBalance: "0"}
	//trader.binFutuUpdateBalances(bnb)
	//nbls = trader.binFutuGetBalances()
	//for _, b := range nbls {
	//	assert.EqualValues(t, true, b.Asset != "BNB")
	//}
}

func Test_Trader_UpdateConfigs(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	trader, err := newTrader(initSharedParticipants(configs), configs)
	assert.EqualValues(t, nil, err)

	assert.EqualValues(t, "5.0", trader.maxLossPerTrade.FormattedString(1))
	assert.EqualValues(t, "10.0", trader.minProfitPerTrade.FormattedString(1))

	configs.Market.Trader.MaxLossPerTrade = 10
	configs.Market.Trader.MinProfitPerTrade = 20

	trader.updateConfigs(configs)

	assert.EqualValues(t, "10.0", trader.maxLossPerTrade.FormattedString(1))
	assert.EqualValues(t, "20.0", trader.minProfitPerTrade.FormattedString(1))
}

func Test_Trader_IsRecentlyTraded(t *testing.T) {
	trader, r, err := testSuit("BTCUSDT")
	assert.EqualValues(t, nil, err)

	o := &bn.CreateOrderResponse{
		Price: "10", OrigQuantity: "20",
		TransactTime: time.Now().Unix() * 1000}
	s := &strategy.Signal{}
	st := newSetup(r, s, big.ONE, o)

	trader.binTrades.Store(r.GetUniqueName(), st)

	assert.EqualValues(t, true, trader.isRecentlyTraded(r))
}
