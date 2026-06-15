package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"banking/gin-service/client"
	"banking/gin-service/db"
	"banking/gin-service/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	temporalclient "go.temporal.io/sdk/client"
)


type mockBankingClient struct {
	err error
}


func (m *mockBankingClient) DebitAccount(accountID string, amount float64, transferRef, tpin string) (*client.DebitResult, error) {
	return &client.DebitResult{}, m.err
}

func (m *mockBankingClient) CreditAccount(accountID string, amount float64, transferRef string) (*client.CreditResult, error) {
	return &client.CreditResult{}, m.err
}


func setupTestDB(t *testing.T) *mongo.Client {
	t.Helper()

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("failed to connect to MongoDB: %v", err)
	}
	if err := mongoClient.Ping(context.Background(), nil); err != nil {
		t.Fatalf("MongoDB ping failed — is MongoDB running? %v", err)
	}
	colName := "transfers_" + uuid.New().String()[:8]
	col := mongoClient.Database("banking_test").Collection(colName)
	db.Repo = &db.MongoTransferRepo{Col: col}

	outboxCol := mongoClient.Database("banking_test").Collection("outbox_" + uuid.New().String()[:8])
	db.OutboxRepo = &db.MongoOutboxRepo{Col: outboxCol}

	db.MongoClient = mongoClient

	t.Cleanup(func() {
		col.Drop(context.Background())
		outboxCol.Drop(context.Background())
		mongoClient.Disconnect(context.Background())
	})

	return mongoClient
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transfer", ProcessTransfer)
	r.GET("/transfer/:id", GetTransferStatus)
	r.GET("/transfers", GetAllTransfers)
	r.PUT("/transfer/:id/cancel", CancelTransfer)
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

type TemporalClient_ struct {
	temporalclient.Client
	executeCalled bool
	returnErr     error
}

func (m *TemporalClient_) ExecuteWorkflow(
	_ context.Context,
	_ temporalclient.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (temporalclient.WorkflowRun, error) {
	m.executeCalled = true
	return nil, m.returnErr
}

func (m *TemporalClient_) Close() {}

// mockUpdateHandle implements temporalclient.WorkflowUpdateHandle and is returned
// by the UpdateWorkflow mock so the handler can call handle.Get(...).
type mockUpdateHandle struct {
	getErr error
}

func (h *mockUpdateHandle) WorkflowID() string { return "" }
func (h *mockUpdateHandle) RunID() string       { return "" }
func (h *mockUpdateHandle) UpdateID() string    { return "" }
func (h *mockUpdateHandle) Get(ctx context.Context, valuePtr interface{}) error {
	return h.getErr
}

// cancelTemporalClient_ extends TemporalClient_ with UpdateWorkflow support for
// CancelTransfer handler tests.
type cancelTemporalClient_ struct {
	TemporalClient_
	updateErr    error // error returned from UpdateWorkflow itself (transport-level)
	handleGetErr error // error returned from handle.Get (business-level rejection)
	updateCalled bool
	lastWorkflowID string
	lastUpdateName string
}

func (m *cancelTemporalClient_) UpdateWorkflow(
	_ context.Context,
	options temporalclient.UpdateWorkflowOptions,
) (temporalclient.WorkflowUpdateHandle, error) {
	m.updateCalled = true
	m.lastWorkflowID = options.WorkflowID
	m.lastUpdateName = options.UpdateName
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &mockUpdateHandle{getErr: m.handleGetErr}, nil
}

func withTemporalClient(t *testing.T, tc temporalclient.Client) {
	t.Helper()
	TemporalClient = tc
	t.Cleanup(func() { TemporalClient = nil })
}


func Test_ProcessTransfer_shouldReturn400WhenRequestBodyIsEmpty(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func Test_ProcessTransfer_shouldReturn400WhenRequestBodyIsMissingRequiredFields(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]interface{}{"from_account": uuid.New().String()}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenFromAccountIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]interface{}{
		"from_account":  "not-a-uuid",
		"to_account":    uuid.New().String(),
		"amount":        500.0,
		"transfer_mode": "NEFT",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenToAccountIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]interface{}{
		"from_account":  uuid.New().String(),
		"to_account":    "invalid-uuid",
		"amount":        500.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenAmountIsZero(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenAmountIsNegative(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        -100.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_ProcessTransfer_shouldReturn400WhenTransferModeIsNotNEFTOrRTGSOrIMPS(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        100.0,
		"transfer_mode": "WIRE",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}


func Test_ProcessTransfer_shouldReturn202WithPendingStatusForIMPS(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        250.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "PENDING", transfer["status"])
}

func Test_ProcessTransfer_shouldReturn202WithPendingStatusForNEFT(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

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
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "PENDING", transfer["status"])
}

func Test_ProcessTransfer_shouldReturn202WithPendingStatusForRTGS(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

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
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "message")
	assert.Contains(t, respBody, "transfer")

	transfer, ok := respBody["transfer"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "PENDING", transfer["status"])
}


func Test_ProcessTransfer_shouldStartTemporalWorkflowForNEFT(t *testing.T) {
	setupTestDB(t)
	mockTC := &TemporalClient_{}
	withTemporalClient(t, mockTC)
	r := setupRouter()

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
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, mockTC.executeCalled, "expected Temporal ExecuteWorkflow to be called for NEFT")
}

