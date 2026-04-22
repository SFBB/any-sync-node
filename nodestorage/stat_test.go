package nodestorage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcMedian(t *testing.T) {
	t.Run("empty slice returns zero without panic", func(t *testing.T) {
		assert.Equal(t, 0.0, calcMedian(nil))
		assert.Equal(t, 0.0, calcMedian([]int{}))
	})
	t.Run("odd length", func(t *testing.T) {
		assert.Equal(t, 3.0, calcMedian([]int{1, 2, 3, 4, 5}))
	})
	t.Run("even length", func(t *testing.T) {
		assert.Equal(t, 2.5, calcMedian([]int{1, 2, 3, 4}))
	})
}
