package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/domain"
)

type aiClient struct {
	engineURL  string
	httpClient *http.Client
}

func NewClient(engineURL string) domain.AIRepository {
	return &aiClient{
		engineURL: engineURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *aiClient) ExportReport(ctx context.Context, address string) ([]byte, error) {
	// 🚨 核心修復 1：安全地替換掉 URL 結尾的 /analyze，避免路徑疊加錯誤
	baseURL := strings.Replace(c.engineURL, "/analyze", "", 1)
	url := fmt.Sprintf("%s/export_report?address=%s", baseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 🚨 核心修復 2：抓出 Python 真實的報錯訊息，幫助未來除錯
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI engine failed (HTTP %d): %s", resp.StatusCode, string(errBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *aiClient) TriggerAnalysis(ctx context.Context, address string, startTime, endTime int64) error {
	payload := map[string]interface{}{
		"address":   address,
		"startTime": startTime,
		"endTime":   endTime,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal AI payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.engineURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create AI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("呼叫 AI 引擎失敗 (可能是超時或網路斷線): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI engine returned status: %d", resp.StatusCode)
	}

	return nil
}