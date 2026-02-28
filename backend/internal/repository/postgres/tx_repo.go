package postgres

import (
	"backend/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type txRepository struct {
	db *sql.DB
}

// 依賴注入 (DI)：透過建構子把 db 塞進來
func NewTransactionRepository(db *sql.DB) domain.TransactionRepository {
	return &txRepository{db: db}
}

func (r *txRepository) UpsertTx(ctx context.Context, from, to, hash, token, txType string, amount float64, timestamp int64) error {
	// 這裡我們暫時將 label 預設為 wallet，真正的標籤解析會由 Usecase 層指揮
	walletQuery := `
		INSERT INTO wallets (address, label) VALUES ($1, 'wallet')
		ON CONFLICT (address) DO NOTHING
	`
	r.db.ExecContext(ctx, walletQuery, from)
	r.db.ExecContext(ctx, walletQuery, to)

	txQuery := `
		INSERT INTO transactions (hash, from_address, to_address, amount, token, timestamp, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (hash, from_address, to_address, token) 
		DO UPDATE SET type = EXCLUDED.type
	`
	_, err := r.db.ExecContext(ctx, txQuery, hash, from, to, amount, token, timestamp, txType)
	return err
}

func (r *txRepository) ResolveLabel(ctx context.Context, address string) string {
	// 從 DB 獲取目前的標籤
	var label string
	err := r.db.QueryRowContext(ctx, "SELECT label FROM wallets WHERE address = $1", address).Scan(&label)
	if err != nil {
		return "wallet"
	}
	return label
}

func (r *txRepository) GetGraph(ctx context.Context, input string, isTxHash bool) ([]domain.CytoElement, error) {
	startAddress := input

	if isTxHash {
		err := r.db.QueryRowContext(ctx, "SELECT to_address FROM transactions WHERE hash = $1 LIMIT 1", input).Scan(&startAddress)
		if err != nil {
			return []domain.CytoElement{}, nil
		}
	}

	query := `
		WITH RECURSIVE connected_nodes AS (
			SELECT $1::varchar AS address, 0 AS depth
			UNION
			SELECT 
				CASE WHEN t.from_address = c.address THEN t.to_address ELSE t.from_address END, 
				c.depth + 1
			FROM transactions t 
			JOIN connected_nodes c ON (t.from_address = c.address OR t.to_address = c.address)
			JOIN wallets w ON c.address = w.address
			WHERE c.depth < 4 
			  AND (w.label IN ('wallet', 'HighRisk') OR c.depth = 0)
			  AND t.amount >= 0.01 -- 🛡️ 阻斷邏輯 2：過濾掉小於 0.01 U 的灰塵與釣魚交易
		)
		SELECT DISTINCT t.hash, t.timestamp, t.from_address, w1.label AS from_label,
			t.to_address, w2.label AS to_label, t.amount, t.token, t.type
		FROM transactions t
		JOIN connected_nodes n1 ON t.from_address = n1.address
		JOIN connected_nodes n2 ON t.to_address = n2.address
		JOIN wallets w1 ON t.from_address = w1.address
		JOIN wallets w2 ON t.to_address = w2.address
		WHERE t.amount >= 0.01 -- 🛡️ 確保最終輸出的圖表絕對沒有 0.00 的連線
		LIMIT 150;
	`
	
	rows, err := r.db.QueryContext(ctx, query, startAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var elements []domain.CytoElement
	addedNodes := make(map[string]bool)

	for rows.Next() {
		var hash, fromAddr, fromLabel, toAddr, toLabel, token, txType string
		var amount float64
		var timestamp int64

		if err := rows.Scan(&hash, &timestamp, &fromAddr, &fromLabel, &toAddr, &toLabel, &amount, &token, &txType); err != nil {
			continue
		}

		if !addedNodes[fromAddr] {
			displayLabel := fromAddr
			if len(fromAddr) >= 10 {
				displayLabel = fromAddr[:6] + "..." + fromAddr[len(fromAddr)-4:]
			}
			if fromLabel != "wallet" && fromLabel != "HighRisk" && fromLabel != "Mixer" {
				displayLabel = fromLabel
			}

			elements = append(elements, domain.CytoElement{Data: domain.CytoData{
				ID: fromAddr, Label: displayLabel, Type: fromLabel,
				IsTarget: fromAddr == startAddress || (isTxHash && hash == input),
			}})
			addedNodes[fromAddr] = true
		}

		if !addedNodes[toAddr] {
			displayLabel := toAddr
			if len(toAddr) >= 10 {
				displayLabel = toAddr[:6] + "..." + toAddr[len(toAddr)-4:]
			}
			if toLabel != "wallet" && toLabel != "HighRisk" && toLabel != "Mixer" {
				displayLabel = toLabel
			}

			elements = append(elements, domain.CytoElement{Data: domain.CytoData{
				ID: toAddr, Label: displayLabel, Type: toLabel,
				IsTarget: toAddr == startAddress,
			}})
			addedNodes[toAddr] = true
		}

		// --- 連線處理 ---
		timeStr := time.Unix(timestamp, 0).Format("01/02 15:04")
		formattedAmount := fmt.Sprintf("%.2f %s", amount, token)
		edgeLabel := fmt.Sprintf("%s\n🕒 %s", formattedAmount, timeStr)

		elements = append(elements, domain.CytoElement{
			Data: domain.CytoData{
				ID: hash, Source: fromAddr, Target: toAddr, Amount: formattedAmount,
				Time: timeStr, EdgeLabel: edgeLabel, Type: txType,
			},
		})
	}
	return elements, nil
}
