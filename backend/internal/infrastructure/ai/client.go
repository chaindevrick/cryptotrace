package ai

import (
	"backend/internal/domain"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type aiClient struct {
	engineURL  string
	httpClient *http.Client
}

func NewClient(engineURL string) domain.AIRepository {
	return &aiClient{
		engineURL:  engineURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *aiClient) TriggerAnalysis(ctx context.Context, address string) error {
	if c.engineURL == "" {
		return nil // 如果沒設定 URL 就略過
	}
	
	reqBody, _ := json.Marshal(map[string]string{"address": address})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.engineURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}