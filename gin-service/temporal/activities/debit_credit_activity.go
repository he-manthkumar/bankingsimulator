package activities

import (
	"context"
	"fmt"
	"log"

	"banking/gin-service/client"
)

type DebitInput struct {
	AccountID   string
	Amount      float64
	TransferRef string
	Tpin        string
}

type CreditInput struct {
	AccountID   string
	Amount      float64
	TransferRef string
}

func (a *SettlementActivity) DebitAccount(ctx context.Context, input DebitInput) (client.DebitResult, error) {
	result, err := a.BankingClient.DebitAccount(
		input.AccountID,
		input.Amount,
		input.TransferRef,
		input.Tpin,
	)
	if err != nil {
		return client.DebitResult{}, fmt.Errorf("debit failed for account %s (transfer %s): %w",
			input.AccountID, input.TransferRef, err)
	}

	log.Printf("Activity: debited account %s, new balance %.2f (transfer %s)",
		input.AccountID, result.NewBalance, input.TransferRef)
	return *result, nil
}

func (a *SettlementActivity) CreditAccount(ctx context.Context, input CreditInput) (client.CreditResult, error) {
	result, err := a.BankingClient.CreditAccount(
		input.AccountID,
		input.Amount,
		input.TransferRef,
	)
	if err != nil {
		return client.CreditResult{}, fmt.Errorf("credit failed for account %s (transfer %s): %w",
			input.AccountID, input.TransferRef, err)
	}

	log.Printf("Activity: credited account %s, new balance %.2f (transfer %s)",
		input.AccountID, result.NewBalance, input.TransferRef)
	return *result, nil
}
