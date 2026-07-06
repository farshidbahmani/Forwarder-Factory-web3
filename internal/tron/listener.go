package tron

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"strconv"
	"sync"
	"time"

	tronclient "github.com/fbsobreira/gotron-sdk/pkg/client"
	tronaddr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
)

var transferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

type DepositType string

const (
	DepositNative DepositType = "native"
	DepositToken  DepositType = "token"
)

type DepositEvent struct {
	Network     string
	Address     string
	Type        DepositType
	TxHash      string
	BlockNumber uint64
	Token       string
	Amount      *big.Int
}

type Listener struct {
	networkName     string
	client          *tronclient.GrpcClient
	addressToUserID map[string]string
	addrMu          sync.RWMutex
	onDeposit       func(DepositEvent)
	processed       map[string]struct{}
	mu              sync.Mutex
	running         bool
	lastBlock       uint64
	cancel          context.CancelFunc
}

func NewListener(networkName string, client *tronclient.GrpcClient, addressToUserID map[string]string, onDeposit func(DepositEvent)) *Listener {
	return &Listener{
		networkName:     networkName,
		client:          client,
		addressToUserID: addressToUserID,
		onDeposit:       onDeposit,
		processed:       map[string]struct{}{},
	}
}

func (l *Listener) IsRunning() bool   { return l.running }
func (l *Listener) LastBlock() uint64 { return l.lastBlock }

func (l *Listener) UpdateAddresses(addressToUserID map[string]string) {
	l.addrMu.Lock()
	l.addressToUserID = addressToUserID
	l.addrMu.Unlock()
}

func (l *Listener) lookupUserID(addr string) (string, bool) {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	id, ok := l.addressToUserID[addr]
	return id, ok
}

func (l *Listener) Start(ctx context.Context) error {
	if l.running {
		return nil
	}
	block, err := l.client.GetNowBlock()
	if err != nil {
		return err
	}
	if block.BlockHeader != nil && block.BlockHeader.RawData != nil {
		l.lastBlock = uint64(block.BlockHeader.RawData.Number)
	}

	ctx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	l.running = true
	go l.loop(ctx)
	return nil
}

func (l *Listener) Stop() {
	if !l.running {
		return
	}
	l.running = false
	if l.cancel != nil {
		l.cancel()
	}
}

func (l *Listener) loop(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			block, err := l.client.GetNowBlock()
			if err != nil {
				continue
			}
			if block.BlockHeader == nil || block.BlockHeader.RawData == nil {
				continue
			}
			num := uint64(block.BlockHeader.RawData.Number)
			if num <= l.lastBlock {
				continue
			}
			for b := l.lastBlock + 1; b <= num; b++ {
				if err := l.processBlock(ctx, int64(b)); err != nil {
					continue
				}
				l.lastBlock = b
			}
		}
	}
}

func (l *Listener) processBlock(ctx context.Context, blockNumber int64) error {
	if len(l.addressToUserID) == 0 {
		return nil
	}

	block, err := l.client.GetBlockByNumCtx(ctx, blockNumber)
	if err != nil {
		return err
	}

	for _, txExt := range block.Transactions {
		if txExt.Transaction == nil {
			continue
		}
		txID := hex.EncodeToString(txExt.Txid)
		decoded, err := transaction.DecodeContractData(txExt.Transaction)
		if err == nil && decoded.Type == "TransferContract" {
			to, _ := decoded.Fields["to_address"].(string)
			amount, _ := decoded.Fields["amount"].(string)
			if to != "" && amount != "0.000000" {
				if _, ok := l.lookupUserID(to); ok {
					key := txID + ":native"
					if l.markProcessed(key) {
						l.onDeposit(DepositEvent{
							Network: l.networkName, Address: to, Type: DepositNative,
							TxHash: txID, BlockNumber: uint64(blockNumber), Amount: parseTronTRXAmount(amount),
						})
					}
				}
			}
		}
	}

	infoList, err := l.client.GetBlockInfoByNumCtx(ctx, blockNumber)
	if err != nil {
		return err
	}
	for _, info := range infoList.TransactionInfo {
		txID := hex.EncodeToString(info.Id)
		for _, lg := range info.Log {
			if len(lg.Topics) < 3 {
				continue
			}
			if !bytes.Equal(lg.Topics[0], transferTopic.Bytes()) {
				continue
			}
			to := topicToAddress(lg.Topics[2])
			if _, ok := l.lookupUserID(to); !ok {
				continue
			}
			token := tronaddr.Address(lg.Address).String()
			key := txID + ":" + token + ":token"
			if l.markProcessed(key) {
				amount := new(big.Int).SetBytes(lg.Data)
				l.onDeposit(DepositEvent{
					Network: l.networkName, Address: to, Type: DepositToken,
					TxHash: txID, BlockNumber: uint64(blockNumber), Token: token, Amount: amount,
				})
			}
		}
	}
	return nil
}

func (l *Listener) markProcessed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.processed[key]; ok {
		return false
	}
	l.processed[key] = struct{}{}
	return true
}

func topicToAddress(topic []byte) string {
	b := topic
	if len(b) > 20 {
		b = b[len(b)-20:]
	}
	return tronaddr.BytesToAddress(b).String()
}

func parseTronTRXAmount(amount string) *big.Int {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil || f <= 0 {
		return big.NewInt(0)
	}
	sun := new(big.Float).Mul(big.NewFloat(f), big.NewFloat(1e6))
	v, _ := sun.Int(nil)
	return v
}
