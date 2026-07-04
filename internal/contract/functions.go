package contract

type Param struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
}

type FunctionDef struct {
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Role        string  `json:"role"`
	Inputs      []Param `json:"inputs"`
}

var Functions = []FunctionDef{
	{Name: "getAddress", Label: "Get User Wallet Address", Description: "Predict deterministic deposit address for a userId (no deployment needed).", Type: "read", Role: "any", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "implementation", Label: "Implementation Address", Description: "Forwarder implementation contract used for clones.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "motherWallet", Label: "Mother Wallet", Description: "Address that receives swept funds.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "relayer", Label: "Relayer", Description: "Address authorized to deploy and sweep.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "owner", Label: "Owner", Description: "Factory owner (admin).", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "TIMELOCK_DELAY", Label: "Timelock Delay", Description: "Seconds required before mother wallet change.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "pendingMotherWallet", Label: "Pending Mother Wallet", Description: "Mother wallet awaiting timelock.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "motherWalletUnlockTime", Label: "Mother Wallet Unlock Time", Description: "Unix timestamp when pending change can be applied.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "deployWallet", Label: "Deploy Wallet", Description: "Deploy user forwarder wallet (relayer only).", Type: "write", Role: "relayer", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "deployAndSweepNative", Label: "Deploy & Sweep Native", Description: "Deploy wallet and sweep native token to mother wallet.", Type: "write", Role: "relayer", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "deployAndSweepToken", Label: "Deploy & Sweep Token", Description: "Deploy wallet and sweep ERC20 token to mother wallet.", Type: "write", Role: "relayer", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}, {Name: "token", Type: "address", Label: "Token Address", Placeholder: "0x..."}}},
	{Name: "emergencyWithdrawNative", Label: "Emergency Withdraw Native", Description: "Owner rescues native token from a user wallet to mother wallet.", Type: "write", Role: "owner", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "emergencyWithdrawToken", Label: "Emergency Withdraw Token", Description: "Owner rescues ERC20 from a user wallet to mother wallet.", Type: "write", Role: "owner", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}, {Name: "token", Type: "address", Label: "Token Address", Placeholder: "0x..."}}},
	{Name: "requestMotherWalletChange", Label: "Request Mother Wallet Change", Description: "Start 48h timelock for mother wallet update.", Type: "write", Role: "owner", Inputs: []Param{{Name: "newMotherWallet", Type: "address", Label: "New Mother Wallet", Placeholder: "0x..."}}},
	{Name: "applyMotherWalletChange", Label: "Apply Mother Wallet Change", Description: "Apply pending mother wallet after timelock.", Type: "write", Role: "owner", Inputs: []Param{}},
	{Name: "cancelMotherWalletChange", Label: "Cancel Mother Wallet Change", Description: "Cancel pending mother wallet change.", Type: "write", Role: "owner", Inputs: []Param{}},
	{Name: "updateRelayer", Label: "Update Relayer", Description: "Set new relayer address.", Type: "write", Role: "owner", Inputs: []Param{{Name: "newRelayer", Type: "address", Label: "New Relayer", Placeholder: "0x..."}}},
	{Name: "transferOwnership", Label: "Transfer Ownership", Description: "Transfer factory ownership (e.g. to multisig).", Type: "write", Role: "owner", Inputs: []Param{{Name: "newOwner", Type: "address", Label: "New Owner", Placeholder: "0x..."}}},
}

func FindFunction(name string) (FunctionDef, bool) {
	for _, f := range Functions {
		if f.Name == name {
			return f, true
		}
	}
	return FunctionDef{}, false
}
