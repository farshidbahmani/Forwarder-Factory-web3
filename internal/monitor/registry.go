package monitor

import (
	"sync"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/network"
)

type walletEntry struct {
	original  string // address as pushed by the caller
	canonical string // normalized form used for matching
}

type networkRegistry struct {
	settings PushSettings
	wallets  map[string]walletEntry // wallet ID -> entry
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

// Addresses returns deduplicated canonical addresses for a network.
func (r *Registry) Addresses(networkName string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.networks[networkName]
	if entry == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(entry.wallets))
	out := make([]string, 0, len(entry.wallets))
	for _, w := range entry.wallets {
		if _, ok := seen[w.canonical]; ok {
			continue
		}
		seen[w.canonical] = struct{}{}
		out = append(out, w.canonical)
	}
	return out
}

func (r *Registry) Replace(networkName string, net network.Config, req WalletPushRequest) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := &networkRegistry{
		settings: req.Setting,
		wallets:  map[string]walletEntry{},
	}
	added, err := r.ingestLocked(entry, net, req)
	if err != nil {
		return 0, err
	}
	r.networks[networkName] = entry
	return added, nil
}

func (r *Registry) Upsert(networkName string, net network.Config, req WalletPushRequest) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.networks[networkName]
	if entry == nil {
		entry = &networkRegistry{wallets: map[string]walletEntry{}}
	}
	entry.settings = req.Setting
	added, err := r.ingestLocked(entry, net, req)
	if err != nil {
		return 0, err
	}
	r.networks[networkName] = entry
	return added, nil
}

func (r *Registry) ingestLocked(entry *networkRegistry, net network.Config, req WalletPushRequest) (int, error) {
	affected := 0
	for id, raw := range req.Wallets {
		if id == "" {
			return 0, apperror.BadRequest("wallet id must not be empty")
		}
		if raw == "" {
			return 0, apperror.BadRequest("wallet address for id \"" + id + "\" must not be empty")
		}
		normalized, err := normalizeForNetwork(raw, net)
		if err != nil {
			return 0, err
		}
		canonical, err := canonicalizeAddress(normalized)
		if err != nil {
			return 0, err
		}
		if existing, ok := entry.wallets[id]; !ok || existing.canonical != canonical {
			affected++
		}
		entry.wallets[id] = walletEntry{original: raw, canonical: canonical}
	}
	return affected, nil
}

// Remove deletes wallets matched by ID or by address.
func (r *Registry) Remove(networkName string, net network.Config, wallets []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.networks[networkName]
	if entry == nil {
		return 0, nil
	}
	removed := 0
	for _, key := range wallets {
		if key == "" {
			continue
		}
		if _, ok := entry.wallets[key]; ok {
			delete(entry.wallets, key)
			removed++
			continue
		}
		canonical, err := canonicalizeAddress(key)
		if err != nil {
			return 0, apperror.BadRequest("\"" + key + "\" is neither a known wallet id nor a valid address")
		}
		for id, w := range entry.wallets {
			if w.canonical == canonical {
				delete(entry.wallets, id)
				removed++
			}
		}
	}
	if len(entry.wallets) == 0 {
		delete(r.networks, networkName)
	}
	return removed, nil
}

func (r *Registry) Count(networkName string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry := r.networks[networkName]; entry != nil {
		return len(entry.wallets)
	}
	return 0
}

// WalletID resolves a canonical address back to its wallet ID.
func (r *Registry) WalletID(networkName, canonical string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.networks[networkName]
	if entry == nil {
		return "", false
	}
	for id, w := range entry.wallets {
		if w.canonical == canonical {
			return id, true
		}
	}
	return "", false
}

func (nr *networkRegistry) view() NetworkWalletRegistry {
	wallets := make(map[string]string, len(nr.wallets))
	for id, w := range nr.wallets {
		wallets[id] = w.original
	}
	return NetworkWalletRegistry{
		Setting: nr.settings,
		Wallets: wallets,
	}
}
