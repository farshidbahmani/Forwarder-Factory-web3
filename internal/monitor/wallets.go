package monitor

type Wallet struct {
	UserID string  `json:"userId"`
	Label  *string `json:"label,omitempty"`
}

// Monitored wallets — hardcoded array for now
var Wallets = []Wallet{
	{UserID: "1", Label: strPtr("user-1")},
	{UserID: "2", Label: strPtr("user-2")},
}

func strPtr(s string) *string { return &s }
