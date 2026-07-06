package tron

import (
	"strings"

	"forwarder-factory/internal/apperror"

	tronaddr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/ethereum/go-ethereum/common"
)

// IsValidAddress accepts base58 (T...) or 41-prefixed hex.
func IsValidAddress(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "T") {
		_, err := tronaddr.Base58ToAddress(s)
		return err == nil
	}
	if strings.HasPrefix(s, "41") || strings.HasPrefix(s, "0x41") {
		_, err := tronaddr.HexToAddress(strings.TrimPrefix(s, "0x"))
		return err == nil
	}
	return common.IsHexAddress(s)
}

// NormalizeAddress returns canonical base58 (T...) form.
func NormalizeAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", apperror.BadRequest("empty address")
	}
	if strings.HasPrefix(s, "T") {
		a, err := tronaddr.Base58ToAddress(s)
		if err != nil {
			return "", apperror.BadRequest("invalid Tron address: " + s)
		}
		return a.String(), nil
	}
	if strings.HasPrefix(s, "41") {
		a, err := tronaddr.HexToAddress(s)
		if err != nil {
			return "", apperror.BadRequest("invalid Tron hex address: " + s)
		}
		return a.String(), nil
	}
	if strings.HasPrefix(s, "0x") && common.IsHexAddress(s) {
		return FormatETHAddress(common.HexToAddress(s)), nil
	}
	return "", apperror.BadRequest("invalid address: " + s)
}

// ToETHAddress converts any supported Tron/EVM address form to 20-byte ABI address.
func ToETHAddress(s string) (common.Address, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "T") {
		a, err := tronaddr.Base58ToAddress(s)
		if err != nil {
			return common.Address{}, apperror.BadRequest("invalid Tron address: " + s)
		}
		b := a.Bytes()
		if len(b) != tronaddr.AddressLength {
			return common.Address{}, apperror.BadRequest("invalid Tron address bytes")
		}
		return common.BytesToAddress(b[1:]), nil
	}
	if strings.HasPrefix(s, "41") {
		a, err := tronaddr.HexToAddress(s)
		if err != nil {
			return common.Address{}, apperror.BadRequest("invalid Tron hex address")
		}
		return common.BytesToAddress(a.Bytes()[1:]), nil
	}
	if common.IsHexAddress(s) {
		return common.HexToAddress(s), nil
	}
	return common.Address{}, apperror.BadRequest("invalid address: " + s)
}

// FormatETHAddress encodes a 20-byte EVM address as Tron base58.
func FormatETHAddress(addr common.Address) string {
	return tronaddr.BytesToAddress(addr.Bytes()).String()
}

// FormatABIOutput formats an ABI address result for Tron networks.
func FormatABIOutput(v interface{}) interface{} {
	switch t := v.(type) {
	case common.Address:
		return FormatETHAddress(t)
	case []byte:
		if len(t) >= 20 {
			return FormatETHAddress(common.BytesToAddress(t[len(t)-20:]))
		}
		return common.BytesToAddress(t).Hex()
	default:
		return v
	}
}
