package monitor

import (
	"testing"

	"forwarder-factory/internal/network"
)

func bnbTestnet(t *testing.T) network.Config {
	t.Helper()
	net, err := network.Get("bnbTestnet")
	if err != nil {
		t.Fatal(err)
	}
	return net
}

func TestRegistryPushWithIDs(t *testing.T) {
	r := NewRegistry()
	net := bnbTestnet(t)

	affected, err := r.Upsert("bnbTestnet", net, WalletPushRequest{
		Network: "bnbTestnet",
		Setting: PushSettings{MinNativeBalance: 5},
		Wallets: map[string]string{
			"1": "0xDD281B850B8a32F2dca05f5058b6656d32C2998f",
			"2": "0x55d398326f99059fF775485246999027B3197955",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if affected != 2 {
		t.Fatalf("affected = %d, want 2", affected)
	}
	if r.Count("bnbTestnet") != 2 {
		t.Fatalf("count = %d, want 2", r.Count("bnbTestnet"))
	}

	view := r.View().Networks["bnbTestnet"]
	if view.Wallets["1"] != "0xDD281B850B8a32F2dca05f5058b6656d32C2998f" {
		t.Fatalf("wallet 1 = %q", view.Wallets["1"])
	}
	if view.Setting.MinNativeBalance != 5 {
		t.Fatalf("minNativeBalance = %v, want 5", view.Setting.MinNativeBalance)
	}

	// Re-pushing the same id/address pair is a no-op.
	affected, err = r.Upsert("bnbTestnet", net, WalletPushRequest{
		Network: "bnbTestnet",
		Wallets: map[string]string{"1": "0xDD281B850B8a32F2dca05f5058b6656d32C2998f"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if affected != 0 {
		t.Fatalf("affected = %d, want 0", affected)
	}
}

func TestRegistryRemoveByIDOrAddress(t *testing.T) {
	r := NewRegistry()
	net := bnbTestnet(t)

	_, err := r.Upsert("bnbTestnet", net, WalletPushRequest{
		Network: "bnbTestnet",
		Wallets: map[string]string{
			"1": "0xDD281B850B8a32F2dca05f5058b6656d32C2998f",
			"2": "0x55d398326f99059fF775485246999027B3197955",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	removed, err := r.Remove("bnbTestnet", net, []string{"1"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed by id = %d, want 1", removed)
	}

	removed, err = r.Remove("bnbTestnet", net, []string{"0x55d398326f99059ff775485246999027b3197955"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed by address = %d, want 1", removed)
	}
	if r.Count("bnbTestnet") != 0 {
		t.Fatalf("count = %d, want 0", r.Count("bnbTestnet"))
	}
}

func TestRegistryWalletID(t *testing.T) {
	r := NewRegistry()
	net := bnbTestnet(t)

	_, err := r.Upsert("bnbTestnet", net, WalletPushRequest{
		Network: "bnbTestnet",
		Wallets: map[string]string{"42": "0xdd281b850b8a32f2dca05f5058b6656d32c2998f"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id, ok := r.WalletID("bnbTestnet", "0xDD281B850B8a32F2dca05f5058b6656d32C2998f")
	if !ok || id != "42" {
		t.Fatalf("WalletID = %q, %v; want \"42\", true", id, ok)
	}
}
