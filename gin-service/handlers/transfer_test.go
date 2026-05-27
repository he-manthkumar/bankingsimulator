package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"banking/gin-service/client"
	"banking/gin-service/db"
	"banking/gin-service/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	temporalclient "go.temporal.io/sdk/client"
	"gorm.io/gorm"
)


type mockBankingClient struct {
	result *client.SettleTransferResponse
	err    error
}

func (m *mockBankingClient) SettleTransfer(fromAccount, toAccount string, amount float64, transferMode, tpin string) (*client.SettleTransferResponse, error) {
	return m.result, m.err
}

func setupTestDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := database.AutoMigrate(&models.Transfer{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	db.DB = database
}

func setupRouterWithClient(bc client.BankingClient) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transfer", func(c *gin.Context) {
		ProcessTransferWithClient(c, bc)
	})
	r.GET("/transfer/:id", GetTransferStatus)
	r.GET("/transfers", GetAllTransfers)
	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("toJSON: %v", err)
	}
	return bytes.NewBuffer(b)
}

type minimalTemporalClient struct {
	temporalclient.Client
	executeCalled bool
	returnErr     error
}

func (m *minimalTemporalClient) ExecuteWorkflow(
	_ context.Context,
	_ temporalclient.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (temporalclient.WorkflowRun, error) {
	m.executeCalled = true
	return nil, m.returnErr
}

func (m *minimalTemporalClient) Close() {}

func withTemporalClient(t *testing.T, tc temporalclient.Client) {
	t.Helper()
	TemporalClient = tc
	t.Cleanup(func() { TemporalClient = nil })
}


func Test_GetInitialStatus_shouldReturnSuccessWhenModeIsIMPS(t *testing.T) {
	assert.Equal(t, "SUCCESS", getInitialStatus("IMPS"))
}

func Test_GetInitialStatus_shouldReturnPendingWhenModeIsNEFT(t *testing.T) {
	assert.Equal(t, "PENDING", getInitialStatus("NEFT"))
}

func Test_GetInitialStatus_shouldReturnProcessingWhenModeIsRTGS(t *testing.T) {
	assert.Equal(t, "PROCESSING", getInitialStatus("RTGS"))
}

func Test_GetInitialStatus_shouldReturnPendingWhenModeIsUnknown(t *testing.T) {
	assert.Equal(t, "PENDING", getInitialStatus("HEMANTH"))
}


func Test_ProcessTransfer_shouldReturn400WhenRequestBodyIsEmpty(t *testing.T) {
	setupTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transfer", ProcessTransfer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_ProcessTransfer_shouldReturn400WhenTransferModeIsInvalid(t *testing.T) {
	setupTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transfer", ProcessTransfer)

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        500.0,
		"transfer_mode": "WIRE",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}


func Test_ProcessTransfer_shouldReturn400WhenRequestBodyIsMissingRequiredFields(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	fromID := uuid.New().String()

	body := map[string]interface{}{
		"from_account": fromID,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenFromAccountIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	toID := uuid.New().String()

	body := map[string]interface{}{
		"from_account":  "not-a-uuid",
		"to_account":    toID,
		"amount":        500.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenToAccountIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	fromID := uuid.New().String()

	body := map[string]interface{}{
		"from_account":  fromID,
		"to_account":    "invalid-uuid",
		"amount":        500.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenAmountIsZero(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenAmountIsNegative(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        -100.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenTransferModeIsNotNEFTOrRTGSOrIMPS(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        100.0,
		"transfer_mode": "WIRE",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}


func Test_ProcessTransfer_shouldReturn202WhenIMPSTransferIsSettledSuccessfully(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        250.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "SUCCESS", transfer["status"])
}

func Test_ProcessTransfer_shouldReturn202WhenIMPSTransferIsMarkedFailedWhenCoreBankingReturnsError(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{
		err: errors.New("service unavailable"),
	})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        250.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "FAILED", transfer["status"])
}


func Test_ProcessTransfer_shouldReturn202WhenNEFTTransferIsInitiatedAsynchronously(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        1000.0,
		"transfer_mode": "NEFT",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "PENDING", transfer["status"])
}

func Test_ProcessTransfer_shouldReturn202WhenRTGSTransferIsInitiatedAsynchronously(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})
	fromID := uuid.New().String()
	toID := uuid.New().String()

	body := map[string]any{
		"from_account":  fromID,
		"to_account":    toID,
		"amount":        200000.0,
		"transfer_mode": "RTGS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "PROCESSING", transfer["status"])
}


func Test_ProcessTransfer_shouldStartTemporalWorkflowForNEFT(t *testing.T) {
	setupTestDB(t)
	mockTC := &minimalTemporalClient{}
	withTemporalClient(t, mockTC)

	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        1000.0,
		"transfer_mode": "NEFT",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, mockTC.executeCalled, "expected Temporal ExecuteWorkflow to be called for NEFT")
}

func Test_ProcessTransfer_shouldStartTemporalWorkflowForRTGS(t *testing.T) {
	setupTestDB(t)
	mockTC := &minimalTemporalClient{}
	withTemporalClient(t, mockTC)

	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        200000.0,
		"transfer_mode": "RTGS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, mockTC.executeCalled, "expected Temporal ExecuteWorkflow to be called for RTGS")
}

func Test_ProcessTransfer_shouldStillReturn202EvenWhenTemporalWorkflowStartFails(t *testing.T) {
	setupTestDB(t)
	mockTC := &minimalTemporalClient{returnErr: errors.New("temporal unavailable")}
	withTemporalClient(t, mockTC)

	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        1000.0,
		"transfer_mode": "NEFT",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func Test_ProcessTransfer_shouldFallBackToGoroutineWhenTemporalClientIsNil(t *testing.T) {
	setupTestDB(t)

	router := setupRouterWithClient(&mockBankingClient{
		result: &client.SettleTransferResponse{Status: "SUCCESS"},
	})

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        1000.0,
		"transfer_mode": "NEFT",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func Test_GetTransferStatus_shouldReturn400WhenTransferIDIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/bad-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_GetTransferStatus_shouldReturn404WhenTransferIsNotFoundInDatabase(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_GetTransferStatus_shouldReturn200WhenTransferExistsInDatabase(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       500.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
	}
	db.DB.Create(&transfer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/"+transfer.ID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, transfer.ID.String(), respBody["id"])
	assert.Equal(t, transfer.TransferMode, respBody["transfer_mode"])
	assert.Equal(t, "PENDING", respBody["status"])
}


func Test_GetAllTransfers_shouldReturn200WithEmptyListWhenNoTransfersExist(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody []any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 0)
}

func Test_GetAllTransfers_shouldReturn200WithAllTransfersWhenTransfersExist(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	transfer1 := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       1000.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
	}
	transfer2 := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       5000.0,
		TransferMode: "IMPS",
		Status:       "SUCCESS",
	}
	db.DB.Create(&transfer1)
	db.DB.Create(&transfer2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody []any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 2)
}

func Test_GetAllTransfers_shouldReturn200WithTransfersOrderedByCreatedAtDescending(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})

	oldTransfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       1000.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
		CreatedAt:    time.Now().Add(-5 * time.Minute),
	}
	newTransfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       9000.0,
		TransferMode: "RTGS",
		Status:       "SUCCESS",
		CreatedAt:    time.Now(),
	}
	db.DB.Create(&oldTransfer)
	db.DB.Create(&newTransfer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody []map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 2)
	assert.Equal(t, newTransfer.ID.String(), respBody[0]["id"])
	assert.Equal(t, oldTransfer.ID.String(), respBody[1]["id"])
}

