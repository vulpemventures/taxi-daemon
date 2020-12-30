package application

import "github.com/tdex-network/tdex-daemon/pkg/explorer"

// BalanceInfo defines a balance for a given asset
type BalanceInfo struct {
	totalBalance       uint64
	confirmedBalance   uint64
	unconfirmedBalance uint64
}

func utxosToBalanceInfos(utxos []explorer.Utxo) map[string]BalanceInfo {
	resultMap := make(map[string]BalanceInfo)
	for _, utxo := range utxos {
		value := utxo.Value()
		isConfirmed := utxo.IsConfirmed()

		// asset already known
		if balanceInfo, ok := resultMap[utxo.Asset()]; ok {
			balanceInfo.totalBalance += value
			if isConfirmed {
				balanceInfo.confirmedBalance += value
				continue
			}

			balanceInfo.unconfirmedBalance += value
			continue
		}

		// new asset = new balance
		balanceInfo := BalanceInfo{
			totalBalance:       value,
			confirmedBalance:   0,
			unconfirmedBalance: 0,
		}

		if isConfirmed {
			balanceInfo.confirmedBalance = value
		} else {
			balanceInfo.unconfirmedBalance = value
		}

		resultMap[utxo.Asset()] = balanceInfo
	}

	return resultMap
}
