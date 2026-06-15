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


type DebitResult struct {
	AccountID   string  `json:"accountId"`
	NewBalance  float64 `json:"newBalance"`
	TransferRef string  `json:"transferRef"`
}

type CreditResult struct {
	AccountID   string  `json:"accountId"`
	NewBalance  float64 `json:"newBalance"`
	TransferRef string  `json:"transferRef"`
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

type debitRequest struct {
	Amount      float64 `json:"amount"`
	TransferRef string  `json:"transferRef"`
	Tpin        string  `json:"tpin"`
}

func (c *CoreBankingClient) DebitAccount(accountID string, amount float64, transferRef, tpin string) (*DebitResult, error) {
	body, err := json.Marshal(debitRequest{Amount: amount, TransferRef: transferRef, Tpin: tpin})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal debit request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/accounts/%s/debit", c.baseURL, accountID),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("core banking service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("debit failed with status %d for account %s", resp.StatusCode, accountID)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read debit response: %w", err)
	}

	var result DebitResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse debit response: %w", err)
	}

	log.Printf("Debit successful: account %s, amount %.2f, ref %s", accountID, amount, transferRef)
	return &result, nil
}

type creditRequest struct {
	Amount      float64 `json:"amount"`
	TransferRef string  `json:"transferRef"`
}

func (c *CoreBankingClient) CreditAccount(accountID string, amount float64, transferRef string) (*CreditResult, error) {
	body, err := json.Marshal(creditRequest{Amount: amount, TransferRef: transferRef})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credit request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/accounts/%s/credit", c.baseURL, accountID),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("core banking service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("credit failed with status %d for account %s", resp.StatusCode, accountID)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read credit response: %w", err)
	}

	var result CreditResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse credit response: %w", err)
	}

	log.Printf("Credit successful: account %s, amount %.2f, ref %s", accountID, amount, transferRef)
	return &result, nil
}