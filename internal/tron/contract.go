package tron

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/factoryabi"

	tronclient "github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
)

type ContractService struct {
	tron *Client
	abi  abi.ABI
}

func NewContractService(tronClient *Client, parsed abi.ABI) *ContractService {
	return &ContractService{tron: tronClient, abi: parsed}
}

func (s *ContractService) GetFactoryInfo(ctx context.Context, networkName string) (*factoryabi.FactoryInfo, error) {
	factoryAddr, _, err := s.tron.FactoryAddress(networkName)
	if err != nil {
		return nil, err
	}
	grpc, _, err := s.tron.GRPC(networkName)
	if err != nil {
		return nil, err
	}
	if err := s.assertDeployed(ctx, grpc, factoryAddr); err != nil {
		return nil, err
	}

	mother, err := s.callAddress(ctx, grpc, factoryAddr, "motherWallet")
	if err != nil {
		return nil, err
	}
	relayer, err := s.callAddress(ctx, grpc, factoryAddr, "relayer")
	if err != nil {
		return nil, err
	}
	owner, err := s.callAddress(ctx, grpc, factoryAddr, "owner")
	if err != nil {
		return nil, err
	}
	impl, err := s.callAddress(ctx, grpc, factoryAddr, "implementation")
	if err != nil {
		return nil, err
	}

	return &factoryabi.FactoryInfo{
		FactoryAddress: factoryAddr,
		MotherWallet:   mother,
		Relayer:        relayer,
		Owner:          owner,
		Implementation: impl,
	}, nil
}

func (s *ContractService) Call(ctx context.Context, networkName, functionName string, rawArgs map[string]string) (*factoryabi.CallResult, error) {
	fn, ok := factoryabi.FindFunction(functionName)
	if !ok {
		return nil, apperror.BadRequest("Unknown function: " + functionName)
	}

	factoryAddr, _, err := s.tron.FactoryAddress(networkName)
	if err != nil {
		return nil, err
	}
	grpc, _, err := s.tron.GRPC(networkName)
	if err != nil {
		return nil, err
	}
	if err := s.assertDeployed(ctx, grpc, factoryAddr); err != nil {
		return nil, err
	}

	args, err := parseArgs(fn, rawArgs)
	if err != nil {
		return nil, err
	}

	if fn.Type == "read" {
		result, err := s.staticCall(ctx, grpc, factoryAddr, functionName, args...)
		if err != nil {
			return nil, err
		}
		return &factoryabi.CallResult{FunctionName: functionName, Type: "read", Result: result}, nil
	}

	role := "owner"
	if fn.Role == "relayer" {
		role = "relayer"
	}
	key, from, _, err := s.tron.PrivateKey(networkName, role)
	if err != nil {
		return nil, err
	}

	txHash, energy, blockNum, err := s.transact(ctx, grpc, key, from, factoryAddr, functionName, args...)
	if err != nil {
		return nil, err
	}
	return &factoryabi.CallResult{
		FunctionName: functionName,
		Type:         "write",
		TxHash:       txHash,
		BlockNumber:  blockNum,
		GasUsed:      strconv.FormatInt(energy, 10),
	}, nil
}

func (s *ContractService) assertDeployed(ctx context.Context, grpc *tronclient.GrpcClient, factoryAddr string) error {
	_, err := grpc.GetContractABICtx(ctx, factoryAddr)
	if err != nil {
		return apperror.BadRequest(fmt.Sprintf(
			"No contract at %s. Check FACTORY_ADDRESS — it may be a wallet address, not the deployed factory.",
			factoryAddr,
		))
	}
	return nil
}

func (s *ContractService) staticCall(ctx context.Context, grpc *tronclient.GrpcClient, contractAddr, method string, args ...interface{}) (interface{}, error) {
	data, err := s.abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	tx, err := grpc.TriggerConstantContractWithDataCtx(ctx, "", contractAddr, data)
	if err != nil {
		return nil, err
	}
	if tx.Result != nil && tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}
	if len(tx.ConstantResult) == 0 {
		return nil, fmt.Errorf("empty constant result for %s", method)
	}
	if reason, err := abi.UnpackRevert(tx.ConstantResult[0]); err == nil && reason != "" {
		return nil, fmt.Errorf("%s", reason)
	}
	results, err := s.abi.Unpack(method, tx.ConstantResult[0])
	if err != nil {
		return nil, err
	}
	return formatOutputs(s.abi, method, results)
}

