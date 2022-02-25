package market

import (
	"fmt"
	"testing"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Tester(t *testing.T) {
	consigPath := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&consigPath)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, "development", configs.Stage)

	//signalPath := "./../../../configs/signals/signal.json"
	//raw, err := ioutil.ReadFile(signalPath)
	//assert.EqualValues(t, nil, err)

	//signal, err := strategy.NewSignalFromBytes(raw)
	//assert.EqualValues(t, nil, err)

	//tester, err := newTester(initSharedParticipants(configs))
	//assert.EqualValues(t, nil, err)

	//stg := strategy.Strategy{
	//	EntryRule:      strategy.NewRule(*signal),
	//	ExitRule:       nil,
	//	RiskRewardRule: strategy.NewRiskRewardRule(-0.02, 0.04),
	//}
	//rs, err := tester.test(ticker, big.NewDecimal(10000), &stg, nil, nil, "./test_result")
	//assert.EqualValues(t, nil, err)
	//assert.EqualValues(t, len(rs.record.Trades), len(rs.record.Trades))
}

func Test_Tester_NotionTest(t *testing.T) {
	consigPath := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&consigPath)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, "development", configs.Stage)

	tester, err := newTester(initSharedParticipants(configs))
	assert.EqualValues(t, nil, err)

	rs, err := tester.test(1645593180000)
	assert.EqualValues(t, nil, err)
	fmt.Println(rs)
}
