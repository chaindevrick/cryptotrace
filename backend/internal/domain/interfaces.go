package domain

import (
	"context"
)

// ==========================================
// Repository Interfaces (與外部世界溝通的合約)
// ==========================================

// TransactionRepository 負責與 PostgreSQL 溝通
type TransactionRepository interface {
	UpsertTx(ctx context.Context, from, to, hash, token, txType string, amount float64, timestamp int64) error
	GetGraph(ctx context.Context, rootAddress string, isTxHash bool) ([]CytoElement, error)
	ResolveLabel(ctx context.Context, address string) string
}

// EtherscanRepository 負責與區塊鏈瀏覽器 API 溝通
type EtherscanRepository interface {
	GetTokenTxs(ctx context.Context, address string, sort string) ([]EtherscanTx, error)
	GetTxSender(ctx context.Context, txHash string) (string, error)
	GetContractName(ctx context.Context, address string) (string, error)
}

// AIRepository 負責觸發 Python AI 引擎
type AIRepository interface {
	TriggerAnalysis(ctx context.Context, address string) error
}

// ==========================================
// Usecase Interfaces (大腦業務邏輯的合約)
// ==========================================

// AnalyzerUsecase 專職處理 Broad (全景分析與 AI 檢測)
type AnalyzerUsecase interface {
	Analyze(ctx context.Context, address string) (int, error)
}

// TracerUsecase 專職處理 Flow (單筆資金精準溯源)
type TracerUsecase interface {
	Trace(ctx context.Context, input string) error
}

// GraphUsecase 專職處理圖表資料生成
type GraphUsecase interface {
	GetGraph(ctx context.Context, address string) ([]CytoElement, error)
}