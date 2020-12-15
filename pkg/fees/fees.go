package fees

import "github.com/shopspring/decimal"

// CalculateFeeIncrease estimates the increase of bitcoin fees depending of how much inputs and outputs we add to a transaction.
func CalculateFeeIncrease(numberOfInputs int, numberOfOuputs int, satPerByte decimal.Decimal) uint64 {
	size := 149*numberOfInputs + (32+32)*numberOfOuputs
	fees := uint64(decimal.NewFromInt(int64(size)).Mul(satPerByte).IntPart())
	// spread := uint64(float32(fees) * rate)

	return fees
}
