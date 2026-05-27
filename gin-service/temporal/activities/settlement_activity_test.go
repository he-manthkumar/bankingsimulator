package activities

import (
	"context"
	"errors"
	"testing"

	"banking/gin-service/client"
	"banking/gin-service/db"
	"banking/gin-service/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)


type mockBankingClient struct {
	result *client.SettleTransferResponse
	err    error
}

func (m *mockBankingClient) SettleTransfer(from, to string, amount float64, mode, tpin string) (*client.SettleTransferResponse, error) {
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
	db.DB.Create(&transfer)
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

	var updated models.Transfer
	db.DB.First(&updated, "id = ?", transfer.ID)
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

	var updated models.Transfer
	db.DB.First(&updated, "id = ?", transfer.ID)
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

	var updated models.Transfer
	db.DB.First(&updated, "id = ?", transfer.ID)
	assert.Equal(t, "FAILED", updated.Status)
}
