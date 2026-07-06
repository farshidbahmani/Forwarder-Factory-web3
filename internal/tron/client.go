package tron

import (
	"crypto/ecdsa"
	"fmt"
	"strings"
	"sync"
	"time"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	tronclient "github.com/fbsobreira/gotron-sdk/pkg/client"
	tronaddr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	defaultFeeLimit    = 150_000_000 // 150 TRX max fee limit (sun)
	defaultGRPCTimeout = 30 * time.Second
)

type Client struct {
	env   *env.Store
	mu    sync.Mutex
	conns map[string]*tronclient.GrpcClient
}

func NewClient(e *env.Store) *Client {
	return &Client{env: e, conns: map[string]*tronclient.GrpcClient{}}
}

func (c *Client) GRPC(networkName string) (*tronclient.GrpcClient, network.Config, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, net, apperror.BadRequest(err.Error())
	}
	if !network.IsTron(net) {
		return nil, net, apperror.BadRequest(networkName + " is not a Tron network")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if conn, ok := c.conns[networkName]; ok {
		conn.SetTimeout(defaultGRPCTimeout)
		return conn, net, nil
	}

	endpoint := normalizeGRPCEndpoint(network.RPCURL(net, c.env.Get))
	conn := tronclient.NewGrpcClientWithTimeout(endpoint, defaultGRPCTimeout)
	if err := conn.Start(tronclient.GRPCInsecure()); err != nil {
		return nil, net, fmt.Errorf("tron grpc connect %s: %w", endpoint, err)
	}
	if apiKey := c.env.Get("TRONGRID_API_KEY"); apiKey != "" {
		_ = conn.SetAPIKey(apiKey)
	}
	c.conns[networkName] = conn
	return conn, net, nil
}

func (c *Client) Close(networkName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, ok := c.conns[networkName]; ok {
		conn.Stop()
		delete(c.conns, networkName)
	}
}

func (c *Client) PrivateKey(networkName, role string) (*ecdsa.PrivateKey, string, network.Config, error) {
	_, net, err := c.GRPC(networkName)
	if err != nil {
		return nil, "", net, err
	}

	keyName := "DEPLOYER_PRIVATE_KEY"
	if role == "relayer" {
		keyName = "RELAYER_PRIVATE_KEY"
	}
	pkHex := c.env.GetForNetwork(keyName, net.EnvSuffix)
	if pkHex == "" || pkHex == "0x..." {
		return nil, "", net, apperror.BadRequest(
			fmt.Sprintf("Missing %s (or global %s) in .env", network.EnvKey(keyName, net), keyName),
		)
	}
	pkHex = strings.TrimPrefix(pkHex, "0x")
	key, err := crypto.HexToECDSA(pkHex)
	if err != nil {
		return nil, "", net, apperror.BadRequest("Invalid private key in .env")
	}
	addr := tronaddr.PubkeyToAddress(key.PublicKey).String()
	return key, addr, net, nil
}

func (c *Client) FactoryAddress(networkName string) (string, network.Config, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return "", net, apperror.BadRequest(err.Error())
	}
	key := network.EnvKey("FACTORY_ADDRESS", net)
	addrStr := c.env.GetForNetwork("FACTORY_ADDRESS", net.EnvSuffix)
	if addrStr == "" {
		return "", net, apperror.BadRequest("No factory deployed. Set " + key + " in .env")
	}
	normalized, err := NormalizeAddress(addrStr)
	if err != nil {
		return "", net, apperror.BadRequest("Invalid " + key + " in .env")
	}
	return normalized, net, nil
}

func normalizeGRPCEndpoint(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "grpc://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimSuffix(url, "/")

	switch {
	case url == "api.trongrid.io", strings.HasPrefix(url, "api.trongrid.io/"):
		return "grpc.trongrid.io:50051"
	case url == "api.shasta.trongrid.io", strings.HasPrefix(url, "api.shasta.trongrid.io/"):
		return "grpc.shasta.trongrid.io:50051"
	case strings.HasPrefix(url, "grpc.trongrid.io"), strings.HasPrefix(url, "grpc.shasta.trongrid.io"):
		if !strings.Contains(url, ":") {
			return url + ":50051"
		}
		return url
	}
	if !strings.Contains(url, ":") {
		return url + ":50051"
	}
	return url
}
