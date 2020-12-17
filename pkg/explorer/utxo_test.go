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
			utxo: Utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				Asset:           "",
				Value:           0,
			},
			expected: false,
		},
		{
			utxo: Utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				Asset:           "",
				Value:           0,
			},
			expected: true,
		},
		{
			utxo: Utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				Asset:           "notnull",
				Value:           10,
			},
			expected: false,
		},
		{
			utxo: Utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				Asset:           "notnull",
				Value:           10000,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.utxo.IsConfidential())
	}
}
