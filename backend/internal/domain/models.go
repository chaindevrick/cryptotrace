package domain

import "encoding/json"

// --- API 請求結構 ---
type RequestBody struct {
	Address string `json:"address" binding:"required"`
}

// --- 前端 Cytoscape 圖表結構 ---
type CytoData struct {
	ID        string `json:"id,omitempty"`
	Label     string `json:"label,omitempty"`
	Type      string `json:"type,omitempty"`
	Source    string `json:"source,omitempty"`
	Target    string `json:"target,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Time      string `json:"time,omitempty"`
	EdgeLabel string `json:"edgeLabel,omitempty"`
	IsTarget  bool   `json:"isTarget"`
}

type CytoElement struct {
	Data CytoData `json:"data"`
}

// --- 外部 API (Etherscan) 結構 ---
type EtherscanTx struct {
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	TimeStamp       string `json:"timeStamp"`
	ContractAddress string `json:"contractAddress"`
}

type ProxyResponse struct {
	Result struct {
		From string `json:"from"`
	} `json:"result"`
}

type EtherscanContractResponse struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
}

type ContractInfo struct {
	ContractName string `json:"ContractName"`
}