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

<<<<<<< HEAD
	trader, err := newTrader(initSharedParticipants(configs), configs)
=======
	trader, err := newTrader(initSharedParticipants(configs))
>>>>>>> ef7ad40c9ceae9b107e5317419f98cf377cda0f6
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, trader.isConnected())
	//for {
	//	time.Sleep(10 * time.Second)
	//}
}
