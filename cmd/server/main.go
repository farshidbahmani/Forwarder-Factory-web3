package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/deploy"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/httpapi"
	"forwarder-factory/internal/monitor"
	"forwarder-factory/internal/wallet"
)

func main() {
	envStore := env.New("")
	chain := blockchain.NewClient(envStore)

	contractSvc, err := contract.NewService(chain)
	if err != nil {
		log.Fatal(err)
	}

	app := &httpapi.App{
		Wallets:   wallet.New(envStore, chain),
		Deploy:    deploy.New(envStore, chain),
		Contracts: contractSvc,
		Monitor:   monitor.New(envStore, contractSvc),
	}

	port := 3000
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Forwarder Factory web app running at http://localhost%s", addr)

	// Start each network separately:
	go func() {
		if _, err := app.Monitor.Start(context.Background(), "bscTestnet"); err != nil {
			log.Printf("[monitor] bscTestnet: %v", err)
		}
	}()

	if err := http.ListenAndServe(addr, httpapi.NewRouter(app)); err != nil {
		log.Fatal(err)
	}
}
