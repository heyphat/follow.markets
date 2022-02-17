package market

import (
	"fmt"
	"strings"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/internal/pkg/strategy"
	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"
)

// trader trades on a setup.
type setup struct {
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

	trades []*tradeUpdate
}

type tradeUpdate struct {
	id       int64  `json:"trade_id"`
	time     int64  `json:"trade_time"`
	cost     string `json:"commission"`
	costAss  string `json:"commission_asset"`
	price    string `json:"price"`
	quantity string `json:"quantity"`
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
			orderStatus:    string(od.Status),
			orderSide:      string(od.Side),
			orderPrice:     od.Price,
			orderQtity:     od.OrigQuantity,
			accTradingFee:  big.ZERO,
			avgFilledPrice: big.ZERO,
			accFilledQtity: big.ZERO,
			pnl:            big.ZERO,
			trades:         make([]*tradeUpdate, 0),
		}
	case runner.Futures:
		od := o.(*bnf.CreateOrderResponse)
		return &setup{
			runner: r, signal: s,
			orderID:        od.OrderID,
			orderTime:      od.UpdateTime,
			orderStatus:    string(od.Status),
			orderSide:      string(od.Side),
			usedLeverage:   leverage,
			orderPrice:     od.Price,
			orderQtity:     od.OrigQuantity,
			accTradingFee:  big.ZERO,
			avgFilledPrice: big.ZERO,
			accFilledQtity: big.ZERO,
			pnl:            big.ZERO,
			trades:         make([]*tradeUpdate, 0),
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
	if s.runner.GetMarketType() != runner.Cash || strings.ToUpper(u.ExecutionType) != "TRADE" {
		return
	}
	s.trades = append(s.trades, &tradeUpdate{
		id:       u.TradeId,
		time:     u.TransactionTime,
		price:    u.LatestPrice,
		quantity: u.LatestVolume,
		cost:     u.FeeCost,
		costAss:  u.FeeAsset,
	})
	//fmt.Println(fmt.Sprintf("new trade: %+v", *(s.trades[len(s.trades)-1])))
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
	if s.runner.GetMarketType() != runner.Futures || strings.ToUpper(string(u.ExecutionType)) != "TRADE" {
		return
	}
	s.trades = append(s.trades, &tradeUpdate{
		id:       u.TradeID,
		time:     u.TradeTime,
		price:    u.LastFilledPrice,
		quantity: u.LastFilledQty,
		cost:     u.Commission,
		costAss:  u.CommissionAsset,
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

type setupJSON struct {
	ticker         string         `json:"ticker"`
	signal         string         `json:"signal"`
	orderID        int64          `json:"order_id"`
	orderTime      int64          `json:"order_time"`
	orderSide      string         `json:"order_side"`
	orderPrice     string         `json:"order_price"`
	orderQtity     string         `json:"order_quantity"`
	orderStatus    string         `json:"order_status"`
	accTradingFee  string         `json:"commission"`
	usedLeverage   string         `json:"leverage"`
	tradingFeeAss  string         `json:"commission_asset"`
	avgFilledPrice string         `json:"avg_filled_price"`
	accFilledQtity string         `json:"acc_filled_quantity"`
	pnl            string         `json:"pnl"`
	trades         []*tradeUpdate `json:"trades"`
}

func (st *setup) convert2JSON() *setupJSON {
	return &setupJSON{
		ticker:         st.runner.GetName(),
		signal:         st.signal.Name,
		orderID:        st.orderID,
		orderTime:      st.orderTime,
		orderSide:      st.orderSide,
		orderQtity:     st.orderQtity,
		orderStatus:    st.orderStatus,
		tradingFeeAss:  st.tradingFeeAss,
		usedLeverage:   st.usedLeverage.FormattedString(0),
		accTradingFee:  st.accTradingFee.FormattedString(10),
		avgFilledPrice: st.avgFilledPrice.FormattedString(10),
		accFilledQtity: st.accFilledQtity.FormattedString(10),
		pnl:            st.pnl.FormattedString(10),
		trades:         st.trades,
	}
}

func (st *setup) description() string {
	t := time.Unix(st.orderTime/1000, 0)
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
		t.Format(simpleLayout),
		st.orderSide,
		st.orderQtity,
		st.orderPrice,
		st.orderStatus,
		st.pnl.FormattedString(8),
		st.pnl.Mul(st.usedLeverage).Mul(st.avgFilledPrice.Mul(st.accFilledQtity)).FormattedString(2),
		st.avgFilledPrice.FormattedString(8),
		st.accFilledQtity.FormattedString(2),
		st.accTradingFee.FormattedString(8),
		len(st.trades),
	)
}
