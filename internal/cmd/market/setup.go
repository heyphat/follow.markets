package market

import (
	"fmt"
	"strings"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"

	db "follow.markets/internal/pkg/database"
	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	"follow.markets/pkg/util"
)

// trader trades on a setup.
type setup struct {
	isClose  bool
	runner   *runner.Runner
	signal   *strategy.Signal
	channels *streamingChannels

	orderID        int64
	orderTime      int64
	orderSide      string
	orderPrice     string
	orderQtity     string
	orderStatus    string
	usedLeverage   big.Decimal
	tradingFeeAss  string
	accTradingFee  big.Decimal
	avgFilledPrice big.Decimal
	accFilledQtity big.Decimal
	pnl            big.Decimal
	lastUpdatedAt  int64

	trades []*db.Trade
}

// newSetup returns a new setup for trader.
func newSetup(r *runner.Runner, s *strategy.Signal, leverage big.Decimal, o interface{}) *setup {
	switch r.GetMarketType() {
	case runner.Cash:
		od := o.(*bn.CreateOrderResponse)
		return &setup{
			runner: r, signal: s,
			orderID:        od.OrderID,
			orderTime:      od.TransactTime,
			lastUpdatedAt:  od.TransactTime,
			orderStatus:    string(od.Status),
			orderSide:      string(od.Side),
			usedLeverage:   leverage,
			orderPrice:     od.Price,
			orderQtity:     od.OrigQuantity,
			tradingFeeAss:  "BNB",
			accTradingFee:  big.ZERO,
			avgFilledPrice: big.ZERO,
			accFilledQtity: big.ZERO,
			pnl:            big.ZERO,
			trades:         make([]*db.Trade, 0),
		}
	case runner.Futures:
		od := o.(*bnf.CreateOrderResponse)
		return &setup{
			runner: r, signal: s,
			orderID:        od.OrderID,
			orderTime:      od.UpdateTime,
			lastUpdatedAt:  od.UpdateTime,
			orderStatus:    string(od.Status),
			orderSide:      string(od.Side),
			usedLeverage:   leverage,
			orderPrice:     od.Price,
			orderQtity:     od.OrigQuantity,
			tradingFeeAss:  "USDT",
			accTradingFee:  big.ZERO,
			avgFilledPrice: big.ZERO,
			accFilledQtity: big.ZERO,
			pnl:            big.ZERO,
			trades:         make([]*db.Trade, 0),
		}
	default:
		return nil
	}
}

// binSpotUpdateTrade update the setupt with new trade activities,
// it adds filled quantity, recomputes average filled price
// and logs trades.
func (s *setup) binSpotUpdateTrade(u bn.WsOrderUpdate) {
	s.orderStatus = u.Status
	s.lastUpdatedAt = u.TransactionTime
	if s.runner.GetMarketType() != runner.Cash || strings.ToUpper(u.ExecutionType) != "TRADE" {
		return
	}
	s.tradingFeeAss = u.FeeAsset
	s.trades = append(s.trades, &db.Trade{
		ID:       u.TradeId,
		Time:     util.ConvertUnixMillisecond2Time(u.TransactionTime),
		Price:    u.LatestPrice,
		Quantity: u.LatestVolume,
		Cost:     u.FeeCost,
		Status:   u.Status,
		CostAss:  u.FeeAsset,
	})
	if s.avgFilledPrice.EQ(big.ZERO) || s.accFilledQtity.EQ(big.ZERO) {
		s.avgFilledPrice = big.NewFromString(u.LatestPrice)
		s.accFilledQtity = big.NewFromString(u.LatestVolume)
		s.accTradingFee = big.NewFromString(u.FeeCost)
		return
	}
	filled := big.NewFromString(u.FilledVolume)
	lastFilled := big.NewFromString(u.LatestVolume)
	lastPrice := big.NewFromString(u.LatestPrice)
	s.avgFilledPrice = s.avgFilledPrice.Mul(s.accFilledQtity.Div(filled)).Add(lastPrice.Mul(lastFilled.Div(filled)))
	s.accFilledQtity = filled
	s.accTradingFee = s.accTradingFee.Add(big.NewFromString(u.FeeCost))
}

