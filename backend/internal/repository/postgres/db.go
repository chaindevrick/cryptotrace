package postgres

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// DBConfig 定義資料庫的連線參數，明確拆分帳號與密碼
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewConnection 建立並回傳 PostgreSQL 連線
func NewConnection(cfg DBConfig) *sql.DB {
	// 動態組合安全的 Key-Value 格式 DSN (Data Source Name)
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	log.Printf("🔌 [DB] 正在連線至 PostgreSQL (Host: %s, User: %s)...", cfg.Host, cfg.User)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ [DB] 連線物件建立失敗: %v", err)
	}

	// 測試實際連線與密碼驗證
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ [DB] Ping 失敗 (請檢查帳號、密碼或白名單設定): %v", err)
	}

	// 初始化 Schema
	schema := `
	CREATE TABLE IF NOT EXISTS wallets (
		address VARCHAR(42) PRIMARY KEY,
		label VARCHAR(100) DEFAULT 'wallet'
	);
	CREATE TABLE IF NOT EXISTS transactions (
		id SERIAL PRIMARY KEY,
		hash VARCHAR(66) NOT NULL,
		from_address VARCHAR(42) REFERENCES wallets(address),
		to_address VARCHAR(42) REFERENCES wallets(address),
		amount DOUBLE PRECISION,
		token VARCHAR(20),
		timestamp BIGINT,
		type VARCHAR(50) DEFAULT 'TRANSFER',
		UNIQUE(hash, from_address, to_address, token)
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("❌ [DB] Schema 初始化失敗: %v", err)
	}

	log.Println("✅ [DB] PostgreSQL 連線成功且 Schema 初始化完畢")
	return db
}