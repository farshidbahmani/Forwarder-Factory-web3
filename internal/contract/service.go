package contract

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/factoryabi"
	"forwarder-factory/internal/network"
	"forwarder-factory/internal/tron"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	chain *blockchain.Client
	tron  *tron.ContractService
	abi   abi.ABI
}

func NewService(chain *blockchain.Client, tronClient *tron.Client) (*Service, error) {
	_, parsedEVM, err := blockchain.LoadFactoryArtifact()
	if err != nil {
		return nil, err
	}
	_, parsedTron, err := blockchain.LoadTronFactoryArtifact()
	if err != nil {
		return nil, err
	}
	return &Service{
		chain: chain,
		tron:  tron.NewContractService(tronClient, parsedTron),
		abi:   parsedEVM,
	}, nil
}

func (s *Service) ListFunctions() []FunctionDef { return Functions }

func (s *Service) GetFactoryInfo(ctx context.Context, networkName string) (*FactoryInfo, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if network.IsTron(net) {
		return s.tron.GetFactoryInfo(ctx, networkName)
	}
	addr, net, err := s.chain.FactoryAddress(networkName)
	if err != nil {
		return nil, err
	}
	client, _, err := s.chain.RPC(networkName)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if err := s.chain.AssertDeployed(ctx, client, net, addr); err != nil {
		return nil, err
	}

	mother, err := s.callAddress(ctx, client, addr, "motherWallet")
	if err != nil {
		return nil, err
	}
	relayer, err := s.callAddress(ctx, client, addr, "relayer")
	if err != nil {
		return nil, err
	}
	owner, err := s.callAddress(ctx, client, addr, "owner")
	if err != nil {
		return nil, err
	}
	impl, err := s.callAddress(ctx, client, addr, "implementation")
	if err != nil {
		return nil, err
	}

	return &FactoryInfo{
		FactoryAddress: addr.Hex(),
		MotherWallet:   mother.Hex(),
		Relayer:        relayer.Hex(),
		Owner:          owner.Hex(),
		Implementation: impl.Hex(),
	}, nil
}

func (s *Service) Call(ctx context.Context, networkName, functionName string, rawArgs map[string]string) (*CallResult, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if network.IsTron(net) {
		return s.tron.Call(ctx, networkName, functionName, rawArgs)
	}

	fn, ok := factoryabi.FindFunction(functionName)
	if !ok {
		return nil, apperror.BadRequest("Unknown function: " + functionName)
	}

	addr, net, err := s.chain.FactoryAddress(networkName)
	if err != nil {
		return nil, err
	}
	client, _, err := s.chain.RPC(networkName)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if err := s.chain.AssertDeployed(ctx, client, net, addr); err != nil {
		return nil, err
	}

	args, err := parseArgs(fn, rawArgs)
	if err != nil {
		return nil, err
	}

	if fn.Type == "read" {
		result, err := s.staticCall(ctx, client, addr, functionName, args...)
		if err != nil {
			return nil, err
		}
		return &CallResult{FunctionName: functionName, Type: "read", Result: result}, nil
	}

	role := "owner"
	if fn.Role == "relayer" {
		role = "relayer"
	}
	opts, writeClient, _, err := s.chain.Signer(networkName, role)
	if err != nil {
		return nil, err
	}
	defer writeClient.Close()

	contract := bind.NewBoundContract(addr, s.abi, writeClient, writeClient, writeClient)
	tx, err := contract.Transact(opts, functionName, args...)
	if err != nil {
		return nil, err
	}
	receipt, err := bind.WaitMined(ctx, writeClient, tx)
	if err != nil {
		return nil, err
	}

	return &CallResult{
		FunctionName: functionName,
		Type:         "write",
		TxHash:       tx.Hash().Hex(),
		BlockNumber:  receipt.BlockNumber.Uint64(),
		GasUsed:      strconv.FormatUint(receipt.GasUsed, 10),
	}, nil
}

func (s *Service) staticCall(ctx context.Context, client *ethclient.Client, addr common.Address, method string, args ...interface{}) (interface{}, error) {
	data, err := s.abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	results, err := s.abi.Unpack(method, out)
	if err != nil {
		return nil, err
	}
	return formatOutputs(s.abi, method, results)
}

func (s *Service) callAddress(ctx context.Context, client *ethclient.Client, addr common.Address, method string) (common.Address, error) {
	v, err := s.staticCall(ctx, client, addr, method)
	if err != nil {
		return common.Address{}, err
	}
	switch t := v.(type) {
	case common.Address:
		return t, nil
	case string:
		return common.HexToAddress(t), nil
	default:
		return common.Address{}, fmt.Errorf("unexpected %s result type", method)
	}
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
			if !common.IsHexAddress(value) {
				return nil, apperror.BadRequest("Invalid address: " + input.Label)
			}
			out = append(out, common.HexToAddress(value))
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
		return t.Hex()
	case []byte:
		return common.BytesToAddress(t).Hex()
	default:
		return v
	}
}
