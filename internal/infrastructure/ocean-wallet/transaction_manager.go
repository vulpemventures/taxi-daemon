package oceanwallet

import (
	"context"

	pb "github.com/vulpemventures/ocean/api-spec/protobuf/gen/go/ocean/v1alpha"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	"google.golang.org/grpc"
)

type txManager struct {
	client pb.TransactionServiceClient
}

func newTxManager(conn *grpc.ClientConn) ports.TransactionManager {
	return &txManager{pb.NewTransactionServiceClient(conn)}
}

func (tm *txManager) SelectUtxosForAccount(
	ctx context.Context, account string,
	targetAsset string, targetAmount uint64, strategy ports.Strategy,
) ([]ports.Utxo, uint64, uint64, error) {
	res, err := tm.client.SelectUtxos(ctx, &pb.SelectUtxosRequest{
		AccountName:  account,
		TargetAsset:  targetAsset,
		TargetAmount: targetAmount,
		Strategy:     toStrategyProto(strategy),
	})
	if err != nil {
		return nil, 0, 0, err
	}

	utxos := make([]ports.Utxo, 0, len(res.GetUtxos()))
	for _, u := range res.GetUtxos() {
		utxos = append(utxos, utxo{u})
	}
	change, expirationDate := res.GetChange(), uint64(res.GetExpirationDate())
	return utxos, change, expirationDate, nil
}

func (tm *txManager) EstimateFees(
	ctx context.Context, inputs []ports.Input, outputs []ports.Output,
	millisatsPerByte uint64,
) (uint64, error) {
	res, err := tm.client.EstimateFees(ctx, &pb.EstimateFeesRequest{
		Inputs:  toInputsProto(inputs),
		Outputs: toOutputsProto(outputs),
	})
	if err != nil {
		return 0, err
	}
	return res.GetFeeAmount(), nil
}

func (tm *txManager) CreateTransaction(
	ctx context.Context, inputs []ports.Input, outputs []ports.Output,
) (string, error) {
	res, err := tm.client.CreatePset(ctx, &pb.CreatePsetRequest{
		Inputs:  toInputsProto(inputs),
		Outputs: toOutputsProto(outputs),
	})
	if err != nil {
		return "", err
	}
	return res.GetPset(), nil
}

func (tm *txManager) UpdateTransaction(
	ctx context.Context, psetBase64 string, inputs []ports.Input, outputs []ports.Output,
) (string, error) {
	res, err := tm.client.UpdatePset(ctx, &pb.UpdatePsetRequest{
		Pset:    psetBase64,
		Inputs:  toInputsProto(inputs),
		Outputs: toOutputsProto(outputs),
	})
	if err != nil {
		return "", err
	}
	return res.GetPset(), nil
}

func (tm *txManager) BlindTransaction(
	ctx context.Context, psetBase64 string, lastBlinder bool,
) (string, error) {
	res, err := tm.client.BlindPset(ctx, &pb.BlindPsetRequest{
		Pset:        psetBase64,
		LastBlinder: lastBlinder,
	})
	if err != nil {
		return "", err
	}
	return res.GetPset(), nil
}

func (tm *txManager) SignTransaction(
	ctx context.Context, psetBase64 string, sighash uint32,
) (string, error) {
	res, err := tm.client.SignPset(ctx, &pb.SignPsetRequest{
		Pset:        psetBase64,
		SighashType: sighash,
	})
	if err != nil {
		return "", err
	}
	return res.GetPset(), nil
}

func toStrategyProto(s ports.Strategy) pb.SelectUtxosRequest_Strategy {
	if s == nil {
		return pb.SelectUtxosRequest_STRATEGY_FRAGMENT
	}
	if s.IsBranchBound() {
		return pb.SelectUtxosRequest_STRATEGY_BRANCH_BOUND
	}
	if s.IsFragment() {
		return pb.SelectUtxosRequest_STRATEGY_FRAGMENT
	}
	return pb.SelectUtxosRequest_STRATEGY_UNSPECIFIED
}

func toInputsProto(ins []ports.Input) []*pb.Input {
	inputs := make([]*pb.Input, 0, len(ins))
	for _, in := range ins {
		inputs = append(inputs, &pb.Input{
			Txid:  in.TxID(),
			Index: in.Index(),
		})
	}
	return inputs
}
func toOutputsProto(outs []ports.Output) []*pb.Output {
	outputs := make([]*pb.Output, 0, len(outs))
	for _, out := range outs {
		outputs = append(outputs, &pb.Output{
			Asset:   out.Asset(),
			Amount:  out.Value(),
			Address: out.Address(),
		})
	}
	return outputs
}
