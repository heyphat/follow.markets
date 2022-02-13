package techanex

import (
	"testing"
	"time"

	bn "github.com/adshao/go-binance/v2"
	bnc "github.com/adshao/go-binance/v2/common"
	"github.com/stretchr/testify/assert"
)

func Test_L1(t *testing.T) {
	d := bn.WsPartialDepthEvent{
		Symbol:       "BTCUSDT",
		LastUpdateID: time.Now().Unix(),
		Bids: []bn.Bid{
			bnc.PriceLevel{
				Price:    "20",
				Quantity: "1",
			},
		},
		Asks: []bn.Ask{
			bnc.PriceLevel{
				Price:    "100",
				Quantity: "1",
			},
		},
	}

	l1 := BinanceSpotBestBidAskFromDepth(d)
	assert.EqualValues(t, "20", l1.BestBid.Price.FormattedString(0))
	assert.EqualValues(t, "100", l1.BestAsk.Price.FormattedString(0))

	spread := l1.Spread()
	assert.EqualValues(t, "80", spread.FormattedString(0))
	spreadOfBid := l1.SpreadPercentageOfBid()
	assert.EqualValues(t, "400", spreadOfBid.FormattedString(0))
}
