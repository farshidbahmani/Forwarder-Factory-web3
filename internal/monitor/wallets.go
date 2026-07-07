package monitor

type TokenSetting struct {
	Token          string  `json:"token"`
	MinTokenBalance float64 `json:"minTokenBalance"`
}

type PushSettings struct {
	MinNativeBalance float64        `json:"minNativeBalance"`
	Tokens           []TokenSetting `json:"tokens"`
}

// WalletPushRequest is the external push payload (scoped to one network).
// Wallets maps a caller-defined wallet ID to its address.
type WalletPushRequest struct {
	Network string            `json:"network"`
	Setting PushSettings      `json:"setting"`
	Wallets map[string]string `json:"wallets"`
}

// WalletRemoveRequest removes wallets by ID or by address.
type WalletRemoveRequest struct {
	Network string   `json:"network"`
	Wallets []string `json:"wallets"`
}

type NetworkWalletRegistry struct {
	Setting PushSettings      `json:"setting"`
	Wallets map[string]string `json:"wallets"`
}

type WalletRegistryView struct {
	Networks map[string]NetworkWalletRegistry `json:"networks"`
}

type WalletMutationResult struct {
	Network           string          `json:"network"`
	TotalWallets      int             `json:"totalWallets"`
	Affected          int             `json:"affected"`
	RefreshedNetworks []NetworkStatus `json:"refreshedNetworks,omitempty"`
}
