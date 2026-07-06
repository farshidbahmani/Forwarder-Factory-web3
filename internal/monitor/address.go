package monitor

import (
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/network"
	"forwarder-factory/internal/tron"

	"github.com/ethereum/go-ethereum/common"
)

func canonicalizeAddress(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperror.BadRequest("empty address")
	}
	if strings.HasPrefix(raw, "T") {
		return tron.NormalizeAddress(raw)
	}
	if common.IsHexAddress(raw) {
		return common.HexToAddress(raw).Hex(), nil
	}
	return "", apperror.BadRequest("invalid address: " + raw)
}

func normalizeForNetwork(raw string, net network.Config) (string, error) {
	if network.IsTron(net) {
		return tron.NormalizeAddress(raw)
	}
	if !common.IsHexAddress(raw) {
		return "", apperror.BadRequest("invalid EVM address: " + raw)
	}
	return common.HexToAddress(raw).Hex(), nil
}

func nativeDecimals(net network.Config) int {
	if network.IsTron(net) {
		return 6
	}
	return 18
}

func tokenDecimals(net network.Config) int {
	if network.IsTron(net) {
		return 6
	}
	return 18
}
