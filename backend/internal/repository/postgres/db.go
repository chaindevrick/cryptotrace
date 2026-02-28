package postgres

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// NewConnection 建立並回傳 PostgreSQL 連線
func NewConnection(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ [DB] Failed to connect: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ [DB] Ping failed: %v", err)
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
		log.Fatalf("❌ [DB] Failed to initialize schema: %v", err)
	}

	log.Println("✅ [DB] PostgreSQL Schema Initialized")
	return db
}