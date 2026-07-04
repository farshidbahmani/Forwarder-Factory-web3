package monitor

import (
	"context"
	"log"
	"sync"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const maxRecentSweeps = 50

type SweepResult struct {
	UserID      string      `json:"userId"`
	Type        DepositType `json:"type"`
	TxHash      string      `json:"txHash"`
	SweepTxHash *string     `json:"sweepTxHash,omitempty"`
	Error       *string     `json:"error,omitempty"`
}

type WatchedAddress struct {
	UserID  string  `json:"userId"`
	Address string  `json:"address"`
	Label   *string `json:"label,omitempty"`
}

type NetworkStatus struct {
	Network          string           `json:"network"`
	Running          bool             `json:"running"`
	LastBlock        *uint64          `json:"lastBlock,omitempty"`
	WatchedAddresses []WatchedAddress `json:"watchedAddresses"`
	RecentSweeps     []SweepResult    `json:"recentSweeps"`
}

type networkState struct {
	networkName      string
	listener         *Listener
	client           *ethclient.Client
	watchedAddresses []WatchedAddress
	recentSweeps     []SweepResult
	mu               sync.Mutex
}

type Service struct {
	env      *env.Store
	contracts *contract.Service
	monitors map[string]*networkState
	mu       sync.Mutex
}

func New(envStore *env.Store, contracts *contract.Service) *Service {
	return &Service{env: envStore, contracts: contracts, monitors: map[string]*networkState{}}
}

func (s *Service) ListMonitoredWallets() []Wallet { return Wallets }

func (s *Service) IsRunning(networkName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.monitors[networkName]
	return st != nil && st.listener != nil && st.listener.IsRunning()
}

func (s *Service) GetStatus(networkName string) (*NetworkStatus, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	s.mu.Lock()
	st := s.monitors[networkName]
	s.mu.Unlock()
	if st == nil {
		return &NetworkStatus{Network: networkName, WatchedAddresses: []WatchedAddress{}, RecentSweeps: []SweepResult{}}, nil
	}
	return st.status(), nil
}

func (s *Service) ListRunning() []NetworkStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []NetworkStatus{}
	for _, st := range s.monitors {
		if st.listener != nil && st.listener.IsRunning() {
			out = append(out, *st.status())
		}
	}
	return out
}

func (s *Service) ResolveAddresses(ctx context.Context, networkName string) ([]WatchedAddress, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	out := []WatchedAddress{}
	for _, w := range Wallets {
		res, err := s.contracts.Call(ctx, networkName, "getAddress", map[string]string{"userId": w.UserID})
		if err != nil {
			return nil, err
		}
		addr, _ := res.Result.(string)
		out = append(out, WatchedAddress{UserID: w.UserID, Address: common.HexToAddress(addr).Hex(), Label: w.Label})
	}
	return out, nil
}

func (s *Service) Start(ctx context.Context, networkName string) (*NetworkStatus, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	factoryKey := network.EnvKey("FACTORY_ADDRESS", net)
	if s.env.Get(factoryKey) == "" {
		return nil, apperror.BadRequest("No factory deployed. Set " + factoryKey + " in .env")
	}

	s.mu.Lock()
	st := s.monitors[networkName]
	if st != nil && st.listener != nil && st.listener.IsRunning() {
		status := st.status()
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	watched, err := s.ResolveAddresses(ctx, networkName)
	if err != nil {
		return nil, err
	}
	addrMap := map[common.Address]string{}
	for _, w := range watched {
		addrMap[common.HexToAddress(w.Address)] = w.UserID
	}

	client, err := ethclient.Dial(network.RPCURL(net, s.env.Get))
	if err != nil {
		return nil, err
	}

	st = &networkState{networkName: networkName, client: client, watchedAddresses: watched, recentSweeps: []SweepResult{}}
	st.listener = NewListener(networkName, client, addrMap, func(ev DepositEvent) {
		s.handleDeposit(ctx, networkName, st, ev)
	})

	if err := st.listener.Start(ctx); err != nil {
		client.Close()
		return nil, err
	}

	s.mu.Lock()
	s.monitors[networkName] = st
	s.mu.Unlock()

	log.Printf("[monitor:%s] started — watching %d wallet(s)", networkName, len(watched))
	return st.status(), nil
}

func (s *Service) Stop(networkName string) (*NetworkStatus, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	s.mu.Lock()
	st := s.monitors[networkName]
	if st == nil {
		s.mu.Unlock()
		return &NetworkStatus{Network: networkName, WatchedAddresses: []WatchedAddress{}, RecentSweeps: []SweepResult{}}, nil
	}
	st.listener.Stop()
	st.client.Close()
	status := st.status()
	status.Running = false
	delete(s.monitors, networkName)
	s.mu.Unlock()
	log.Printf("[monitor:%s] stopped", networkName)
	return status, nil
}

func (s *Service) handleDeposit(ctx context.Context, networkName string, st *networkState, ev DepositEvent) {
	sweep := SweepResult{UserID: ev.UserID, Type: ev.Type, TxHash: ev.TxHash}
	log.Printf("[monitor:%s] deposit: userId=%s type=%s tx=%s", networkName, ev.UserID, ev.Type, ev.TxHash)

	var res *contract.CallResult
	var err error
	if ev.Type == DepositNative {
		res, err = s.contracts.Call(ctx, networkName, "deployAndSweepNative", map[string]string{"userId": ev.UserID})
	} else {
		res, err = s.contracts.Call(ctx, networkName, "deployAndSweepToken", map[string]string{"userId": ev.UserID, "token": ev.Token})
	}
	if err != nil {
		msg := err.Error()
		sweep.Error = &msg
		log.Printf("[monitor:%s] sweep failed: userId=%s — %s", networkName, ev.UserID, msg)
	} else {
		sweep.SweepTxHash = &res.TxHash
		log.Printf("[monitor:%s] sweep sent: userId=%s sweepTx=%s", networkName, ev.UserID, res.TxHash)
	}

	st.mu.Lock()
	st.recentSweeps = append([]SweepResult{sweep}, st.recentSweeps...)
	if len(st.recentSweeps) > maxRecentSweeps {
		st.recentSweeps = st.recentSweeps[:maxRecentSweeps]
	}
	st.mu.Unlock()
}

func (ns *networkState) status() *NetworkStatus {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	var last *uint64
	if ns.listener != nil {
		lb := ns.listener.LastBlock()
		last = &lb
	}
	sweeps := make([]SweepResult, len(ns.recentSweeps))
	copy(sweeps, ns.recentSweeps)
	watched := make([]WatchedAddress, len(ns.watchedAddresses))
	copy(watched, ns.watchedAddresses)
	return &NetworkStatus{
		Network:          ns.networkName,
		Running:          ns.listener != nil && ns.listener.IsRunning(),
		LastBlock:        last,
		WatchedAddresses: watched,
		RecentSweeps:     sweeps,
	}
}
