package usecase

import (
	"backend/internal/domain"
	"context"
	"strconv"
	"strings"
	"time"
)

type analyzerUsecase struct {
	BaseUsecase // 繼承 BaseUsecase 的所有屬性與方法
}

func NewAnalyzerUsecase(base BaseUsecase) domain.AnalyzerUsecase {
	return &analyzerUsecase{BaseUsecase: base}
}

func (uc *analyzerUsecase) Analyze(ctx context.Context, address string) (int, error) {
	txs, err := uc.EtherscanRepo.GetTokenTxs(ctx, address, "desc")
	if err != nil {
		return 0, err
	}

	count := 0
	limit := 50
	if len(txs) < limit {
		limit = len(txs)
	}

	for _, tx := range txs[:limit] {
		tokenName, exists := uc.Contracts[strings.ToLower(tx.ContractAddress)]
		if !exists { continue }

		amount := uc.FormatAmount(tx.Value)
		if amount <= 0 { continue }

		timestamp, _ := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err := uc.TxRepo.UpsertTx(ctx, strings.ToLower(tx.From), strings.ToLower(tx.To), tx.Hash, tokenName, "TRANSFER", amount, timestamp); err == nil {
			count++
		}
	}

	if count > 0 {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = uc.AIRepo.TriggerAnalysis(bgCtx, address)
		}()
	}

	return count, nil
}