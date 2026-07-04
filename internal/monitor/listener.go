package monitor

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var transferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

type DepositType string

const (
	DepositNative DepositType = "native"
	DepositToken  DepositType = "token"
)

type DepositEvent struct {
	Network     string
	UserID      string
	Type        DepositType
	TxHash      string
	BlockNumber uint64
	Token       string
}

type Listener struct {
	networkName     string
	client          *ethclient.Client
	addressToUserID map[common.Address]string
	onDeposit       func(DepositEvent)
	processed       map[string]struct{}
	mu              sync.Mutex
	running         bool
	lastBlock       uint64
	cancel          context.CancelFunc
}

func NewListener(networkName string, client *ethclient.Client, addressToUserID map[common.Address]string, onDeposit func(DepositEvent)) *Listener {
	return &Listener{
		networkName:     networkName,
		client:          client,
		addressToUserID: addressToUserID,
		onDeposit:       onDeposit,
		processed:       map[string]struct{}{},
	}
}

func (l *Listener) IsRunning() bool  { return l.running }
func (l *Listener) LastBlock() uint64 { return l.lastBlock }

func (l *Listener) Start(ctx context.Context) error {
	if l.running {
		return nil
	}
	block, err := l.client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	l.lastBlock = block

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
	headers := make(chan *types.Header)
	sub, err := l.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		l.running = false
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			if err != nil {
				return
			}
		case h := <-headers:
			if h == nil {
				continue
			}
			_ = l.processBlock(ctx, h.Number.Uint64())
			l.lastBlock = h.Number.Uint64()
		}
	}
}

func (l *Listener) processBlock(ctx context.Context, blockNumber uint64) error {
	if len(l.addressToUserID) == 0 {
		return nil
	}

	block, err := l.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return err
	}

	for _, tx := range block.Transactions() {
		if tx.To() == nil || tx.Value().Sign() == 0 {
			continue
		}
		to := *tx.To()
		userID, ok := l.addressToUserID[to]
		if !ok {
			continue
		}
		key := tx.Hash().Hex() + ":native"
		if l.markProcessed(key) {
			l.onDeposit(DepositEvent{
				Network: l.networkName, UserID: userID, Type: DepositNative,
				TxHash: tx.Hash().Hex(), BlockNumber: blockNumber,
			})
		}
	}

	bn := big.NewInt(int64(blockNumber))
	logs, err := l.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: bn,
		ToBlock:   bn,
		Topics:    [][]common.Hash{{transferTopic}},
	})
	if err != nil {
		return err
	}
	for _, lg := range logs {
		if len(lg.Topics) < 3 {
			continue
		}
		to := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
		userID, ok := l.addressToUserID[to]
		if !ok {
			continue
		}
		key := lg.TxHash.Hex() + ":" + lg.Address.Hex() + ":token"
		if l.markProcessed(key) {
			l.onDeposit(DepositEvent{
				Network: l.networkName, UserID: userID, Type: DepositToken,
				TxHash: lg.TxHash.Hex(), BlockNumber: blockNumber, Token: lg.Address.Hex(),
			})
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
