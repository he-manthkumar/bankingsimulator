package activities

import (
	"context"
	"fmt"
	"log"

	"banking/gin-service/client"
	"banking/gin-service/db"

	"github.com/google/uuid"
)

type SettlementActivity struct {
	BankingClient client.BankingClient
}
type SettlementInput struct {
	TransferID   string
	FromAccount  string
	ToAccount    string
	Amount       float64
	TransferMode string
	Tpin         string
}

func (a *SettlementActivity) SettleTransfer(ctx context.Context, input SettlementInput) error {
	result, err := a.BankingClient.SettleTransfer(
		input.FromAccount,
		input.ToAccount,
		input.Amount,
		input.TransferMode,
		input.Tpin,
	)
	status := "FAILED"
	if err == nil && result != nil {
		status = result.Status
	}

	transferID, parseErr := uuid.Parse(input.TransferID)
	if parseErr != nil {
		return fmt.Errorf("invalid transfer ID %q: %w", input.TransferID, parseErr)
	}

	if dbErr := db.Repo.UpdateStatus(ctx, transferID, status); dbErr != nil {
		log.Printf("Temporal activity: failed to update transfer %s in DB: %v", input.TransferID, dbErr)
	}

	if err != nil {
		return fmt.Errorf("spring boot settlement failed for transfer %s: %w", input.TransferID, err)
	}

	log.Printf("Temporal activity: %s transfer %s settled with status: %s",
		input.TransferMode, input.TransferID, status)
	return nil
}
