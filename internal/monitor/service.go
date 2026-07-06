package monitor

import (
	"context"
	"log"
	"sync"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"
	"forwarder-factory/internal/tron"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const maxRecentSweeps = 50

type SweepResult struct {
	Address     string      `json:"address"`
	Type        DepositType `json:"type"`
	TxHash      string      `json:"txHash"`
	SweepTxHash *string     `json:"sweepTxHash,omitempty"`
	Error       *string     `json:"error,omitempty"`
}

type NetworkStatus struct {
	Network          string        `json:"network"`
	Running          bool          `json:"running"`
	LastBlock        *uint64       `json:"lastBlock,omitempty"`
	WatchedAddresses []string      `json:"watchedAddresses"`
	RecentSweeps     []SweepResult `json:"recentSweeps"`
}

type networkState struct {
	networkName      string
	evmListener      *Listener
	tronListener     *tron.Listener
	evmClient        *ethclient.Client
	watchedAddresses []string
	recentSweeps     []SweepResult
	mu               sync.Mutex
}

type Service struct {
	env       *env.Store
	contracts *contract.Service
	tron      *tron.Client
	registry  *Registry
	monitors  map[string]*networkState
	mu        sync.Mutex
}

func New(envStore *env.Store, contracts *contract.Service, tronClient *tron.Client) *Service {
	return &Service{
		env:       envStore,
		contracts: contracts,
		tron:      tronClient,
		registry:  NewRegistry(),
		monitors:  map[string]*networkState{},
	}
}

func (s *Service) ListMonitoredWallets() WalletRegistryView { return s.registry.View() }

func (s *Service) ReplaceWallets(ctx context.Context, req WalletPushRequest) (*WalletMutationResult, error) {
	net, err := validatePush(req)
	if err != nil {
		return nil, err
	}
	affected, err := s.registry.Replace(req.Network, net, req)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.refreshRunningMonitor(ctx, req.Network)
	if err != nil {
		return nil, err
	}
	return &WalletMutationResult{
		Network:           req.Network,
		TotalWallets:      s.registry.Count(req.Network),
		Affected:          affected,
		RefreshedNetworks: refreshed,
	}, nil
}

func (s *Service) PushWallets(ctx context.Context, req WalletPushRequest) (*WalletMutationResult, error) {
	net, err := validatePush(req)
	if err != nil {
		return nil, err
	}
	affected, err := s.registry.Upsert(req.Network, net, req)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.refreshRunningMonitor(ctx, req.Network)
	if err != nil {
		return nil, err
	}
	return &WalletMutationResult{
		Network:           req.Network,
		TotalWallets:      s.registry.Count(req.Network),
		Affected:          affected,
		RefreshedNetworks: refreshed,
	}, nil
}

func (s *Service) RemoveWallets(ctx context.Context, req WalletRemoveRequest) (*WalletMutationResult, error) {
	if req.Network == "" {
		return nil, apperror.BadRequest("network is required")
	}
	net, err := network.Get(req.Network)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if len(req.Wallets) == 0 {
		return nil, apperror.BadRequest("wallets is required")
	}
	removed, err := s.registry.Remove(req.Network, net, req.Wallets)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.refreshRunningMonitor(ctx, req.Network)
	if err != nil {
		return nil, err
	}
	return &WalletMutationResult{
		Network:           req.Network,
		TotalWallets:      s.registry.Count(req.Network),
		Affected:          removed,
		RefreshedNetworks: refreshed,
	}, nil
}

func validatePush(req WalletPushRequest) (network.Config, error) {
	if req.Network == "" {
		return network.Config{}, apperror.BadRequest("network is required")
	}
	net, err := network.Get(req.Network)
	if err != nil {
		return network.Config{}, apperror.BadRequest(err.Error())
	}
	if len(req.Wallets) == 0 {
		return network.Config{}, apperror.BadRequest("wallets is required")
	}
	return net, nil
}

func (s *Service) refreshRunningMonitor(ctx context.Context, networkName string) ([]NetworkStatus, error) {
	s.mu.Lock()
	st := s.monitors[networkName]
	running := st != nil && s.isRunning(st)
	s.mu.Unlock()
	if !running {
		return nil, nil
	}

	watched, err := s.buildWatched(networkName)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st = s.monitors[networkName]
	if st == nil || !s.isRunning(st) {
		return nil, nil
	}
	st.mu.Lock()
	st.watchedAddresses = watched
	st.mu.Unlock()
	if st.evmListener != nil {
		st.evmListener.UpdateAddresses(watchedToEVMMap(watched))
	}
	if st.tronListener != nil {
		st.tronListener.UpdateAddresses(watchedToTronMap(watched))
	}
	return []NetworkStatus{*st.status()}, nil
}

func watchedToEVMMap(watched []string) map[common.Address]string {
	m := make(map[common.Address]string, len(watched))
	for _, addr := range watched {
		m[common.HexToAddress(addr)] = addr
	}
	return m
}

func watchedToTronMap(watched []string) map[string]string {
	m := make(map[string]string, len(watched))
	for _, addr := range watched {
		m[addr] = addr
	}
	return m
}

func (s *Service) IsRunning(networkName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.monitors[networkName]
	return st != nil && s.isRunning(st)
}

func (s *Service) GetStatus(networkName string) (*NetworkStatus, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	s.mu.Lock()
	st := s.monitors[networkName]
	s.mu.Unlock()
	if st == nil {
		return &NetworkStatus{Network: networkName, WatchedAddresses: []string{}, RecentSweeps: []SweepResult{}}, nil
	}
	return st.status(), nil
}

func (s *Service) ListRunning() []NetworkStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []NetworkStatus{}
	for _, st := range s.monitors {
		if s.isRunning(st) {
			out = append(out, *st.status())
		}
	}
	return out
}

