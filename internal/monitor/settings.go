package monitor

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func (s *PushSettings) tokenMin(token string) (float64, bool) {
	token = strings.ToLower(strings.TrimSpace(token))
	for _, t := range s.Tokens {
		if strings.EqualFold(t.Token, token) {
			return t.MinValueTransfer, true
		}
	}
	return 0, false
}

func (s *PushSettings) nativeMeetsMin(amount *big.Int, decimals int) bool {
	if s.NativeMinValueTransfer <= 0 {
		return amount.Sign() > 0
	}
	return humanAmount(amount, decimals) >= s.NativeMinValueTransfer
}

func (s *PushSettings) tokenMeetsMin(token string, amount *big.Int, decimals int) bool {
	min, ok := s.tokenMin(token)
	if !ok {
		return false
	}
	if min <= 0 {
		return amount.Sign() > 0
	}
	return humanAmount(amount, decimals) >= min
}

func humanAmount(amount *big.Int, decimals int) float64 {
	if amount == nil || amount.Sign() == 0 {
		return 0
	}
	f := new(big.Float).SetInt(amount)
	div := new(big.Float).SetFloat64(pow10(decimals))
	v, _ := new(big.Float).Quo(f, div).Float64()
	return v
}

func pow10(n int) float64 {
	out := 1.0
	for i := 0; i < n; i++ {
		out *= 10
	}
	return out
}

func normalizeTokenKey(addr string) string {
	if common.IsHexAddress(addr) {
		return common.HexToAddress(addr).Hex()
	}
	return strings.ToLower(strings.TrimSpace(addr))
}
