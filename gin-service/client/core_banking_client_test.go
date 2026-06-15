package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestServer(t *testing.T, statusCode int, response interface{}, checks ...func(r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, check := range checks {
			check(r)
		}
		w.WriteHeader(statusCode)
		if response != nil {
			json.NewEncoder(w).Encode(response)
		}
	}))
}

func newTestClient(server *httptest.Server) *CoreBankingClient {
	return &CoreBankingClient{baseURL: server.URL, httpClient: server.Client()}
}


func Test_NewCoreBankingClient_shouldReturnDefaultURLWhenEnvNotSet(t *testing.T) {
	os.Unsetenv("SPRING_BOOT_URL")

	c := NewCoreBankingClient()

	assert.NotNil(t, c)
	assert.Equal(t, "http://localhost:8080", c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func Test_NewCoreBankingClient_shouldReturnEnvURLWhenEnvIsSet(t *testing.T) {
	os.Setenv("SPRING_BOOT_URL", "http://custom-host:9090")
	defer os.Unsetenv("SPRING_BOOT_URL")

	c := NewCoreBankingClient()

	assert.NotNil(t, c)
	assert.Equal(t, "http://custom-host:9090", c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func Test_GetTransaction_shouldReturnTransactionWhenTransactionIdExists(t *testing.T) {
	txnID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	server := newTestServer(t, http.StatusOK, map[string]interface{}{
		"id": txnID, "status": "SUCCESS",
	}, func(r *http.Request) {
		assert.Equal(t, "/transactions/"+txnID, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
	})
	defer server.Close()

	result, err := newTestClient(server).GetTransaction(txnID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, txnID, result["id"])
}

func Test_GetTransaction_shouldReturnNilWhenTransactionIdDoesNotExist(t *testing.T) {
	txnID := "99999999-9999-9999-9999-999999999999"

	server := newTestServer(t, http.StatusNotFound, nil, func(r *http.Request) {
		assert.Equal(t, "/transactions/"+txnID, r.URL.Path)
	})
	defer server.Close()

	result, err := newTestClient(server).GetTransaction(txnID)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func Test_GetTransaction_shouldReturnErrorWhenServerReturnsNon200Non404Status(t *testing.T) {
	server := newTestServer(t, http.StatusInternalServerError, nil)
	defer server.Close()

	result, err := newTestClient(server).GetTransaction("cccccccc-cccc-cccc-cccc-cccccccccccc")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func Test_GetTransaction_shouldReturnErrorWhenServerReturnsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid-json{{{"))
	}))
	defer server.Close()

	result, err := newTestClient(server).GetTransaction("cccccccc-cccc-cccc-cccc-cccccccccccc")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// DEAD CODE — Test_SettleTransfer_shouldReturnErrorWhenServerIsUnreachable commented out.
// func Test_SettleTransfer_shouldReturnErrorWhenServerIsUnreachable(t *testing.T) { ... }


func Test_GetTransaction_shouldReturnErrorWhenServerIsUnreachable(t *testing.T) {
	server := newTestServer(t, http.StatusOK, nil)
	server.Close()

	result, err := newTestClient(server).GetTransaction("cccccccc-cccc-cccc-cccc-cccccccccccc")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "core banking service unavailable")
	assert.Nil(t, result)
}