func (s *ContractService) callAddress(ctx context.Context, grpc *tronclient.GrpcClient, contractAddr, method string) (string, error) {
	v, err := s.staticCall(ctx, grpc, contractAddr, method)
	if err != nil {
		return "", err
	}
	switch t := v.(type) {
	case string:
		return t, nil
	case common.Address:
		return FormatETHAddress(t), nil
	default:
		return "", fmt.Errorf("unexpected %s result type", method)
	}
}

func (s *ContractService) transact(ctx context.Context, grpc *tronclient.GrpcClient, key *ecdsa.PrivateKey, from, contractAddr, method string, args ...interface{}) (string, int64, uint64, error) {
	data, err := s.abi.Pack(method, args...)
	if err != nil {
		return "", 0, 0, err
	}
	txExt, err := grpc.TriggerContractWithDataCtx(ctx, from, contractAddr, data, defaultFeeLimit, 0, "", 0)
	if err != nil {
		return "", 0, 0, err
	}
	if txExt.Result != nil && txExt.Result.Code > 0 {
		return "", 0, 0, fmt.Errorf("%s", string(txExt.Result.Message))
	}

	signed, err := signTransaction(key, txExt)
	if err != nil {
		return "", 0, 0, err
	}
	if _, err := grpc.BroadcastCtx(ctx, signed); err != nil {
		return "", 0, 0, err
	}

	txID := hex.EncodeToString(txExt.Txid)
	info, err := waitMined(ctx, grpc, txExt.Txid)
	if err != nil {
		return txID, 0, 0, err
	}
	var energy int64
	if info.Receipt != nil {
		energy = info.Receipt.EnergyUsageTotal
	}
	return txID, energy, uint64(info.BlockNumber), nil
}

func signTransaction(key *ecdsa.PrivateKey, txExt *api.TransactionExtention) (*core.Transaction, error) {
	sig, err := signer.NewPrivateKeySigner(key)
	if err != nil {
		return nil, err
	}
	return sig.Sign(txExt.Transaction)
}

func parseArgs(fn factoryabi.FunctionDef, raw map[string]string) ([]interface{}, error) {
	if raw == nil {
		raw = map[string]string{}
	}
	out := make([]interface{}, 0, len(fn.Inputs))
	for _, input := range fn.Inputs {
		value := strings.TrimSpace(raw[input.Name])
		if value == "" {
			return nil, apperror.BadRequest("Missing parameter: " + input.Label)
		}
		switch input.Type {
		case "uint256":
			n := new(big.Int)
			if _, ok := n.SetString(value, 10); !ok {
				return nil, apperror.BadRequest("Invalid uint256: " + input.Label)
			}
			out = append(out, n)
		case "address":
			eth, err := ToETHAddress(value)
			if err != nil {
				return nil, err
			}
			out = append(out, eth)
		default:
			out = append(out, value)
		}
	}
	return out, nil
}

func formatOutputs(parsed abi.ABI, method string, values []interface{}) (interface{}, error) {
	m, ok := parsed.Methods[method]
	if !ok {
		if len(values) == 1 {
			return formatValue(values[0]), nil
		}
		return values, nil
	}
	if len(m.Outputs) == 0 {
		return nil, nil
	}
	if len(values) == 1 {
		return formatValue(values[0]), nil
	}
	named := map[string]interface{}{}
	for i, out := range m.Outputs {
		key := out.Name
		if key == "" {
			key = fmt.Sprintf("%d", i)
		}
		named[key] = formatValue(values[i])
	}
	return named, nil
}

func formatValue(v interface{}) interface{} {
	switch t := v.(type) {
	case *big.Int:
		return t.String()
	case common.Address:
		return FormatETHAddress(t)
	case []byte:
		if len(t) >= 20 {
			return FormatETHAddress(common.BytesToAddress(t[len(t)-20:]))
		}
		return common.BytesToAddress(t).Hex()
	default:
		return v
	}
}
