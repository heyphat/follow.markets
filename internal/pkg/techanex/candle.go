package techanex

import "github.com/sdcoffey/big"

func change(purchased, sold big.Decimal) big.Decimal {
	if purchased.EQ(big.ZERO) {
		return big.ZERO
	}
	return sold.Sub(purchased).Div(purchased).Mul(big.NewDecimal(100.0))
}

func LowHigh(low, high big.Decimal) big.Decimal {
	return change(low, high)
}

func OpenClose(open, closeP big.Decimal) big.Decimal {
	return change(open, closeP)
}

func OpenHigh(open, high big.Decimal) big.Decimal {
	return change(open, high)
}

func OpenLow(open, low big.Decimal) big.Decimal {
	return change(open, low)
}

func LowClose(low, closeP big.Decimal) big.Decimal {
	return change(low, closeP)
}

func HighClose(high, closeP big.Decimal) big.Decimal {
	return change(high, closeP)
}
