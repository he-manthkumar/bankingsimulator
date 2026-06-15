package activities

import (
	"context"
	"errors"
	"os"
	"testing"

	"banking/gin-service/client"
	"banking/gin-service/db"
	"banking/gin-service/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

type mockBankingClient struct {
	err error
}

func (m *mockBankingClient) DebitAccount(accountID string, amount float64, transferRef, tpin string) (*client.DebitResult, error) {
	return &client.DebitResult{}, m.err
}

func (m *mockBankingClient) CreditAccount(accountID string, amount float64, transferRef string) (*client.CreditResult, error) {
	return &client.CreditResult{}, m.err
}

// setupTestDB connects to MongoDB, wires up db.Repo and db.OutboxRepo with
// unique, isolated collections, and registers a cleanup that drops them.
func setupTestDB(t *testing.T) {
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

	outboxColName := "outbox_" + uuid.New().String()[:8]
	outboxCol := mongoClient.Database("banking_test").Collection(outboxColName)
	db.OutboxRepo = &db.MongoOutboxRepo{Col: outboxCol}

	t.Cleanup(func() {
		col.Drop(context.Background())       //nolint:errcheck
		outboxCol.Drop(context.Background()) //nolint:errcheck
		mongoClient.Disconnect(context.Background()) //nolint:errcheck
	})
}

// seedTransfer inserts a transfer into db.Repo and returns it.
func seedTransfer(t *testing.T, mode, status string) models.Transfer {
	t.Helper()
	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  uuid.New(),
		ToAccount:    uuid.New(),
		Amount:       1000.0,
		TransferMode: mode,
		Status:       status,
	}
	if err := db.Repo.Create(context.Background(), &transfer); err != nil {
		t.Fatalf("failed to seed transfer: %v", err)
	}
	return transfer
}

// mongoURI returns the configured MONGO_URI or the local default.
func mongoURI() string {
	if v := os.Getenv("MONGO_URI"); v != "" {
		return v
	}
	return "mongodb://localhost:27017"
}

// ─── DebitAccount ─────────────────────────────────────────────────────────────

func TestDebitAccount_shouldReturnDebitResultWhenBankingClientSucceeds(t *testing.T) {
	setupTestDB(t)

	act := &SettlementActivity{BankingClient: &mockBankingClient{err: nil}}
	input := DebitInput{
		AccountID:   uuid.New().String(),
		Amount:      500.0,
		TransferRef: uuid.New().String(),
		Tpin:        "1234",
	}

	result, err := act.DebitAccount(context.Background(), input)

	assert.NoError(t, err)
	assert.IsType(t, client.DebitResult{}, result)
}

func TestDebitAccount_shouldReturnErrorWhenBankingClientFails(t *testing.T) {
	setupTestDB(t)

	bankErr := errors.New("insufficient funds")
	act := &SettlementActivity{BankingClient: &mockBankingClient{err: bankErr}}
	input := DebitInput{
		AccountID:   uuid.New().String(),
		Amount:      9999.0,
		TransferRef: uuid.New().String(),
		Tpin:        "0000",
	}

	_, err := act.DebitAccount(context.Background(), input)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "debit failed for account")
}

func TestDebitAccount_shouldWrapErrorWithAccountAndTransferRef(t *testing.T) {
	setupTestDB(t)

	accountID := uuid.New().String()
	transferRef := uuid.New().String()
	bankErr := errors.New("account frozen")
	act := &SettlementActivity{BankingClient: &mockBankingClient{err: bankErr}}
	input := DebitInput{
		AccountID:   accountID,
		Amount:      100.0,
		TransferRef: transferRef,
		Tpin:        "5678",
	}

	_, err := act.DebitAccount(context.Background(), input)

	assert.Error(t, err)
	// The wrapped message must include both the account ID and transfer reference.
	assert.ErrorContains(t, err, accountID)
	assert.ErrorContains(t, err, transferRef)
}

// ─── CreditAccount ────────────────────────────────────────────────────────────