func Test_GetAllTransfers_shouldReturn500WhenDatabaseFails(t *testing.T) {
	setupTestDB(t)
	router := setupRouterWithClient(&mockBankingClient{})
	sqlDB, err := db.DB.DB()
	if err == nil {
		sqlDB.Close()
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_SimulateSettlement_shouldMarkTransferAsFailedWhenCoreBankingClientReturnsErrorForNEFT(t *testing.T) {
	setupTestDB(t)

	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       1000.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
	}
	db.DB.Create(&transfer)

	mockClient := &mockBankingClient{err: errors.New("service unavailable")}

	simulateSettlement(transfer, mockClient, "1234")

	time.Sleep(100 * time.Millisecond)

	var updated models.Transfer
	db.DB.First(&updated, "id = ?", transfer.ID)
	assert.Equal(t, "FAILED", updated.Status)
}

func Test_SimulateSettlement_shouldMarkTransferAsFailedWhenCoreBankingClientReturnsErrorForRTGS(t *testing.T) {
	setupTestDB(t)

	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       1000.0,
		TransferMode: "RTGS",
		Status:       "PROCESSING",
	}
	db.DB.Create(&transfer)

	mockClient := &mockBankingClient{err: errors.New("service unavailable")}

	simulateSettlement(transfer, mockClient, "1234")

	time.Sleep(100 * time.Millisecond)

	var updated models.Transfer
	db.DB.First(&updated, "id = ?", transfer.ID)
	assert.Equal(t, "FAILED", updated.Status)
}