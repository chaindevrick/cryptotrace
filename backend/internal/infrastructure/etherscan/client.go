package etherscan

import (
	"backend/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type etherscanClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) domain.EtherscanRepository {
	return &etherscanClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *etherscanClient) GetTokenTxs(ctx context.Context, address string, sort string) ([]domain.EtherscanTx, error) {
	url := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=tokentx&address=%s&startblock=0&endblock=99999999&sort=%s&apikey=%s", address, sort, c.apiKey)
	
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var esResp struct {
		Status string               `json:"status"`
		Result []domain.EtherscanTx `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, err
	}
	return esResp.Result, nil
}

func (c *etherscanClient) GetTxSender(ctx context.Context, txHash string) (string, error) {
	url := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=proxy&action=eth_getTransactionByHash&txhash=%s&apikey=%s", txHash, c.apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	var proxyData domain.ProxyResponse
	json.NewDecoder(resp.Body).Decode(&proxyData)
	return proxyData.Result.From, nil
}

func (c *etherscanClient) GetContractName(ctx context.Context, address string) (string, error) {
	time.Sleep(200 * time.Millisecond) // 避免 Rate Limit
	url := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=contract&action=getsourcecode&address=%s&apikey=%s", address, c.apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	var esResp domain.EtherscanContractResponse
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err == nil && esResp.Status == "1" {
		var infos []domain.ContractInfo
		if err := json.Unmarshal(esResp.Result, &infos); err == nil && len(infos) > 0 && infos[0].ContractName != "" {
			return "📜 " + infos[0].ContractName, nil
		}
	}
	return "", nil
}