package workflows

import (
	"fmt"
	"time"

	"banking/gin-service/temporal/activities"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type SettlementWorkflowInput struct {
	TransferID   string
	FromAccount  string
	ToAccount    string
	Amount       float64
	TransferMode string
	Tpin         string
}

func SettlementWorkflow(ctx workflow.Context, input SettlementWorkflowInput) error {

	var delay time.Duration
	switch input.TransferMode {
	case "NEFT":
		delay = 30 * time.Second 
	case "RTGS":
		delay = 15 * time.Second 
	default:
		return fmt.Errorf("unsupported transfer mode for settlement workflow: %s", input.TransferMode)
	}

	if err := workflow.Sleep(ctx, delay); err != nil {
		return err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)


	var act *activities.SettlementActivity
	return workflow.ExecuteActivity(ctx, act.SettleTransfer, activities.SettlementInput{
		TransferID:   input.TransferID,
		FromAccount:  input.FromAccount,
		ToAccount:    input.ToAccount,
		Amount:       input.Amount,
		TransferMode: input.TransferMode,
		Tpin:         input.Tpin,
	}).Get(ctx, nil)
}
