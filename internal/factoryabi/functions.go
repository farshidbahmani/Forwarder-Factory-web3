package factoryabi

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

type CallResult struct {
	FunctionName string      `json:"functionName"`
	Type         string      `json:"type"`
	Result       interface{} `json:"result,omitempty"`
	TxHash       string      `json:"txHash,omitempty"`
	BlockNumber  uint64      `json:"blockNumber,omitempty"`
	GasUsed      string      `json:"gasUsed,omitempty"`
}

type FactoryInfo struct {
	FactoryAddress string `json:"factoryAddress"`
	MotherWallet   string `json:"motherWallet"`
	Relayer        string `json:"relayer"`
	Owner          string `json:"owner"`
	Implementation string `json:"implementation"`
}

var Functions = []FunctionDef{
	{Name: "getAddress", Label: "Get User Wallet Address", Description: "Predict deterministic deposit address for a userId (no deployment needed).", Type: "read", Role: "any", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "implementation", Label: "Implementation Address", Description: "Forwarder implementation contract used for clones.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "setImplementation", Label: "Set Implementation", Description: "Tron only: link deployed Forwarder implementation (owner, once). Required after factory deploy before getAddress works.", Type: "write", Role: "owner", Inputs: []Param{{Name: "implementation", Type: "address", Label: "Implementation Address", Placeholder: "T... or 0x..."}}},
	{Name: "motherWallet", Label: "Mother Wallet", Description: "Address that receives swept funds.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "relayer", Label: "Relayer", Description: "Address authorized to deploy and sweep.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "owner", Label: "Owner", Description: "Factory owner (admin).", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "TIMELOCK_DELAY", Label: "Timelock Delay", Description: "Seconds required before mother wallet change.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "pendingMotherWallet", Label: "Pending Mother Wallet", Description: "Mother wallet awaiting timelock.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "motherWalletUnlockTime", Label: "Mother Wallet Unlock Time", Description: "Unix timestamp when pending change can be applied.", Type: "read", Role: "any", Inputs: []Param{}},
	{Name: "deployWallet", Label: "Deploy Wallet", Description: "Deploy user forwarder wallet (relayer only).", Type: "write", Role: "relayer", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "sweepNative", Label: "Sweep Native", Description: "Sweep native token (TRX/ETH/BNB) from a deployed forwarder wallet to mother wallet.", Type: "write", Role: "relayer", Inputs: []Param{{Name: "wallet", Type: "address", Label: "Wallet Address", Placeholder: "T... or 0x..."}}},
	{Name: "sweepToken", Label: "Sweep Token", Description: "Sweep token (TRC20/ERC20) from a deployed forwarder wallet to mother wallet.", Type: "write", Role: "relayer", Inputs: []Param{{Name: "wallet", Type: "address", Label: "Wallet Address", Placeholder: "T... or 0x..."}, {Name: "token", Type: "address", Label: "Token Address", Placeholder: "T... or 0x..."}}},
	{Name: "emergencyWithdrawNative", Label: "Emergency Withdraw Native", Description: "Owner rescues native token from a user wallet to mother wallet.", Type: "write", Role: "owner", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}}},
	{Name: "emergencyWithdrawToken", Label: "Emergency Withdraw Token", Description: "Owner rescues token from a user wallet to mother wallet.", Type: "write", Role: "owner", Inputs: []Param{{Name: "userId", Type: "uint256", Label: "User ID", Placeholder: "12345"}, {Name: "token", Type: "address", Label: "Token Address", Placeholder: "T... or 0x..."}}},
	{Name: "requestMotherWalletChange", Label: "Request Mother Wallet Change", Description: "Start 48h timelock for mother wallet update.", Type: "write", Role: "owner", Inputs: []Param{{Name: "newMotherWallet", Type: "address", Label: "New Mother Wallet", Placeholder: "T... or 0x..."}}},
	{Name: "applyMotherWalletChange", Label: "Apply Mother Wallet Change", Description: "Apply pending mother wallet after timelock.", Type: "write", Role: "owner", Inputs: []Param{}},
	{Name: "cancelMotherWalletChange", Label: "Cancel Mother Wallet Change", Description: "Cancel pending mother wallet change.", Type: "write", Role: "owner", Inputs: []Param{}},
	{Name: "updateRelayer", Label: "Update Relayer", Description: "Set new relayer address.", Type: "write", Role: "owner", Inputs: []Param{{Name: "newRelayer", Type: "address", Label: "New Relayer", Placeholder: "T... or 0x..."}}},
	{Name: "transferOwnership", Label: "Transfer Ownership", Description: "Transfer factory ownership (e.g. to multisig).", Type: "write", Role: "owner", Inputs: []Param{{Name: "newOwner", Type: "address", Label: "New Owner", Placeholder: "T... or 0x..."}}},
}

func FindFunction(name string) (FunctionDef, bool) {
	for _, f := range Functions {
		if f.Name == name {
			return f, true
		}
	}
	return FunctionDef{}, false
}
