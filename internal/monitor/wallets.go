package monitor

type TokenSetting struct {
	Token            string  `json:"token"`
	MinValueTransfer float64 `json:"minValueTransfer"`
}

type PushSettings struct {
	NativeMinValueTransfer float64        `json:"nativeMinValueTransfer"`
	Tokens                 []TokenSetting `json:"tokens"`
}

// WalletPushRequest is the external push payload (scoped to one network).
type WalletPushRequest struct {
	Network string       `json:"network"`
	Setting PushSettings `json:"setting"`
	Wallets []string     `json:"wallets"`
}

type WalletRemoveRequest struct {
	Network string   `json:"network"`
	Wallets []string `json:"wallets"`
}

type NetworkWalletRegistry struct {
	Setting PushSettings `json:"setting"`
	Wallets []string     `json:"wallets"`
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
