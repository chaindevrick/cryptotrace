package usecase

import (
	"backend/internal/domain"
	"context"
	"log"
	"strconv"
	"strings"
	
)

type tracerUsecase struct {
	BaseUsecase
}

func NewTracerUsecase(base BaseUsecase) domain.TracerUsecase {
	return &tracerUsecase{BaseUsecase: base}
}

func (uc *tracerUsecase) Trace(ctx context.Context, input string) error {
	input = strings.ToLower(strings.TrimSpace(input))

	if len(input) == 66 {
		log.Printf("🌊 [Tracer] 啟動交易哈希精準溯源: %s", input)
		sender, err := uc.EtherscanRepo.GetTxSender(ctx, input)
		if err != nil || sender == "" { return err }
		sender = strings.ToLower(sender)

		txs, _ := uc.EtherscanRepo.GetTokenTxs(ctx, sender, "desc")
		found := false
		for _, tx := range txs {
			if strings.ToLower(tx.Hash) == input {
				tokenName, exists := uc.Contracts[strings.ToLower(tx.ContractAddress)]
				if !exists { continue }

				amount := uc.FormatAmount(tx.Value)
				txTime, _ := strconv.ParseInt(tx.TimeStamp, 10, 64)
				toAddr := strings.ToLower(tx.To)

				uc.TxRepo.UpsertTx(ctx, sender, toAddr, input, tokenName, "Trace", amount, txTime)
				visited := make(map[string]bool)
				uc.traceFlowStrict(ctx, toAddr, amount, tokenName, txTime, 1, visited)
				found = true
				break
			}
		}
		if !found { return nil }
	} else if len(input) == 42 {
		visited := make(map[string]bool)
		uc.traceFlowStrict(ctx, input, 0, "", 0, 0, visited)
	}
	return nil
}

func (uc *tracerUsecase) traceFlowStrict(ctx context.Context, address string, incomingAmount float64, token string, startTime int64, depth int, visited map[string]bool) {
	if depth >= 4 || visited[address] { return }
	visited[address] = true

	label := uc.ResolveLabel(ctx, address)
	if depth > 0 && label != "wallet" && label != "HighRisk" {
		log.Printf("🛑 [Trace L%d] 資金流入已知實體 (%s)，停止後續追蹤。", depth, label)
		return
	}

	txs, err := uc.EtherscanRepo.GetTokenTxs(ctx, address, "asc")
	if err != nil { return }

	for _, tx := range txs {
		txTime, _ := strconv.ParseInt(tx.TimeStamp, 10, 64)
		amount := uc.FormatAmount(tx.Value)
		tokenName, exists := uc.Contracts[strings.ToLower(tx.ContractAddress)]

		if !exists || strings.ToLower(tx.From) != address || (token != "" && tokenName != token) { continue }
		if depth > 0 && (txTime <= startTime || txTime > startTime+(7*24*3600)) { continue }

		isMatch := false
		if depth == 0 {
			isMatch = amount > 100
		} else {
			ratio := amount / incomingAmount
			isMatch = (ratio >= 0.5 && ratio <= 1.05)
		}

		if isMatch {
			nextAddr := strings.ToLower(tx.To)
			log.Printf("   🎯 [Trace L%d] 精準匹配成功! %.2f %s 流向 -> %s", depth, amount, tokenName, nextAddr)
			
			uc.TxRepo.UpsertTx(ctx, address, nextAddr, tx.Hash, tokenName, "Trace", amount, txTime)
			uc.traceFlowStrict(ctx, nextAddr, amount, tokenName, txTime, depth+1, visited)
			break
		}
	}
}