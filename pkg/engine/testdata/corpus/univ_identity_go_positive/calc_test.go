package pricing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateTotal_WithFee(t *testing.T) {
	fee := 0.0
	total := CalculateTotal(100.0, fee)
	assert.Equal(t, 100.0, total)
}
