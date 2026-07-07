package monitor

import (
	"context"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	pollInterval = 3 * time.Second
	// Cap catch-up work per tick so a lagging monitor doesn't burst-hammer the RPC.
	maxBlocksPerPoll = 10
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
	client          *ethclient.Client
	addressToUserID map[common.Address]string
	tokens          []common.Address
	addrMu          sync.RWMutex
	disableTokenLogs bool
	onDeposit       func(DepositEvent)
	processed       map[string]struct{}
	mu              sync.Mutex
	running         bool
	lastBlock       uint64
	cancel          context.CancelFunc
}

func NewListener(networkName string, client *ethclient.Client, addressToUserID map[common.Address]string, tokens []common.Address, onDeposit func(DepositEvent)) *Listener {
	return &Listener{
		networkName:     networkName,
		client:          client,
		addressToUserID: addressToUserID,
		tokens:          tokens,
		onDeposit:       onDeposit,
		processed:       map[string]struct{}{},
	}
}

func (l *Listener) IsRunning() bool  { return l.running }
func (l *Listener) LastBlock() uint64 { return l.lastBlock }

func (l *Listener) UpdateAddresses(addressToUserID map[common.Address]string) {
	l.addrMu.Lock()
	l.addressToUserID = addressToUserID
	l.addrMu.Unlock()
}

func (l *Listener) UpdateTokens(tokens []common.Address) {
	l.addrMu.Lock()
	l.tokens = tokens
	l.disableTokenLogs = false
	l.addrMu.Unlock()
}

func (l *Listener) tokenFilter() []common.Address {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	out := make([]common.Address, len(l.tokens))
	copy(out, l.tokens)
	return out
}

func (l *Listener) lookupUserID(addr common.Address) (string, bool) {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	id, ok := l.addressToUserID[addr]
	return id, ok
}

func (l *Listener) tokenLogsEnabled() bool {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	return !l.disableTokenLogs
}

func (l *Listener) disableTokenLogsOnce(reason error) {
	l.addrMu.Lock()
	defer l.addrMu.Unlock()
	if l.disableTokenLogs {
		return
	}
	l.disableTokenLogs = true
	log.Printf("[monitor:%s] token log scanning disabled (RPC limitation): %v", l.networkName, reason)
}

func (l *Listener) watchedCount() int {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	return len(l.addressToUserID)
}

// recipientTopics returns watched addresses encoded as log topics, so token
// transfers can be filtered server-side instead of fetching every Transfer log.
func (l *Listener) recipientTopics() []common.Hash {
	l.addrMu.RLock()
	defer l.addrMu.RUnlock()
	out := make([]common.Hash, 0, len(l.addressToUserID))
	for addr := range l.addressToUserID {
		out = append(out, common.BytesToHash(addr.Bytes()))
	}
	return out
}

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
		// HTTP RPC endpoints don't support subscriptions; fall back to polling.
		log.Printf("[monitor:%s] head subscription unavailable (%v), falling back to polling every %s", l.networkName, err, pollInterval)
		l.pollLoop(ctx)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			if err != nil {
				log.Printf("[monitor:%s] head subscription error (%v), falling back to polling every %s", l.networkName, err, pollInterval)
				l.pollLoop(ctx)
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

func (l *Listener) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			head, err := l.client.BlockNumber(ctx)
			if err != nil {
				log.Printf("[monitor:%s] failed to fetch latest block: %v", l.networkName, err)
				continue
			}
			if head <= l.lastBlock {
				continue
			}
			from := l.lastBlock + 1
			to := head
			if to > from+maxBlocksPerPoll-1 {
				to = from + maxBlocksPerPoll - 1
			}
			if l.watchedCount() == 0 {
				l.lastBlock = to
				continue
			}
			// Token logs may be restricted by public RPCs; native scanning should still proceed.
			if err := l.processTokenTransfers(ctx, from, to); err != nil {
				log.Printf("[monitor:%s] failed to scan token transfers in blocks %d-%d: %v", l.networkName, from, to, err)
			}
			for b := from; b <= to; b++ {
				if err := l.processNativeTransfers(ctx, b); err != nil {
					log.Printf("[monitor:%s] failed to process block %d: %v", l.networkName, b, err)
					break
				}
				l.lastBlock = b
			}
		}
	}
}

func (l *Listener) processBlock(ctx context.Context, blockNumber uint64) error {
	if l.watchedCount() == 0 {
		return nil
	}
	if err := l.processNativeTransfers(ctx, blockNumber); err != nil {
		return err
	}
	return l.processTokenTransfers(ctx, blockNumber, blockNumber)
}

func (l *Listener) processNativeTransfers(ctx context.Context, blockNumber uint64) error {
	block, err := l.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return err
	}

	for _, tx := range block.Transactions() {
		if tx.To() == nil || tx.Value().Sign() == 0 {
			continue
		}
		to := *tx.To()
		userID, ok := l.lookupUserID(to)
		if !ok {
			continue
		}
		key := tx.Hash().Hex() + ":native"
		if l.markProcessed(key) {
			l.onDeposit(DepositEvent{
				Network: l.networkName, Address: userID, Type: DepositNative,
				TxHash: tx.Hash().Hex(), BlockNumber: blockNumber, Amount: tx.Value(),
			})
		}
	}
	return nil
}

func (l *Listener) processTokenTransfers(ctx context.Context, fromBlock, toBlock uint64) error {
	if !l.tokenLogsEnabled() {
		return nil
	}
	recipients := l.recipientTopics()
	if len(recipients) == 0 {
		return nil
	}
	// Only tokens listed in settings are ever swept, so restrict the filter to
	// them. Public RPCs also reject eth_getLogs without an address filter.
	tokens := l.tokenFilter()
	if len(tokens) == 0 {
		return nil
	}
	logs, err := l.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: tokens,
		Topics:    [][]common.Hash{{transferTopic}, nil, recipients},
	})
	if err != nil {
		// Some public RPCs restrict eth_getLogs and return "archive requests require a personal token".
		// Disable token scanning so native monitoring continues uninterrupted.
		msg := err.Error()
		if strings.Contains(msg, "Archive requests require a personal token") || strings.Contains(msg, "403 Forbidden") {
			l.disableTokenLogsOnce(err)
		}
		return err
	}
	for _, lg := range logs {
		if len(lg.Topics) < 3 {
			continue
		}
		to := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
		userID, ok := l.lookupUserID(to)
		if !ok {
			continue
		}
		key := lg.TxHash.Hex() + ":" + lg.Address.Hex() + ":token"
		if l.markProcessed(key) {
			amount := new(big.Int).SetBytes(lg.Data)
			l.onDeposit(DepositEvent{
				Network: l.networkName, Address: userID, Type: DepositToken,
				TxHash: lg.TxHash.Hex(), BlockNumber: lg.BlockNumber, Token: lg.Address.Hex(),
				Amount: amount,
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
