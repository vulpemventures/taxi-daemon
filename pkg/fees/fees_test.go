package fees

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculate(t *testing.T) {
	b, err := CalculateWitness(2, 4, 0.1, 0.1)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, b.Size, uint64(566), "Size calcuation is wrong")
}
