package fees

// CalculateFeeIncrease estimates the increase of bitcoin fees depending of how much inputs and outputs we add to a transaction.
func CalculateFeeIncrease(numberOfInputs int, numberOfOuputs int, satPerByte float32) uint64 {
	size := 149*numberOfInputs + (32+32)*numberOfOuputs
	fees := uint64(float32(size) * satPerByte)
	// spread := uint64(float32(fees) * rate)

	return fees
}
