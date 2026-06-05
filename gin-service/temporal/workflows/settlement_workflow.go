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
		delay = 10 * time.Minute // next batch window
	case "RTGS":
		delay = 5 * time.Minute  // real-time gross settlement
	default:
		return fmt.Errorf("unsupported transfer mode: %s", input.TransferMode)
	}

	if err := workflow.Sleep(ctx, delay); err != nil {
		return err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
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