func (s *Service) buildWatched(networkName string) ([]string, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	out := make([]string, 0)
	for _, raw := range s.registry.Addresses(networkName) {
		addr, err := normalizeForNetwork(raw, net)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

func (s *Service) ResolveAddresses(ctx context.Context, networkName string) ([]string, error) {
	return s.buildWatched(networkName)
}

func (s *Service) Start(ctx context.Context, networkName string) (*NetworkStatus, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	factoryKey := network.EnvKey("FACTORY_ADDRESS", net)
	if s.env.GetForNetwork("FACTORY_ADDRESS", net.EnvSuffix) == "" {
		return nil, apperror.BadRequest("No factory deployed. Set " + factoryKey + " in .env")
	}

	s.mu.Lock()
	st := s.monitors[networkName]
	if st != nil && s.isRunning(st) {
		status := st.status()
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	watched, err := s.buildWatched(networkName)
	if err != nil {
		return nil, err
	}

	st = &networkState{networkName: networkName, watchedAddresses: watched, recentSweeps: []SweepResult{}}

	if network.IsTron(net) {
		if err := s.startTron(ctx, networkName, st, watched); err != nil {
			return nil, err
		}
	} else {
		if err := s.startEVM(ctx, networkName, net, st, watched); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	s.monitors[networkName] = st
	s.mu.Unlock()

	log.Printf("[monitor:%s] started — watching %d wallet(s)", networkName, len(watched))
	return st.status(), nil
}

func (s *Service) startEVM(ctx context.Context, networkName string, net network.Config, st *networkState, watched []string) error {
	addrMap := map[common.Address]string{}
	for _, addr := range watched {
		addrMap[common.HexToAddress(addr)] = addr
	}
	client, err := ethclient.Dial(network.RPCURL(net, s.env.Get))
	if err != nil {
		return err
	}
	st.evmClient = client
	st.evmListener = NewListener(networkName, client, addrMap, func(ev DepositEvent) {
		s.handleDeposit(ctx, networkName, st, ev)
	})
	return st.evmListener.Start(ctx)
}

func (s *Service) startTron(ctx context.Context, networkName string, st *networkState, watched []string) error {
	addrMap := map[string]string{}
	for _, addr := range watched {
		addrMap[addr] = addr
	}
	grpc, _, err := s.tron.GRPC(networkName)
	if err != nil {
		return err
	}
	st.tronListener = tron.NewListener(networkName, grpc, addrMap, func(ev tron.DepositEvent) {
		s.handleDeposit(ctx, networkName, st, DepositEvent{
			Network: ev.Network, Address: ev.Address, Type: DepositType(ev.Type),
			TxHash: ev.TxHash, BlockNumber: ev.BlockNumber, Token: ev.Token, Amount: ev.Amount,
		})
	})
	return st.tronListener.Start(ctx)
}

func (s *Service) Stop(networkName string) (*NetworkStatus, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	s.mu.Lock()
	st := s.monitors[networkName]
	if st == nil {
		s.mu.Unlock()
		return &NetworkStatus{Network: networkName, WatchedAddresses: []string{}, RecentSweeps: []SweepResult{}}, nil
	}
	if st.evmListener != nil {
		st.evmListener.Stop()
	}
	if st.tronListener != nil {
		st.tronListener.Stop()
	}
	if st.evmClient != nil {
		st.evmClient.Close()
	}
	if st.tronListener != nil {
		s.tron.Close(networkName)
	}
	status := st.status()
	status.Running = false
	delete(s.monitors, networkName)
	s.mu.Unlock()
	log.Printf("[monitor:%s] stopped", networkName)
	return status, nil
}

func (s *Service) handleDeposit(ctx context.Context, networkName string, st *networkState, ev DepositEvent) {
	net, _ := network.Get(networkName)
	settings := s.registry.Settings(networkName)

	sweep := SweepResult{Address: ev.Address, Type: ev.Type, TxHash: ev.TxHash}
	log.Printf("[monitor:%s] deposit: address=%s type=%s tx=%s", networkName, ev.Address, ev.Type, ev.TxHash)

	if ev.Type == DepositNative {
		if !settings.nativeMeetsMin(ev.Amount, nativeDecimals(net)) {
			log.Printf("[monitor:%s] skip native deposit below min: address=%s", networkName, ev.Address)
			return
		}
	} else {
		if !settings.tokenMeetsMin(ev.Token, ev.Amount, tokenDecimals(net)) {
			log.Printf("[monitor:%s] skip token deposit below min or token not listed: address=%s token=%s", networkName, ev.Address, ev.Token)
			return
		}
	}

	var res *contract.CallResult
	var err error
	if ev.Type == DepositNative {
		res, err = s.contracts.Call(ctx, networkName, "sweepNative", map[string]string{"wallet": ev.Address})
	} else {
		res, err = s.contracts.Call(ctx, networkName, "sweepToken", map[string]string{"wallet": ev.Address, "token": ev.Token})
	}
	if err != nil {
		msg := err.Error()
		sweep.Error = &msg
		log.Printf("[monitor:%s] sweep failed: address=%s — %s", networkName, ev.Address, msg)
	} else {
		sweep.SweepTxHash = &res.TxHash
		log.Printf("[monitor:%s] sweep sent: address=%s sweepTx=%s", networkName, ev.Address, res.TxHash)
	}

	st.appendSweep(sweep)
}

func (ns *networkState) appendSweep(sweep SweepResult) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.recentSweeps = append([]SweepResult{sweep}, ns.recentSweeps...)
	if len(ns.recentSweeps) > maxRecentSweeps {
		ns.recentSweeps = ns.recentSweeps[:maxRecentSweeps]
	}
}

func (s *Service) isRunning(st *networkState) bool {
	if st.evmListener != nil && st.evmListener.IsRunning() {
		return true
	}
	if st.tronListener != nil && st.tronListener.IsRunning() {
		return true
	}
	return false
}

func (ns *networkState) status() *NetworkStatus {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	var last *uint64
	if ns.evmListener != nil {
		lb := ns.evmListener.LastBlock()
		last = &lb
	} else if ns.tronListener != nil {
		lb := ns.tronListener.LastBlock()
		last = &lb
	}
	sweeps := make([]SweepResult, len(ns.recentSweeps))
	copy(sweeps, ns.recentSweeps)
	watched := make([]string, len(ns.watchedAddresses))
	copy(watched, ns.watchedAddresses)
	running := false
	if ns.evmListener != nil {
		running = ns.evmListener.IsRunning()
	} else if ns.tronListener != nil {
		running = ns.tronListener.IsRunning()
	}
	return &NetworkStatus{
		Network:          ns.networkName,
		Running:          running,
		LastBlock:        last,
		WatchedAddresses: watched,
		RecentSweeps:     sweeps,
	}
}
