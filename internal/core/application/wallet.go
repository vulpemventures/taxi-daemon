package application

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/psetv2"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

const (
	LBTCAccount   = "lbtc"
	PayoutAccount = "payout"
)

type Wallet interface {
	UpdatePartialWithFees(
		ctx context.Context, partial string, millisatPerByte uint64,
	) (updatedPartial string, expirationDate uint64, err error)
	CreatePartialWithFeesAndPayout(
		ctx context.Context, price *domain.Price,
		estimatedSize, millisatPerByte uint64,
	) (
		partial string, selectedUtxos []ports.Utxo,
		expirationDate, feeAmount, payourAmount uint64, err error,
	)
	PartialContainsKnownPayout(
		partial string, knownAddresses []string,
	) (bool, error)
	PartialContainsKnownUtxos(
		partial string, utxos []ports.Utxo,
	) (bool, error)
	OnUtxosUnlocked(handler func(utxos []ports.Utxo))
	OnUtxosSpent(handler func(utxos []ports.Utxo))
}

type wallet struct {
	oceanSvc  ports.OceanWallet
	lbtcAsset string
	network   *network.Network

	utxosUnlockedHandler func([]ports.Utxo)
	utxosSpentHandler    func([]ports.Utxo)
}

func NewWallet(oceanSvc ports.OceanWallet) (Wallet, error) {
	ctx := context.Background()
	status, err := oceanSvc.WalletManager().Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ocean wallet status: %s", err)
	}
	if !status.IsInitialized() || !status.IsUnlocked() || !status.IsSynced() {
		return nil, fmt.Errorf(
			"ocean wallet must be initialized, unlocked and in sync, got %+v", status,
		)
	}
	info, err := oceanSvc.WalletManager().GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ocean wallet info: %s", err)
	}

	var lbtcAccountExists, payoutAccountExists bool
	for _, account := range info.Accounts() {
		if account.Name() == LBTCAccount {
			lbtcAccountExists = true
		}
		if account.Name() == PayoutAccount {
			payoutAccountExists = true
		}
	}

	if !lbtcAccountExists {
		if _, _, err := oceanSvc.AccountManager().CreateAccount(
			ctx, LBTCAccount,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to create %s account: %s", LBTCAccount, err,
			)
		}
	}
	if !payoutAccountExists {
		if _, _, err := oceanSvc.AccountManager().CreateAccount(
			ctx, PayoutAccount,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to create %s account: %s", PayoutAccount, err,
			)
		}
	}

	net := &network.Liquid
	if info.Network().IsTestnet() {
		net = &network.Testnet
	}
	if info.Network().IsRegtest() {
		net = &network.Regtest
		net.AssetID = info.NativeAsset()
	}

	w := &wallet{oceanSvc, info.NativeAsset(), net, nil, nil}
	go w.startListeningToUxoNotifications()
	return w, nil
}

func (w *wallet) UpdatePartialWithFees(
	ctx context.Context, psetBase64 string, millisatPerByte uint64,
) (string, uint64, error) {
	ptx, err := psetv2.NewPsetFromBase64(psetBase64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid partial transaction: %s", err)
	}

	ins := make([]ports.Input, 0, ptx.Global.InputCount)
	for _, in := range ptx.Inputs {
		ins = append(ins, input{in.PreviousTxid, in.PreviousTxIndex})
	}

	outs := make([]ports.Output, 0, ptx.Global.OutputCount)
	for _, out := range ptx.Outputs {
		addr := w.getOutputAddress(out.Script, out.BlindingPubkey)
		outs = append(outs, output{out.Asset, out.Value, addr})
	}

	dummyFeeAmount, err := w.oceanSvc.TransactionManager().EstimateFees(
		ctx, ins, outs, millisatPerByte,
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to estimate fee amount for tx: %s", err)
	}

	dummyFeeAmount = roundUpAmount(dummyFeeAmount)
	rawLBTCAsset, _ := elementsutil.AssetHashToBytes(w.lbtcAsset)

	insToAdd, change, expirationDate, err := w.selectLBTCUtxos(ctx, dummyFeeAmount)
	if err != nil {
		return "", 0, fmt.Errorf("failed to select %s utxos: %s", LBTCAccount, err)
	}
	outsToAdd := make([]ports.Output, 0)
	if change > 0 {
		addr, err := w.getLBTCChangeAddress(ctx)
		if err != nil {
			return "", 0, fmt.Errorf(
				"failed to generate %s change address: %s", LBTCAccount, err,
			)
		}
		outsToAdd = append(outsToAdd, output{
			asset:   rawLBTCAsset,
			value:   change,
			address: addr,
		})
	}

	ins = append(ins, utxos(insToAdd).keys()...)
	outs = append(outs, outsToAdd...)
	feeAmount, err := w.oceanSvc.TransactionManager().EstimateFees(
		ctx, ins, outs, millisatPerByte,
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to estimate fee amount for tx: %s", err)
	}

	delta := int(dummyFeeAmount) - int(feeAmount)
	if delta == 0 {
		outsToAdd = make([]ports.Output, 0)
	}
	if delta > 0 {
		out := outsToAdd[0].(output)
		out.value += uint64(delta)
		outsToAdd[0] = out
	}
	outsToAdd = append(outsToAdd, output{
		asset: rawLBTCAsset,
		value: feeAmount,
	})

	updatedPset, err := w.oceanSvc.TransactionManager().UpdateTransaction(
		ctx, psetBase64, utxos(insToAdd).keys(), outsToAdd,
	)
	if err != nil {
		return "", 0, fmt.Errorf(
			"failed to add %s ins and outs to partial transaciton: %s",
			LBTCAccount, err,
		)
	}

	// Tx needs to be blinded if it cointans LBTC change output.
	if len(outsToAdd) > 1 {
		updatedPset, err = w.oceanSvc.TransactionManager().BlindTransaction(ctx, updatedPset, false)
		if err != nil {
			return "", 0, fmt.Errorf("failed to blind LBTC outs: %s", err)
		}
	}

	sighash := uint32(txscript.SigHashAll)
	signedPset, err := w.oceanSvc.TransactionManager().SignTransaction(
		ctx, updatedPset, sighash,
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign partial transaction: %s", err)
	}

	return signedPset, expirationDate, nil
}

