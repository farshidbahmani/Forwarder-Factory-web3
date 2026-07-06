//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/deploy"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/tron"
)

func main() {
	envStore := env.New("")
	chain := blockchain.NewClient(envStore)
	tronClient := tron.NewClient(envStore)
	deploySvc := deploy.New(envStore, chain, tronClient)

	ctx := context.Background()
	network := "tronShasta"

	fmt.Println("Completing Tron factory setup (Forwarder + setImplementation)...")
	res, err := deploySvc.Deploy(ctx, network, false, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Factory:", res.FactoryAddress)
	fmt.Println("Implementation:", res.ImplementationAddress)

	contractSvc, err := contract.NewService(chain, tronClient)
	if err != nil {
		log.Fatal(err)
	}
	predicted, err := contractSvc.Call(ctx, network, "getAddress", map[string]string{"userId": "2"})
	if err != nil {
		log.Fatal("getAddress:", err)
	}
	fmt.Println("getAddress(2):", predicted.Result)
}
