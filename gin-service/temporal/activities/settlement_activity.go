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


func (a *SettlementActivity) UpdateTransferStatus(ctx context.Context, transferID, status, reason string) error {
	id, err := uuid.Parse(transferID)
	if err != nil {
		return fmt.Errorf("invalid transfer ID %q: %w", transferID, err)
	}
	if dbErr := db.Repo.UpdateStatusWithReason(ctx, id, status, reason); dbErr != nil {
		return fmt.Errorf("failed to update transfer %s status to %s: %w", transferID, status, dbErr)
	}
	log.Printf("Activity: transfer %s status → %s (reason: %q)", transferID, status, reason)
	return nil
}

func (a *SettlementActivity) CleanupOutbox(ctx context.Context, transferID string) error {
	if err := db.OutboxRepo.DeleteByTransferID(ctx, transferID); err != nil {
		log.Printf("Activity: failed to delete outbox entry for transfer %s: %v", transferID, err)
	}
	log.Printf("Activity: outbox entry cleaned up for transfer %s", transferID)
	return nil
}

