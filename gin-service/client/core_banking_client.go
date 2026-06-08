package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"			
)

type CoreBankingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewCoreBankingClient() *CoreBankingClient {
	baseURL := os.Getenv("SPRING_BOOT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &CoreBankingClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type SettleTransferRequest struct {
	FromAccount   string  `json:"fromAccount"`
	ToAccount     string  `json:"toAccount"`
	Amount        float64 `json:"amount"`
	TransferMode  string  `json:"transferMode"`
	Tpin          string  `json:"tpin"`
	CorrelationID string  `json:"correlationId"`
}

type SettleTransferResponse struct {
	ID           string  `json:"id"`
	FromAccount  string  `json:"fromAccount"`
	ToAccount    string  `json:"toAccount"`
	Amount       float64 `json:"amount"`
	TransferMode string  `json:"transferMode"`
	Status       string  `json:"status"`
}

func (c *CoreBankingClient) SettleTransfer(fromAccount, toAccount string, amount float64, transferMode string, tpin string, correlationID string) (*SettleTransferResponse, error) {
	payload := SettleTransferRequest{
		FromAccount:   fromAccount,
		ToAccount:     toAccount,
		Amount:        amount,
		TransferMode:  transferMode,
		Tpin:          tpin,
		CorrelationID: correlationID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/transactions/settle", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("core banking service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("settlement failed with status: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result SettleTransferResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("Settlement successful via CoreBankingClient: txn %s", result.ID)
	return &result, nil
}

func (c *CoreBankingClient) GetTransaction(transactionID string) (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/transactions/" + transactionID)
	if err != nil {
		return nil, fmt.Errorf("core banking service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("core banking returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}