func (w *wallet) CreatePartialWithFeesAndPayout(
	ctx context.Context, price *domain.Price,
	estimatedSize, millisatPerByte uint64,
) (string, []ports.Utxo, uint64, uint64, uint64, error) {
	dummyFeeAmount := uint64(
		float64(estimatedSize) * float64(millisatPerByte) / math.Pow10(3),
	)

	insToAdd, change, expirationDate, err := w.selectLBTCUtxos(
		ctx, dummyFeeAmount,
	)
	if err != nil {
		return "", nil, 0, 0, 0, fmt.Errorf("failed to select %s utxos: %s", LBTCAccount, err)
	}

	_, payoutAddr, err := w.getPayoutAddress(ctx)
	if err != nil {
		return "", nil, 0, 0, 0, fmt.Errorf(
			"failed to generate %s address: %s", PayoutAccount, err,
		)
	}

	asset := price.AssetHash
	rawAsset, _ := elementsutil.AssetHashToBytes(asset)
	rawLBTCAsset, _ := elementsutil.AssetHashToBytes(w.lbtcAsset)
	// We don't want any change in return, hence the total fee amount is just
	// the sum of target amount and eventual change.
	feeAmount := dummyFeeAmount + change
	payoutAmount := getPayoutAmount(price, feeAmount)

	outsToAdd := []ports.Output{
		output{rawAsset, payoutAmount, payoutAddr},
		output{rawLBTCAsset, feeAmount, ""},
	}

	partial, err := w.oceanSvc.TransactionManager().CreateTransaction(
		ctx, utxos(insToAdd).keys(), outsToAdd,
	)
	if err != nil {
		return "", nil, 0, 0, 0, fmt.Errorf("failed to create partial transaction: %s", err)
	}

	sighash := uint32(txscript.SigHashSingle | txscript.SigHashAnyOneCanPay)
	signedPartial, err := w.oceanSvc.TransactionManager().SignTransaction(
		ctx, partial, sighash,
	)
	if err != nil {
		return "", nil, 0, 0, 0, fmt.Errorf("failed to sign partial transaction: %s", err)
	}

	return signedPartial, insToAdd, expirationDate, feeAmount, payoutAmount, nil
}

