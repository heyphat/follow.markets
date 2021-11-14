package techanex

import "github.com/sdcoffey/big"

type Trade struct {
	Price        big.Decimal
	Quantity     big.Decimal
	TradeTime    int64
	IsBuyerMaker bool
}

func NewTrade() *Trade {
	return &Trade{
		Price:    big.ZERO,
		Quantity: big.ZERO,
	}
}
