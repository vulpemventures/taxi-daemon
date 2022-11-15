package application

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/ocean/v1"

	"github.com/btcsuite/btcd/btcec/v2"
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
		userUnblindedInput []UnblindedInput,
	) (updatedPartial string, expirationDate uint64, err error)
	CreatePartialWithFeesAndPayout(
		ctx context.Context, price *domain.Price,
		estimatedSize, millisatPerByte uint64,
	) (
		partial string, selectedUtxos []ports.Utxo,
		expirationDate, feeAmount, payourAmount uint64, err error,
	)
	PartialContainsKnownPayout(
		partial string, knownScripts []string,
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
	userUnblindedInput []UnblindedInput,
) (string, uint64, error) {
	unblindedInputs := make([]*pb.UnblindedInput, 0, len(userUnblindedInput))
	for _, v := range userUnblindedInput {
		unblindedInputs = append(unblindedInputs, &pb.UnblindedInput{
			Index:         v.Index,
			Asset:         v.Asset,
			Amount:        v.Amount,
			AssetBlinder:  v.AssetBlinder,
			AmountBlinder: v.AmountBlinder,
		})
	}

	ptx, err := psetv2.NewPsetFromBase64(psetBase64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid partial transaction: %s", err)
	}

	if int(ptx.Global.InputCount) != len(userUnblindedInput) {
		return "", 0, fmt.Errorf("invalid partial transaction: number of unblinded inputs must match number of tx inputs")
	}

	ins := make([]ports.Input, 0, ptx.Global.InputCount)
	for _, in := range ptx.Inputs {
		ins = append(ins, input{
			in.PreviousTxid, in.PreviousTxIndex, in.GetUtxo().Script,
		})
	}

	outs := make([]ports.Output, 0, ptx.Global.OutputCount)
	for _, out := range ptx.Outputs {
		addr := w.getOutputAddress(out.Script, out.BlindingPubkey)
		outs = append(outs, output{out.Asset, out.Value, addr})
	}

	rawLBTCAsset, _ := elementsutil.AssetHashToBytes(w.lbtcAsset)
	outsToAdd := make([]ports.Output, 0)
	// Since our inputs are confidential, when the user tx is full unconf we add
	// a dummy output to make sure at least one output is blinded
	if !ptx.NeedsBlinding() {
		outsToAdd = append(outsToAdd, output{asset: rawLBTCAsset})
	}

	dummyFeeAmount, err := w.oceanSvc.TransactionManager().EstimateFees(
		ctx, ins, outs, millisatPerByte,
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to estimate fee amount for tx: %s", err)
	}

	dummyFeeAmount = roundUpFeeAmount(dummyFeeAmount)
	insToAdd, change, expirationDate, err := w.selectLBTCUtxos(ctx, dummyFeeAmount)
	if err != nil {
		return "", 0, fmt.Errorf("failed to select %s utxos: %s", LBTCAccount, err)
	}
	outsToAdd = append(outsToAdd, output{
		asset: rawLBTCAsset,
		value: dummyFeeAmount + change,
	})

	updatedPset, err := w.oceanSvc.TransactionManager().UpdateTransaction(
		ctx, psetBase64, utxos(insToAdd).toInputs(), outsToAdd,
	)
	if err != nil {
		return "", 0, fmt.Errorf(
			"failed to add %s ins and outs to partial transaciton: %s",
			LBTCAccount, err,
		)
	}

	blindedPset, err := w.oceanSvc.TransactionManager().BlindTransaction(
		ctx, updatedPset, true, unblindedInputs)
	if err != nil {
		return "", 0, fmt.Errorf("failed to blind partial transaction: %s", err)
	}

	sighash := uint32(txscript.SigHashAll)
	signedPset, err := w.oceanSvc.TransactionManager().SignTransaction(
		ctx, blindedPset, sighash,
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
		ctx, utxos(insToAdd).toInputs(), outsToAdd,
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
	partial string, knownScripts []string,
) (bool, error) {
	ptx, err := psetv2.NewPsetFromBase64(partial)
	if err != nil {
		return false, err
	}

	for _, out := range ptx.Outputs {
		if len(out.Script) > 0 {
			outScript := hex.EncodeToString(out.Script)
			for _, v := range knownScripts {
				if v == outScript {
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
		scriptHash := script[3 : len(script)-2]
		if scriptType == address.P2Sh {
			version = w.network.ScriptHash
			scriptHash = script[2 : len(script)-1]
		}
		addr = address.ToBase58(&address.Base58{
			Version: version,
			Data:    scriptHash,
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

	addr, _ = address.ToConfidential(&address.AddressInfo{
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

// roundUpFeeAmount rounds up the given amount to the next hundred/thousand.
// based on its order of magnitude.
func roundUpFeeAmount(amount uint64) uint64 {
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

type input struct {
	txid   []byte
	index  uint32
	script []byte
}

func (i input) TxID() string {
	return elementsutil.TxIDFromBytes(i.txid)
}
func (i input) Index() uint32 {
	return i.index
}
func (i input) Script() string {
	return hex.EncodeToString(i.script)
}

// TODO: compute correct size based on utxo script type
func (i input) ScriptSigSize() int {
	return 0
}

// TODO: compute correct size based on utxo script type
func (i input) WitnessSize() int {
	return 0
}

type output struct {
	asset   []byte
	value   uint64
	address string
}

func (o output) Asset() string {
	if len(o.asset) == 33 {
		return elementsutil.AssetHashFromBytes(o.asset)
	}
	return hex.EncodeToString(elementsutil.ReverseBytes(o.asset))
}
func (o output) Value() uint64 {
	return o.value
}
func (o output) Script() string {
	if o.address != "" {
		script, _ := address.ToOutputScript(o.address)
		return hex.EncodeToString(script)
	}
	// is fee out
	if o.value > 0 {
		return ""
	}
	// is dummy conf output, let's return OP_RETURN script
	return hex.EncodeToString([]byte{txscript.OP_RETURN})
}
func (o output) BlindingKey() string {
	if o.address != "" {
		info, _ := address.FromConfidential(o.address)
		// is unconfidential
		if info == nil {
			return ""
		}
		return hex.EncodeToString(info.BlindingKey)
	}
	// is fee out
	if o.value > 0 {
		return ""
	}
	// is dummy conf output, let's return an ephemeral pubkey
	key, _ := btcec.NewPrivateKey()
	return hex.EncodeToString(key.PubKey().SerializeCompressed())
}

type utxos []ports.Utxo

func (u utxos) toInputs() []ports.Input {
	ins := make([]ports.Input, 0, len(u))
	for _, utxo := range u {
		txid, _ := elementsutil.TxIDToBytes(utxo.TxID())
		ins = append(ins, input{txid, utxo.Index(), utxo.Script()})
	}
	return ins
}
