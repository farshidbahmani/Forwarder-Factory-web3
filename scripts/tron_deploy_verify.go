//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"forwarder-factory/internal/contract"
	"forwarder-factory/internal/deploy"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/tron"
)

func main() {
	envStore := env.New("")
	chain := blockchain.NewClient(envStore)
	tronClient := tron.NewClient(envStore)

	ctx := context.Background()
	network := "tronShasta"
	if len(os.Args) > 1 {
		network = os.Args[1]
	}

	deploySvc := deploy.New(envStore, chain, tronClient)
	fmt.Println("Deploying ForwarderFactoryTron on", network, "...")
	res, err := deploySvc.Deploy(ctx, network, false)
	if err != nil {
		log.Fatal("deploy:", err)
	}
	fmt.Println("Factory:", res.FactoryAddress)

	contractSvc, err := contract.NewService(chain, tronClient)
	if err != nil {
		log.Fatal(err)
	}

	userID := "999"
	predicted, err := contractSvc.Call(ctx, network, "getAddress", map[string]string{"userId": userID})
	if err != nil {
		log.Fatal("getAddress:", err)
	}
	fmt.Println("getAddress(999):", predicted.Result)

	deployed, err := contractSvc.Call(ctx, network, "deployWallet", map[string]string{"userId": userID})
	if err != nil {
		log.Fatal("deployWallet:", err)
	}
	fmt.Println("deployWallet tx:", deployed.TxHash)

	predicted2, err := contractSvc.Call(ctx, network, "getAddress", map[string]string{"userId": userID})
	if err != nil {
		log.Fatal("getAddress after deploy:", err)
	}
	fmt.Println("getAddress after deploy:", predicted2.Result)

	if fmt.Sprint(predicted.Result) != fmt.Sprint(predicted2.Result) {
		log.Fatal("address mismatch before/after deploy")
	}
	fmt.Println("OK: Tron address prediction matches deployment")
}
