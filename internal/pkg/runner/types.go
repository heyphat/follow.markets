package runner

import "strings"

type AssetClass string

const (
	Crypto AssetClass = "CRYPTO"
	Stock  AssetClass = "STOCK"
	Forex  AssetClass = "Forex"
)

type MarketType string

const (
	Cash    MarketType = "CASH"
	Margin  MarketType = "MARGIN"
	Futures MarketType = "FUTURES"
)

func ValidateMarket(market string) (MarketType, bool) {
	if strings.ToUpper(market) == "CASH" {
		return Cash, true
	}
	if strings.ToUpper(market) == "FUTURES" {
		return Futures, true
	}
	//if strings.ToUpper(market) == "MARGIN" {
	//	return Margin, true
	//}
	return Cash, false
}
