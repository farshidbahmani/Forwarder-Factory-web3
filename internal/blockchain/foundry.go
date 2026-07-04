package blockchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ForgeBin returns the path to the forge executable.
func ForgeBin() (string, error) {
	return resolveFoundryTool("forge", "FORGE")
}

// CastBin returns the path to the cast executable.
func CastBin() (string, error) {
	return resolveFoundryTool("cast", "CAST")
}

func resolveFoundryTool(name, envKey string) (string, error) {
	if p := os.Getenv(envKey); p != "" {
		if fileExists(p) {
			return p, nil
		}
	}

	if dir := os.Getenv("FOUNDRY_BIN"); dir != "" {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p, nil
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".foundry", "bin", name)
		if fileExists(p) {
			return p, nil
		}
	}

	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	return "", fmt.Errorf(
		"%s not found in PATH — install Foundry (https://book.getfoundry.sh/getting-started/installation) or set %s or FOUNDRY_BIN in .env",
		name, envKey,
	)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
