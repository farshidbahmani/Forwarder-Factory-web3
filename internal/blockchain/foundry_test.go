package blockchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForgeBinFindsDefaultInstall(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	defaultPath := filepath.Join(home, ".foundry", "bin", "forge")
	if _, err := os.Stat(defaultPath); err != nil {
		t.Skip("forge not installed at", defaultPath)
	}

	t.Setenv("FORGE", "")
	t.Setenv("FOUNDRY_BIN", "")
	t.Setenv("PATH", "")

	got, err := ForgeBin()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultPath {
		t.Fatalf("got %q want %q", got, defaultPath)
	}
}
