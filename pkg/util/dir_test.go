package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Dir(t *testing.T) {
	path := "../../configs/signals/"
	files, err := IOReadDir(path)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1, len(files))
}
