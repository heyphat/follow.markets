package techanex

import (
	"strings"

	bn "github.com/adshao/go-binance/v2"
	bnf "github.com/adshao/go-binance/v2/futures"
	"github.com/sdcoffey/big"
)

func BinanceSpotBestBidAskFromDepth(d *bn.WsPartialDepthEvent) *L1 {
	l1 := NewL1()
	if len(d.Bids) > 0 {
		l1.BestBid.Price = big.NewFromString(d.Bids[0].Price)
		l1.BestBid.Quantity = big.NewFromString(d.Bids[0].Quantity)
	}
	if len(d.Asks) > 0 {
		l1.BestAsk.Price = big.NewFromString(d.Asks[0].Price)
		l1.BestAsk.Quantity = big.NewFromString(d.Asks[0].Quantity)
	}
	return l1
}

func BinanceFutuBestBidAskFromDepth(d *bnf.WsDepthEvent) *L1 {
	l1 := NewL1()
	if len(d.Bids) > 0 {
		l1.BestBid.Price = big.NewFromString(d.Bids[0].Price)
		l1.BestBid.Quantity = big.NewFromString(d.Bids[0].Quantity)
	}
	if len(d.Asks) > 0 {
		l1.BestAsk.Price = big.NewFromString(d.Asks[0].Price)
		l1.BestAsk.Quantity = big.NewFromString(d.Asks[0].Quantity)
	}
	return l1
}

type PriceLevel struct {
	Price    big.Decimal
	Quantity big.Decimal
}

func NewPriceLevel() *PriceLevel {
	return &PriceLevel{
		Price:    big.ZERO,
		Quantity: big.ZERO,
	}
}

type L1 struct {
	BestBid *PriceLevel
	BestAsk *PriceLevel
}

func NewL1() *L1 {
	return &L1{
		BestBid: NewPriceLevel(),
		BestAsk: NewPriceLevel(),
	}
}

// If BUY, returns best current bid, since you are holding asset, you would like to sell it to the best bid.
// If SOLD, returs best current ask, since you are selling asset, you would like to buy it back from the best ask.
func (l1 *L1) L1ForClosingTrade(side string) *PriceLevel {
	if strings.ToUpper(side) == "BUY" {
		return l1.BestBid
	}
	return l1.BestAsk
}

func (l1 *L1) L1ForOpeningTrade(side string) *PriceLevel {
	if strings.ToUpper(side) == "BUY" {
		return l1.BestAsk
	}
	return l1.BestBid
}

func (l1 *L1) Spread() big.Decimal {
	return l1.BestAsk.Price.Sub(l1.BestBid.Price)
}

func (l1 *L1) SpreadPercentageOfBid() big.Decimal {
	if l1.BestBid.Price.EQ(big.ZERO) {
		return big.ZERO
	}
	return l1.Spread().Div(l1.BestBid.Price).Mul(big.NewDecimal(100.0))
}
