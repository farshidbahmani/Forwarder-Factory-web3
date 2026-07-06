package monitor

import (
	"sync"

	"forwarder-factory/internal/network"
)

type networkRegistry struct {
	settings  PushSettings
	addresses map[string]string // canonical -> original
}

type Registry struct {
	mu       sync.RWMutex
	networks map[string]*networkRegistry
}

func NewRegistry() *Registry {
	return &Registry{
		networks: map[string]*networkRegistry{},
	}
}

func (r *Registry) View() WalletRegistryView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]NetworkWalletRegistry, len(r.networks))
	for name, entry := range r.networks {
		out[name] = entry.view()
	}
	return WalletRegistryView{Networks: out}
}

func (r *Registry) Settings(networkName string) PushSettings {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry := r.networks[networkName]; entry != nil {
		return entry.settings
	}
	return PushSettings{}
}

func (r *Registry) Addresses(networkName string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.networks[networkName]
	if entry == nil {
		return nil
	}
	out := make([]string, 0, len(entry.addresses))
	for _, addr := range entry.addresses {
		out = append(out, addr)
	}
	return out
}

func (r *Registry) Replace(networkName string, net network.Config, req WalletPushRequest) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks[networkName] = &networkRegistry{
		settings:  req.Setting,
		addresses: map[string]string{},
	}
	return r.ingestLocked(r.networks[networkName], net, req)
}

func (r *Registry) Upsert(networkName string, net network.Config, req WalletPushRequest) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.networks[networkName]
	if entry == nil {
		entry = &networkRegistry{addresses: map[string]string{}}
		r.networks[networkName] = entry
	}
	entry.settings = req.Setting
	return r.ingestLocked(entry, net, req)
}

func (r *Registry) ingestLocked(entry *networkRegistry, net network.Config, req WalletPushRequest) (int, error) {
	added := 0
	for _, raw := range req.Wallets {
		if raw == "" {
			continue
		}
		normalized, err := normalizeForNetwork(raw, net)
		if err != nil {
			return 0, err
		}
		canonical, err := canonicalizeAddress(normalized)
		if err != nil {
			return 0, err
		}
		if _, ok := entry.addresses[canonical]; !ok {
			added++
		}
		entry.addresses[canonical] = raw
	}
	return added, nil
}

func (r *Registry) Remove(networkName string, net network.Config, addresses []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.networks[networkName]
	if entry == nil {
		return 0, nil
	}
	removed := 0
	for _, raw := range addresses {
		normalized, err := normalizeForNetwork(raw, net)
		if err != nil {
			return 0, err
		}
		canonical, err := canonicalizeAddress(normalized)
		if err != nil {
			return 0, err
		}
		if _, ok := entry.addresses[canonical]; ok {
			delete(entry.addresses, canonical)
			removed++
		}
	}
	if len(entry.addresses) == 0 {
		delete(r.networks, networkName)
	}
	return removed, nil
}

func (r *Registry) Count(networkName string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry := r.networks[networkName]; entry != nil {
		return len(entry.addresses)
	}
	return 0
}

func (nr *networkRegistry) view() NetworkWalletRegistry {
	wallets := make([]string, 0, len(nr.addresses))
	for _, addr := range nr.addresses {
		wallets = append(wallets, addr)
	}
	return NetworkWalletRegistry{
		Setting: nr.settings,
		Wallets: wallets,
	}
}
