package usecase

import (
	"backend/internal/domain"
	"context"
	"log"
	"strconv"
	"sync"
)

// BaseUsecase 提供共用的依賴與輔助函式 (組合模式)
type BaseUsecase struct {
	TxRepo        domain.TransactionRepository
	EtherscanRepo domain.EtherscanRepository
	AIRepo        domain.AIRepository

	LabelCache map[string]string
	CacheMutex *sync.RWMutex
	Contracts  map[string]string
}

func NewBaseUsecase(tr domain.TransactionRepository, er domain.EtherscanRepository, ar domain.AIRepository) BaseUsecase {
	return BaseUsecase{
		TxRepo:        tr,
		EtherscanRepo: er,
		AIRepo:        ar,
		LabelCache: map[string]string{
			"0x28c6c06298d514db089934071355e22af1d4a120": "Binance 14",
			"0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be": "Binance Deposit",
			"0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640": "Binance 3",
			"0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": "Mixer",
			"0x0000000000000000000000000000000000000000": "Null Address",
		},
		CacheMutex: &sync.RWMutex{},
		Contracts: map[string]string{
			"0xdac17f958d2ee523a2206206994597c13d831ec7": "USDT",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": "USDC",
		},
	}
}

func (b *BaseUsecase) FormatAmount(valueStr string) float64 {
	val, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}
	return val / 1000000.0
}

func (b *BaseUsecase) ResolveLabel(ctx context.Context, address string) string {
	b.CacheMutex.RLock()
	if val, ok := b.LabelCache[address]; ok {
		b.CacheMutex.RUnlock()
		return val
	}
	b.CacheMutex.RUnlock()

	dbLabel := b.TxRepo.ResolveLabel(ctx, address)
	if dbLabel != "wallet" {
		return dbLabel
	}

	contractName, err := b.EtherscanRepo.GetContractName(ctx, address)
	if err == nil && contractName != "" {
		b.CacheMutex.Lock()
		b.LabelCache[address] = contractName
		b.CacheMutex.Unlock()
		log.Printf("🔍 [Base] 發現智能合約: %s -> %s", address, contractName)
		return contractName
	}

	b.CacheMutex.Lock()
	b.LabelCache[address] = "wallet"
	b.CacheMutex.Unlock()
	return "wallet"
}