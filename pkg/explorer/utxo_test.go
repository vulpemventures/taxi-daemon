package explorer

import (
	"testing"

	"github.com/tj/assert"
)

func TestIsConfidential(t *testing.T) {
	tests := []struct {
		utxo     Utxo
		expected bool
	}{
		{
			utxo: utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				UnblindedAsset:  "",
				UnblindedValue:  0,
			},
			expected: false,
		},
		{
			utxo: utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				UnblindedAsset:  "",
				UnblindedValue:  0,
			},
			expected: true,
		},
		{
			utxo: utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				UnblindedAsset:  "notnull",
				UnblindedValue:  10,
			},
			expected: false,
		},
		{
			utxo: utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				UnblindedAsset:  "notnull",
				UnblindedValue:  10000,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.utxo.IsConfidential())
	}
}
