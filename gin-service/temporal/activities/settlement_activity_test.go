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

type mockBankingClient struct {
	result *client.SettleTransferResponse
	err    error
}

func (m *mockBankingClient) SettleTransfer(from, to string, amount float64, mode, tpin, correlationID string) (*client.SettleTransferResponse, error) {
	return m.result, m.err
}

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

	t.Cleanup(func() {
		col.Drop(context.Background())
		mongoClient.Disconnect(context.Background())
	})
}

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

func TestSettleTransfer_shouldUpdateStatusToSuccessWhenBankingClientSucceeds(t *testing.T) {
	setupTestDB(t)
	transfer := seedTransfer(t, "NEFT", "PENDING")

	act := &SettlementActivity{
		BankingClient: &mockBankingClient{
			result: &client.SettleTransferResponse{Status: "SUCCESS"},
		},
	}

	err := act.SettleTransfer(context.Background(), SettlementInput{
		TransferID:   transfer.ID.String(),
		FromAccount:  transfer.FromAccount.String(),
		ToAccount:    transfer.ToAccount.String(),
		Amount:       transfer.Amount,
		TransferMode: "NEFT",
		Tpin:         "1234",
	})

	assert.NoError(t, err)

	updated, findErr := db.Repo.FindByID(context.Background(), transfer.ID)
	assert.NoError(t, findErr)
	assert.Equal(t, "SUCCESS", updated.Status)
}

func TestSettleTransfer_shouldUpdateStatusToFailedAndReturnErrorWhenBankingClientFails(t *testing.T) {
	setupTestDB(t)
	transfer := seedTransfer(t, "RTGS", "PROCESSING")

	act := &SettlementActivity{
		BankingClient: &mockBankingClient{
			err: errors.New("spring boot unavailable"),
		},
	}

	err := act.SettleTransfer(context.Background(), SettlementInput{
		TransferID:   transfer.ID.String(),
		FromAccount:  transfer.FromAccount.String(),
		ToAccount:    transfer.ToAccount.String(),
		Amount:       transfer.Amount,
		TransferMode: "RTGS",
		Tpin:         "1234",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spring boot settlement failed")

	updated, findErr := db.Repo.FindByID(context.Background(), transfer.ID)
	assert.NoError(t, findErr)
	assert.Equal(t, "FAILED", updated.Status)
}

func TestSettleTransfer_shouldReturnErrorImmediatelyWhenTransferIDIsInvalidUUID(t *testing.T) {
	setupTestDB(t)

	act := &SettlementActivity{
		BankingClient: &mockBankingClient{
			result: &client.SettleTransferResponse{Status: "SUCCESS"},
		},
	}

	err := act.SettleTransfer(context.Background(), SettlementInput{
		TransferID:   "not-a-valid-uuid",
		FromAccount:  uuid.New().String(),
		ToAccount:    uuid.New().String(),
		Amount:       500.0,
		TransferMode: "NEFT",
		Tpin:         "1234",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transfer ID")
}

func TestSettleTransfer_shouldUpdateStatusToFailedWhenBankingClientReturnsNilResult(t *testing.T) {
	setupTestDB(t)
	transfer := seedTransfer(t, "NEFT", "PENDING")

	act := &SettlementActivity{
		BankingClient: &mockBankingClient{
			result: nil,
			err:    errors.New("unexpected nil"),
		},
	}

	err := act.SettleTransfer(context.Background(), SettlementInput{
		TransferID:   transfer.ID.String(),
		FromAccount:  transfer.FromAccount.String(),
		ToAccount:    transfer.ToAccount.String(),
		Amount:       transfer.Amount,
		TransferMode: "NEFT",
		Tpin:         "1234",
	})

	assert.Error(t, err)

	updated, findErr := db.Repo.FindByID(context.Background(), transfer.ID)
	assert.NoError(t, findErr)
	assert.Equal(t, "FAILED", updated.Status)
}
