package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	tronclient "github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

const (
	defaultTxWaitTimeout = 90 * time.Second
	deployTxWaitTimeout  = 3 * time.Minute
)

func waitMined(ctx context.Context, grpc *tronclient.GrpcClient, txID []byte) (*core.TransactionInfo, error) {
	return waitMinedWithTimeout(ctx, grpc, txID, defaultTxWaitTimeout)
}

func waitDeployMined(ctx context.Context, grpc *tronclient.GrpcClient, txID []byte) (*core.TransactionInfo, error) {
	return waitMinedWithTimeout(ctx, grpc, txID, deployTxWaitTimeout)
}

func waitMinedWithTimeout(ctx context.Context, grpc *tronclient.GrpcClient, txID []byte, timeout time.Duration) (*core.TransactionInfo, error) {
	id := hex.EncodeToString(txID)
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := grpc.GetTransactionInfoByIDCtx(ctx, id)
		if err != nil {
			lastErr = err
			time.Sleep(3 * time.Second)
			continue
		}

		ok, failed, msg := txMinedStatus(info)
		if ok {
			return info, nil
		}
		if failed {
			if msg == "" {
				msg = "transaction reverted"
			}
			return nil, fmt.Errorf("transaction failed: %s", msg)
		}

		time.Sleep(3 * time.Second)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("timeout waiting for transaction %s: %v", id, lastErr)
	}
	return nil, fmt.Errorf("timeout waiting for transaction %s", id)
}

func txMinedStatus(info *core.TransactionInfo) (ok, failed bool, msg string) {
	if info == nil || info.BlockNumber <= 0 {
		return false, false, ""
	}
	if info.Receipt != nil {
		switch info.Receipt.Result {
		case core.Transaction_Result_SUCCESS:
			return true, false, ""
		case core.Transaction_Result_DEFAULT:
			// still confirming
		default:
			return false, true, info.Receipt.Result.String()
		}
	}
	if info.Result == core.TransactionInfo_FAILED {
		msg = string(info.ResMessage)
		return false, true, msg
	}
	if len(info.ContractAddress) > 0 || info.Result == core.TransactionInfo_SUCESS {
		return true, false, ""
	}
	if info.BlockNumber > 0 {
		return true, false, ""
	}
	return false, false, ""
}
