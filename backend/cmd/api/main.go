package main

import (
	"backend/internal/delivery/http"
	"backend/internal/infrastructure/ai"
	"backend/internal/infrastructure/etherscan"
	"backend/internal/repository/postgres"
	"backend/internal/usecase"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// 1. 載入環境變數
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	etherscanAPIKey := os.Getenv("ETHERSCAN_API_KEY")
	aiEngineURL := os.Getenv("AI_ENGINE_URL")

	// 2. 初始化基礎設施 (Infrastructure)
	dbConn := postgres.NewConnection(dbURL)
	defer dbConn.Close()

	// 3. 初始化資料庫與外部 API 的實作 (Repository)
	txRepo := postgres.NewTransactionRepository(dbConn)
	etherscanRepo := etherscan.NewClient(etherscanAPIKey)
	aiRepo := ai.NewClient(aiEngineURL)

	// 4. 初始化核心業務邏輯大腦 (Usecases)
	baseUC := usecase.NewBaseUsecase(txRepo, etherscanRepo, aiRepo)
	analyzerUC := usecase.NewAnalyzerUsecase(baseUC)
	tracerUC := usecase.NewTracerUsecase(baseUC)
	graphUC := usecase.NewGraphUsecase(baseUC)

	// 5. 初始化 HTTP 介面 (Delivery)
	httpHandler := http.NewForensicsHandler(analyzerUC, tracerUC, graphUC)
	router := http.NewRouter(httpHandler)

	// 6. 啟動伺服器
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	
	log.Printf("🚀 Enterprise CryptoTrace Backend running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}