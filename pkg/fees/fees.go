package fees

//Breakdown defines the response of the Calculate method
type Breakdown struct {
	Size       uint64
	SatPerByte float32
	Fee        uint64
	Rate       float32
	Spread     uint64
	Total      uint64
}

const defaultSatPerByte float32 = 0.1
const defaultRate float32 = 0.1

// CalculateWitness estimates the size and bitcoin fees (both network and for the topup service) of a segwit transaction
func CalculateWitness(vin int, vout int, satPerByte float32, rate float32) (*Breakdown, error) {

	size := 12 + (149*(vin) + ((32 + 32) * (vout)))

	if satPerByte < defaultSatPerByte {
		satPerByte = defaultSatPerByte
	}
	if rate < defaultRate {
		rate = defaultRate
	}

	fees := uint64(float32(size) * satPerByte)
	spread := uint64(float32(fees) * rate)
	total := fees + spread

	return &Breakdown{
		Size:       uint64(size),
		SatPerByte: satPerByte,
		Fee:        fees,
		Rate:       rate,
		Spread:     spread,
		Total:      total,
	}, nil
}