func (w *wallet) PartialContainsKnownPayout(
	partial string, knownAddresses []string,
) (bool, error) {
	ptx, err := psetv2.NewPsetFromBase64(partial)
	if err != nil {
		return false, err
	}

	for _, out := range ptx.Outputs {
		if len(out.Script) > 0 {
			addr := w.getOutputAddress(out.Script, out.BlindingPubkey)
			for _, knownAddr := range knownAddresses {
				if addr == knownAddr {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (w *wallet) PartialContainsKnownUtxos(
	partial string, knownUtxos []ports.Utxo,
) (bool, error) {
	ptx, err := psetv2.NewPsetFromBase64(partial)
	if err != nil {
		return false, err
	}

	for _, in := range ptx.Inputs {
		for _, knownUtxo := range knownUtxos {
			if knownUtxo.TxID() == elementsutil.TxIDFromBytes(in.PreviousTxid) &&
				knownUtxo.Index() == in.PreviousTxIndex {
				return true, nil
			}
		}
	}
	return false, nil
}

func (w *wallet) OnUtxosUnlocked(handler func(utxos []ports.Utxo)) {
	if w.utxosUnlockedHandler == nil {
		w.utxosUnlockedHandler = handler
	}
}

func (w *wallet) OnUtxosSpent(handler func(utxos []ports.Utxo)) {
	if w.utxosSpentHandler == nil {
		w.utxosSpentHandler = handler
	}
}

func (w *wallet) startListeningToUxoNotifications() {
	for notify := range w.oceanSvc.NotificationManager().UtxoChannel() {
		eventType := notify.EventType()
		if eventType.IsSpent() {
			if w.utxosSpentHandler != nil {
				w.utxosSpentHandler(notify.Utxos())
			}
		}
		if eventType.IsUnlocked() {
			if w.utxosUnlockedHandler != nil {
				w.utxosUnlockedHandler(notify.Utxos())
			}
		}
	}
}

func (w *wallet) getOutputAddress(script, blindingKey []byte) string {
	scriptType := address.GetScriptType(script)
	var addr string
	switch scriptType {
	case address.P2PkhScript, address.P2ShScript:
		version := w.network.PubKeyHash
		if scriptType == address.P2Sh {
			version = w.network.ScriptHash
		}
		addr = address.ToBase58(&address.Base58{
			Version: version,
			Data:    script,
		})
	default:
		addr, _ = address.ToBech32(&address.Bech32{
			Prefix:  w.network.Bech32,
			Version: script[0],
			Program: script[2:],
		})
	}

	if len(blindingKey) == 0 {
		return addr
	}

	addr, _ = address.ToConfidential(&address.ConfidentialAddress{
		Address:     addr,
		BlindingKey: blindingKey,
	})
	return addr
}

func (w *wallet) selectLBTCUtxos(
	ctx context.Context, targetAmount uint64,
) ([]ports.Utxo, uint64, uint64, error) {
	return w.oceanSvc.TransactionManager().SelectUtxosForAccount(
		ctx, LBTCAccount, w.lbtcAsset, targetAmount, nil,
	)
}

func (w *wallet) getLBTCChangeAddress(ctx context.Context) (string, error) {
	addr, err := w.oceanSvc.AccountManager().DeriveChangeAddressForAccount(
		ctx, LBTCAccount, 1,
	)
	if err != nil {
		return "", err
	}
	return addr[0], nil
}

func (w *wallet) getPayoutAddress(ctx context.Context) (string, string, error) {
	addr, err := w.oceanSvc.AccountManager().DeriveAddressesForAccount(
		ctx, PayoutAccount, 1,
	)
	if err != nil {
		return "", "", err
	}

	ca, _ := address.FromConfidential(addr[0])
	return addr[0], ca.Address, nil
}

func roundUpAmount(amount uint64) uint64 {
	roundedAmount := float64(amount)
	orderOfMagnitude := 0
	for roundedAmount > 10 {
		roundedAmount /= 10
		orderOfMagnitude++
	}
	roundedAmount = math.Ceil(roundedAmount)
	roundedAmount *= math.Pow10(orderOfMagnitude)
	return uint64(roundedAmount)
}

func getPayoutAmount(price *domain.Price, feeAmount uint64) uint64 {
	feeAmountAsDecimal := decimal.NewFromInt(int64(feeAmount))
	assetAmountAsDecimal := price.Amount.Mul(feeAmountAsDecimal)

	spreadRate := decimal.NewFromFloat(float64(price.BasisPoint) * 0.01 / 100)
	serviceFeeAmountAsDecimal := assetAmountAsDecimal.Mul(spreadRate)

	payoutAmount := assetAmountAsDecimal.Add(serviceFeeAmountAsDecimal).
		BigInt().Uint64()
	return payoutAmount
}

type portableInput struct {
	in psetv2.Input
}

func (i portableInput) TxID() string {
	return elementsutil.TxIDFromBytes(i.in.PreviousTxid)
}

func (i portableInput) Index() uint32 {
	return i.in.PreviousTxIndex
}

func (i portableInput) Asset() string {
	prevout := i.in.GetUtxo()
	if prevout.IsConfidential() {
		return hex.EncodeToString(prevout.Asset)
	}
	return elementsutil.AssetHashFromBytes(prevout.Asset)
}

func (i portableInput) Value() uint64 {
	prevout := i.in.GetUtxo()
	if prevout.IsConfidential() {
		return 0
	}
	val, _ := elementsutil.ValueFromBytes(prevout.Value)
	return val
}

type input struct {
	txid  []byte
	index uint32
}

func (i input) TxID() string {
	return elementsutil.TxIDFromBytes(i.txid)
}
func (i input) Index() uint32 {
	return i.index
}

type output struct {
	asset   []byte
	value   uint64
	address string
}

func (o output) Asset() string {
	return elementsutil.AssetHashFromBytes(o.asset)
}

func (o output) Value() uint64 {
	return o.value
}

func (o output) Address() string {
	return o.address
}

type utxos []ports.Utxo

func (u utxos) keys() []ports.UtxoKey {
	keys := make([]ports.UtxoKey, 0, len(u))
	for _, utxo := range u {
		keys = append(keys, utxo)
	}
	return keys
}
