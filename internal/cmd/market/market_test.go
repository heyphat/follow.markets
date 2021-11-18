package market

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Market(t *testing.T) {
	path := "./../../../configs/dev_configs.json"
	market, err := NewMarket(&path)
	assert.EqualValues(t, nil, err)
	fmt.Println(market)
}
