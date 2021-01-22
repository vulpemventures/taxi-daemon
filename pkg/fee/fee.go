package fee

import "math"

// EstimateTxFee is used to compute the fee amount without the whole transaction
func EstimateTxFee(numInputs, numOutputs int, withChange bool, milliSatsPerBytes int) uint64 {
	satsPerBytes := float64(milliSatsPerBytes) / 1000
	baseSize := calcTxSize(
		false, withChange,
		numInputs, numOutputs,
	)
	totalSize := calcTxSize(
		true, withChange,
		numInputs, numOutputs,
	)
	weight := baseSize*3 + totalSize
	vsize := (weight + 3) / 4

	size := uint64(float64(vsize) * satsPerBytes)

	return size
}

func calcTxSize(withWitness, withChange bool, numInputs, numOutputs int) int {
	inputsSize := calcInputsSize(withWitness, numInputs)
	outputsSize := calcOutputsSize(withWitness, withChange, numOutputs)

	return 9 +
		varIntSerializeSize(uint64(numInputs)) +
		varIntSerializeSize(uint64(numOutputs)) +
		inputsSize +
		outputsSize
}

func calcInputsSize(withWitness bool, numInputs int) int {
	// prevout hash & index for each input
	size := (32 + 8) * numInputs
	if withWitness {
		// scriptsig + pubkey per each witness of all inputs
		size += numInputs * (72 + 33)
	}
	return size
}

func calcOutputsSize(withWitness, withChange bool, numOutputs int) int {
	// assetcommitment, valuecommitment, nonce
	baseOutputSize := 33 + 33 + 33
	size := baseOutputSize * numOutputs
	if withWitness {
		// rangeproof & surjection proof
		size += (4174 + 67) * numOutputs
	}

	if withChange {
		size += baseOutputSize
		if withWitness {
			size += 4174 + 67
		}
	}

	// fee asset, fee nonce, fee amount
	size += 33 + 1 + 9
	return size
}

func varIntSerializeSize(val uint64) int {
	// The value is small enough to be represented by itself, so it's
	// just 1 byte.
	if val < 0xfd {
		return 1
	}

	// Discriminant 1 byte plus 2 bytes for the uint16.
	if val <= math.MaxUint16 {
		return 3
	}

	// Discriminant 1 byte plus 4 bytes for the uint32.
	if val <= math.MaxUint32 {
		return 5
	}

	// Discriminant 1 byte plus 8 bytes for the uint64.
	return 9
}