func Test_ProcessTransfer_shouldStartTemporalWorkflowForRTGS(t *testing.T) {
	setupTestDB(t)
	mockTC := &TemporalClient_{}
	withTemporalClient(t, mockTC)
	r := setupRouter()

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
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, mockTC.executeCalled, "expected Temporal ExecuteWorkflow to be called for RTGS")
}

func Test_ProcessTransfer_shouldStartTemporalWorkflowForIMPS(t *testing.T) {
	setupTestDB(t)
	mockTC := &TemporalClient_{}
	withTemporalClient(t, mockTC)
	r := setupRouter()

	body := map[string]any{
		"from_account":  uuid.New().String(),
		"to_account":    uuid.New().String(),
		"amount":        500.0,
		"transfer_mode": "IMPS",
		"tpin":          "1234",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/transfer", toJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, mockTC.executeCalled, "expected Temporal ExecuteWorkflow to be called for IMPS")
}

func Test_ProcessTransfer_shouldStillReturn202EvenWhenTemporalWorkflowStartFails(t *testing.T) {
	setupTestDB(t)
	mockTC := &TemporalClient_{returnErr: errors.New("temporal unavailable")}
	withTemporalClient(t, mockTC)
	r := setupRouter()

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
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func Test_ProcessTransfer_shouldReturn202EvenWhenTemporalClientIsNil(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

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
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "NEFT transfer initiated")
}


func Test_GetTransferStatus_shouldReturn400WhenTransferIDIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/bad-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_GetTransferStatus_shouldReturn404WhenTransferIsNotFoundInDatabase(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_GetTransferStatus_shouldReturn200WhenTransferExistsInDatabase(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       500.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
	}
	db.Repo.Create(context.Background(), &transfer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfer/"+transfer.ID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, transfer.ID.String(), respBody["id"])
	assert.Equal(t, transfer.TransferMode, respBody["transfer_mode"])
	assert.Equal(t, "PENDING", respBody["status"])
}


func Test_GetAllTransfers_shouldReturn200WithEmptyListWhenNoTransfersExist(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respBody []any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 0)
}

func Test_GetAllTransfers_shouldReturn200WithAllTransfersWhenTransfersExist(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

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
	db.Repo.Create(context.Background(), &transfer1)
	db.Repo.Create(context.Background(), &transfer2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respBody []any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 2)
}

func Test_GetAllTransfers_shouldReturn200WithTransfersOrderedByCreatedAtDescending(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

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
	db.Repo.Create(context.Background(), &oldTransfer)
	db.Repo.Create(context.Background(), &newTransfer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var respBody []map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Len(t, respBody, 2)
	assert.Equal(t, newTransfer.ID.String(), respBody[0]["id"])
	assert.Equal(t, oldTransfer.ID.String(), respBody[1]["id"])
}

func Test_GetAllTransfers_shouldReturn500WhenDatabaseFails(t *testing.T) {
	mongoClient := setupTestDB(t)
	r := setupRouter()

	mongoClient.Disconnect(context.Background())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/transfers", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

// ─── CancelTransfer tests ────────────────────────────────────────────────────

func Test_CancelTransfer_shouldReturn400WhenTransferIDIsNotValidUUID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/transfer/not-a-uuid/cancel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

// CancelTransfer checks TemporalClient == nil before touching the DB; when no
// Temporal client is set the handler returns 503 immediately.
func Test_CancelTransfer_shouldReturn503WhenTemporalClientIsNil(t *testing.T) {
	setupTestDB(t)
	// Ensure TemporalClient is nil (default after setupTestDB cleanup).
	TemporalClient = nil
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/transfer/"+uuid.New().String()+"/cancel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_CancelTransfer_shouldReturn200WhenCancellationSucceeds(t *testing.T) {
	setupTestDB(t)

	// Seed a PENDING transfer so there is something to cancel.
	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       500.0,
		TransferMode: "NEFT",
		Status:       "PENDING",
	}
	assert.NoError(t, db.Repo.Create(context.Background(), &transfer))

	// Mock that UpdateWorkflow succeeds and handle.Get returns nil.
	mockTC := &cancelTemporalClient_{}
	withTemporalClient(t, mockTC)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/transfer/"+transfer.ID.String()+"/cancel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockTC.updateCalled, "expected UpdateWorkflow to be called")
	assert.Equal(t, "settlement-"+transfer.ID.String(), mockTC.lastWorkflowID)
	assert.Equal(t, "requestCancellation", mockTC.lastUpdateName)

	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "cancellation confirmed", respBody["message"])
}

func Test_CancelTransfer_shouldReturn409WhenWorkflowRejectsCancel(t *testing.T) {
	setupTestDB(t)

	// Mock UpdateWorkflow succeeds at the transport level but handle.Get returns
	// a business-level error (e.g. "cannot cancel: transfer is already DEBITING").
	mockTC := &cancelTemporalClient_{
		handleGetErr: errors.New("cannot cancel: transfer is already DEBITING"),
	}
	withTemporalClient(t, mockTC)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/transfer/"+uuid.New().String()+"/cancel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}

func Test_CancelTransfer_shouldReturn500WhenUpdateWorkflowFails(t *testing.T) {
	setupTestDB(t)

	// Mock UpdateWorkflow failing at the transport level (e.g. network error).
	mockTC := &cancelTemporalClient_{
		updateErr: errors.New("temporal server unreachable"),
	}
	withTemporalClient(t, mockTC)
	r := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/transfer/"+uuid.New().String()+"/cancel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var respBody map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Contains(t, respBody, "error")
}