// binFutuUpdateTrade update the setupt with new trade activities,
// it adds filled quantity, recomputes average filled price
// and logs trades.
func (s *setup) binFutuUpdateTrade(u bnf.WsOrderTradeUpdate) {
	s.orderStatus = string(u.Status)
	s.lastUpdatedAt = u.TradeTime
	if s.runner.GetMarketType() != runner.Futures || strings.ToUpper(string(u.ExecutionType)) != "TRADE" {
		return
	}
	s.tradingFeeAss = u.CommissionAsset
	s.trades = append(s.trades, &db.Trade{
		ID:       u.TradeID,
		Time:     util.ConvertUnixMillisecond2Time(u.TradeTime),
		Price:    u.LastFilledPrice,
		Quantity: u.LastFilledQty,
		Cost:     u.Commission,
		Status:   string(u.Status),
		CostAss:  u.CommissionAsset,
	})
	if s.avgFilledPrice.EQ(big.ZERO) || s.accFilledQtity.EQ(big.ZERO) {
		s.avgFilledPrice = big.NewFromString(u.LastFilledPrice)
		s.accFilledQtity = big.NewFromString(u.LastFilledQty)
		s.accTradingFee = big.NewFromString(u.Commission)
		return
	}
	filled := big.NewFromString(u.AccumulatedFilledQty)
	lastFilled := big.NewFromString(u.LastFilledQty)
	lastPrice := big.NewFromString(u.LastFilledPrice)
	s.avgFilledPrice = s.avgFilledPrice.Mul(s.accFilledQtity.Div(filled)).Add(lastPrice.Mul(lastFilled.Div(filled)))
	s.accFilledQtity = filled
	s.accTradingFee = s.accTradingFee.Add(big.NewFromString(u.Commission))
}

func (st *setup) convertDB() *db.Setup {
	return &db.Setup{
		Ticker:         st.runner.GetUniqueName(),
		Market:         string(st.runner.GetMarketType()),
		Broker:         "Binance",
		Signal:         st.signal.Name,
		OrderID:        st.orderID,
		OrderTime:      util.ConvertUnixMillisecond2Time(st.orderTime),
		OrderSide:      st.orderSide,
		OrderPrice:     st.orderPrice,
		OrderQtity:     st.orderQtity,
		OrderStatus:    st.orderStatus,
		TradingFeeAss:  st.tradingFeeAss,
		UsedLeverage:   st.usedLeverage.FormattedString(0),
		AccTradingFee:  st.accTradingFee.FormattedString(8),
		AvgFilledPrice: st.avgFilledPrice.FormattedString(8),
		AccFilledQtity: st.accFilledQtity.FormattedString(8),
		PNL:            st.pnl.Mul(big.NewDecimal(100.0)).FormattedString(2),
		DollarPNL:      st.pnl.Mul(st.usedLeverage).Mul(st.avgFilledPrice.Mul(st.accFilledQtity)).FormattedString(2),
		LastUpdatedAt:  util.ConvertUnixMillisecond2Time(st.lastUpdatedAt),
		Trades:         st.trades,
	}
}

func (st *setup) description() string {
	s := `
=================================
|         TRADE REPORT          |
=================================
|           ORDER               |
---------------------------------
ticker:         %s, 
signal:         %s,
market:         %s, 
leverage:       %sx,
order time:     %s,
close time:     %s,
order side:     %s,
order quantity: %s,
orer price:     %s,
order status:   %s,
|-------------------------------|
|           RESULT              | 
---------------------------------
pnl:                %s,
pnl dollar:         %s, 
avg. filled price:  %s,
acc. filled volume: %s,
acc. trading fee:   %s,
n. of trades:       %d,
=================================
`
	return fmt.Sprintf(s,
		st.runner.GetName(),
		st.signal.Name,
		st.runner.GetMarketType(),
		st.usedLeverage.FormattedString(0),
		util.ConvertUnixMillisecond2Time(st.orderTime).Format(simpleLayout),
		util.ConvertUnixMillisecond2Time(st.lastUpdatedAt).Format(simpleLayout),
		st.orderSide,
		st.orderQtity,
		st.orderPrice,
		st.orderStatus,
		st.pnl.Mul(big.NewDecimal(100.0)).FormattedString(2)+"%",
		st.pnl.Mul(st.usedLeverage).Mul(st.avgFilledPrice.Mul(st.accFilledQtity)).FormattedString(2),
		st.avgFilledPrice.FormattedString(8),
		st.accFilledQtity.FormattedString(2),
		st.accTradingFee.FormattedString(8),
		len(st.trades),
	)
}
