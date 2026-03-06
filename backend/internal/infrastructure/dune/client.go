package dune

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/duneanalytics/duneapi-client-go/config"
	"github.com/duneanalytics/duneapi-client-go/dune"
	"github.com/duneanalytics/duneapi-client-go/models"
)

// SyncLabels 從 Dune API 抓取標籤並寫入 PostgreSQL
func SyncLabels(db *sql.DB, apiKey string) error {
	queryID := 6786625
	log.Printf("🌐 [Dune Sync] 開始透過官方 SDK 同步標籤 (Query ID: %d)...", queryID)

	// 1. 初始化 Dune 官方 SDK Client
	env := config.FromAPIKey(apiKey)
	client := dune.NewDuneClient(env)

	// 2. 使用 models.ExecuteRequest 來包裝查詢參數
	req := models.ExecuteRequest{
		QueryID: queryID,
	}

	// ✨ 3. 執行 Query 並取得結果 (加入自動重試與延遲機制)
	var rows []map[string]any
	var err error
	maxRetries := 3 // 設定最多重試 3 次

	for i := 1; i <= maxRetries; i++ {
		rows, err = client.RunQueryGetRows(req)
		if err == nil {
			break // 成功取得資料，跳出迴圈！
		}

		log.Printf("⚠️ [Dune Sync] 第 %d 次呼叫 Dune API 失敗 (原因: %v)，準備重試...", i, err)
		if i < maxRetries {
			// 休息 3 秒後再試，讓 CPU 資源恢復或避開瞬間的網路抖動
			time.Sleep(3 * time.Second)
		}
	}

	// 如果重試了 3 次還是失敗，就真的報錯放棄
	if err != nil {
		return fmt.Errorf("Dune SDK 執行失敗 (已重試 %d 次): %v", maxRetries, err)
	}

	// 4. 準備寫入 PostgreSQL
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 準備 Upsert 語法：遇到重複的地址就更新標籤
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO wallets (address, label) 
        VALUES ($1, $2) 
        ON CONFLICT (address) DO UPDATE SET label = EXCLUDED.label;
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	// 5. 解析 SDK 回傳的 Rows (結構為 []map[string]any)
	for _, row := range rows {
		// 安全取值與轉型
		addrObj, okAddr := row["address"]
		nameObj, okName := row["name"]
		catObj, okCat := row["category"]

		if !okAddr || !okName || !okCat {
			continue // 略過欄位不完整的資料
		}

		addr := strings.ToLower(fmt.Sprintf("%v", addrObj))
		name := fmt.Sprintf("%v", nameObj)
		category := fmt.Sprintf("%v", catObj)

		// 轉換成 CryptoTrace 系統標準標籤
		systemLabel := name
		if category == "mixer" {
			systemLabel = "Mixer"
		} else if category == "cex" {
			systemLabel = "Exchange"
		} else if category == "hack" || category == "phishing" {
			systemLabel = "HighRisk"
		}

		// 某些合約地址可能是 \x 開頭，將其轉為標準 0x
		if strings.HasPrefix(addr, "\\x") {
			addr = "0x" + addr[2:]
		}

		// 寫入資料庫
		_, err = stmt.ExecContext(ctx, addr, systemLabel)
		if err != nil {
			log.Printf("⚠️ 寫入失敗 %s: %v", addr, err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("✅ [Dune Sync] 成功透過官方 SDK 同步並寫入 %d 個實體標籤！", count)
	return nil
}
