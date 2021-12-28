package market

import (
	"testing"

	"follow.markets/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Notifier(t *testing.T) {
	path := "./../../../configs/deploy.configs.json"
	configs, err := config.NewConfigs(&path)
	assert.EqualValues(t, nil, err)

	notifier, err := newNotifier(initSharedParticipants(configs), configs)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, false, notifier.connected)

	notifier.notify("test")
}
