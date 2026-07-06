package env

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type Store struct {
	path string
	mu   sync.RWMutex
}

func New(path string) *Store {
	if path == "" {
		path = filepath.Join(mustWd(), ".env")
	}
	s := &Store{path: path}
	s.Reload()
	return s
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func (s *Store) Reload() {
	_ = godotenv.Load(s.path)
}

func (s *Store) Get(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readFile()[key]
}

func (s *Store) GetForNetwork(baseKey, envSuffix string) string {
	if v := s.Get(envSuffix + "_" + baseKey); v != "" {
		return v
	}
	// legacy env prefix (BSC_TESTNET → BNB_TESTNET)
	if envSuffix == "BNB_TESTNET" {
		if v := s.Get("BSC_TESTNET_" + baseKey); v != "" {
			return v
		}
	}
	// legacy: BASE_KEY_NETWORK_SUFFIX
	if v := s.Get(baseKey + "_" + envSuffix); v != "" {
		return v
	}
	return s.Get(baseKey)
}

func (s *Store) readFile() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readFileLocked()
}

func (s *Store) readFileLocked() map[string]string {
	out := map[string]string{}
	f, err := os.Open(s.path)
	if err != nil {
		return out
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			out[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
		}
	}
	return out
}
