package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	unconfidentialAddress = "ert1qj2cs63xx2e60qst6ddpxarq43yl5udrs676zv7"
)

func TestGetUnspents(t *testing.T) {
	// faucet the address to create unspent
	_, err := faucet(unconfidentialAddress, explorerEndpoint)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(3 * time.Second)

	// test :
	explorerSvc := NewEsplorerService(explorerEndpoint)
	unspents, err := explorerSvc.GetUnspents(unconfidentialAddress)

	assert.Nil(t, err)
	assert.Equal(t, true, len(unspents) > 0)
}