func TestCreditAccount_shouldReturnCreditResultWhenBankingClientSucceeds(t *testing.T) {
	setupTestDB(t)

	act := &SettlementActivity{BankingClient: &mockBankingClient{err: nil}}
	input := CreditInput{
		AccountID:   uuid.New().String(),
		Amount:      750.0,
		TransferRef: uuid.New().String(),
	}

	result, err := act.CreditAccount(context.Background(), input)

	assert.NoError(t, err)
	assert.IsType(t, client.CreditResult{}, result)
}

func TestCreditAccount_shouldReturnErrorWhenBankingClientFails(t *testing.T) {
	setupTestDB(t)

	bankErr := errors.New("account not found")
	act := &SettlementActivity{BankingClient: &mockBankingClient{err: bankErr}}
	input := CreditInput{
		AccountID:   uuid.New().String(),
		Amount:      200.0,
		TransferRef: uuid.New().String(),
	}

	_, err := act.CreditAccount(context.Background(), input)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "credit failed for account")
}

// ─── UpdateTransferStatus ─────────────────────────────────────────────────────

func TestUpdateTransferStatus_shouldUpdateStatusInMongoDBWhenValidIDProvided(t *testing.T) {
	setupTestDB(t)

	transfer := seedTransfer(t, "IMPS", "PENDING")
	act := &SettlementActivity{BankingClient: &mockBankingClient{}}

	err := act.UpdateTransferStatus(context.Background(), transfer.ID.String(), "SUCCESS", "")

	assert.NoError(t, err)

	// Verify the status was persisted in MongoDB.
	updated, fetchErr := db.Repo.FindByID(context.Background(), transfer.ID)
	assert.NoError(t, fetchErr)
	assert.Equal(t, "SUCCESS", updated.Status)
}

func TestUpdateTransferStatus_shouldReturnErrorWhenTransferIDIsInvalidUUID(t *testing.T) {
	setupTestDB(t)

	act := &SettlementActivity{BankingClient: &mockBankingClient{}}

	err := act.UpdateTransferStatus(context.Background(), "not-a-uuid", "SUCCESS", "")

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid transfer ID")
}

func TestUpdateTransferStatus_shouldUpdateStatusWithFailureReason(t *testing.T) {
	setupTestDB(t)

	transfer := seedTransfer(t, "NEFT", "PENDING")
	act := &SettlementActivity{BankingClient: &mockBankingClient{}}
	reason := "debit failed: insufficient funds"

	err := act.UpdateTransferStatus(context.Background(), transfer.ID.String(), "FAILED", reason)

	assert.NoError(t, err)

	// Confirm the status transition was persisted.
	updated, fetchErr := db.Repo.FindByID(context.Background(), transfer.ID)
	assert.NoError(t, fetchErr)
	assert.Equal(t, "FAILED", updated.Status)
}

// ─── CleanupOutbox ────────────────────────────────────────────────────────────

func TestCleanupOutbox_shouldReturnNilEvenWhenOutboxEntryDoesNotExist(t *testing.T) {
	setupTestDB(t)

	act := &SettlementActivity{BankingClient: &mockBankingClient{}}

	// No outbox entry seeded — DeleteByTransferID on a missing document is a
	// no-op at the MongoDB layer. The activity must not propagate that as an
	// error and must always return nil.
	err := act.CleanupOutbox(context.Background(), uuid.New().String())

	assert.NoError(t, err)
}

func TestCleanupOutbox_shouldReturnNilWhenDeleteSucceeds(t *testing.T) {
	setupTestDB(t)

	transfer := seedTransfer(t, "RTGS", "PENDING")

	// Insert an outbox entry for the transfer so there is a real document to
	// delete, giving DeleteByTransferID a matched document to remove.
	outboxMongoRepo, ok := db.OutboxRepo.(*db.MongoOutboxRepo)
	if !ok {
		t.Skip("OutboxRepo is not *MongoOutboxRepo — skipping direct insert")
	}

	outboxEntry := &models.OutboxEntry{
		ID:         uuid.New().String(),
		TransferID: transfer.ID.String(),
		Status:     "UNPROCESSED",
	}
	_, insertErr := outboxMongoRepo.Col.InsertOne(context.Background(), outboxEntry)
	assert.NoError(t, insertErr)

	act := &SettlementActivity{BankingClient: &mockBankingClient{}}

	err := act.CleanupOutbox(context.Background(), transfer.ID.String())

	assert.NoError(t, err)
}
