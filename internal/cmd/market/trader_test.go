package market

import (
	"fmt"
	"testing"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Trader(t *testing.T) {
	fmt.Println("start to test on trader")
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	trader, err := newTrader(initSharedParticipants(configs))
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())
	//for {
	//	time.Sleep(10 * time.Second)
	//}
}
