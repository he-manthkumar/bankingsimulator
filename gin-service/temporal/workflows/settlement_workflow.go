package workflows

import (
	"fmt"
	"time"

	"banking/gin-service/temporal/activities"
	"banking/gin-service/temporal/saga"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type WorkflowState struct {
	State            string `json:"state"`
	ElapsedSeconds   int64  `json:"elapsed_seconds"`
	RemainingSeconds int64  `json:"remaining_seconds"`
	CancelRequested  bool   `json:"cancel_requested"`
	DebitCompleted   bool   `json:"debit_completed"`
	OutboxReady      bool   `json:"outbox_ready"`
	FailureReason    string `json:"failure_reason,omitempty"`
}

type SettlementWorkflowInput struct {
	TransferID   string
	FromAccount  string
	ToAccount    string
	Amount       float64
	TransferMode string
	Tpin         string
}

func SettlementWorkflow(ctx workflow.Context, input SettlementWorkflowInput) error {
	startTime := workflow.Now(ctx)

	var delay time.Duration
	switch input.TransferMode {
	case "NEFT":
		delay = 30 * time.Second
	case "RTGS":
		delay = 15 * time.Second
	case "IMPS":
		delay = 0
	default:
		return fmt.Errorf("unsupported transfer mode: %s", input.TransferMode)
	}

	state := &WorkflowState{State: "WAITING"}

	if err := workflow.SetQueryHandler(ctx, "getStatus", func() (WorkflowState, error) {
		elapsed := int64(workflow.Now(ctx).Sub(startTime).Seconds())
		remaining := int64(delay.Seconds()) - elapsed
		if remaining < 0 {
			remaining = 0
		}
		state.ElapsedSeconds = elapsed
		state.RemainingSeconds = remaining
		return *state, nil
	}); err != nil {
		return fmt.Errorf("failed to register getStatus query handler: %w", err)
	}

	cancelCh := workflow.NewChannel(ctx)
	if err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		"requestCancellation",
		func(ctx workflow.Context) error {
			state.CancelRequested = true
			cancelCh.Send(ctx, struct{}{})
			return nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context) error {
				if state.State != "WAITING" {
					return temporal.NewApplicationError(
						fmt.Sprintf("cannot cancel: transfer is already %s", state.State),
						"REJECTED",
					)
				}
				return nil
			},
		},
	); err != nil {
		return fmt.Errorf("failed to register requestCancellation update handler: %w", err)
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 40 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, ao)
	var act *activities.SettlementActivity

	outboxReadyCh := workflow.GetSignalChannel(ctx, "outbox-ready")
	timerFuture := workflow.NewTimer(ctx, delay)

	selector := workflow.NewSelector(ctx)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
	})
	selector.AddReceive(outboxReadyCh, func(c workflow.ReceiveChannel, more bool) {
		state.OutboxReady = true
	})
	selector.AddReceive(cancelCh, func(c workflow.ReceiveChannel, more bool) {
	})
	selector.Select(ctx)

	if state.CancelRequested {
		state.State = "CANCELLED"
		_ = workflow.ExecuteActivity(actCtx, act.UpdateTransferStatus, input.TransferID, "CANCELLED", "cancelled by user before settlement").Get(ctx, nil)
		_ = workflow.ExecuteActivity(actCtx, act.CleanupOutbox, input.TransferID).Get(ctx, nil)
		return nil
	}
	s := &saga.Saga{}
	state.State = "DEBITING"

	s.AddCompensation(func(sagaCtx workflow.Context) error {
		sagaAo := workflow.ActivityOptions{
			StartToCloseTimeout: 40 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts:    5,
				InitialInterval:    5 * time.Second,
				BackoffCoefficient: 2.0,
			},
		}
		return workflow.ExecuteActivity(
			workflow.WithActivityOptions(sagaCtx, sagaAo),
			act.CreditAccount,
			activities.CreditInput{
				AccountID:   input.FromAccount,
				Amount:      input.Amount,
				TransferRef: input.TransferID + "-refund",
			},
		).Get(sagaCtx, nil)
	})

	debitErr := workflow.ExecuteActivity(actCtx, act.DebitAccount, activities.DebitInput{
		AccountID:   input.FromAccount,
		Amount:      input.Amount,
		TransferRef: input.TransferID,
		Tpin:        input.Tpin,
	}).Get(ctx, nil)

	if debitErr != nil {
		state.State = "FAILED"
		state.FailureReason = debitErr.Error()
		_ = workflow.ExecuteActivity(actCtx, act.UpdateTransferStatus, input.TransferID, "FAILED", debitErr.Error()).Get(ctx, nil)
		_ = workflow.ExecuteActivity(actCtx, act.CleanupOutbox, input.TransferID).Get(ctx, nil)
		return nil
	}

	state.DebitCompleted = true

	state.State = "CREDITING"

	creditErr := workflow.ExecuteActivity(actCtx, act.CreditAccount, activities.CreditInput{
		AccountID:   input.ToAccount,
		Amount:      input.Amount,
		TransferRef: input.TransferID,
	}).Get(ctx, nil)

	if creditErr != nil {

		state.State = "COMPENSATED"
		state.FailureReason = creditErr.Error()
		s.Compensate(ctx)
		_ = workflow.ExecuteActivity(actCtx, act.UpdateTransferStatus, input.TransferID, "COMPENSATED",
			fmt.Sprintf("receiver credit failed — sender refunded: %s", creditErr.Error())).Get(ctx, nil)
		_ = workflow.ExecuteActivity(actCtx, act.CleanupOutbox, input.TransferID).Get(ctx, nil)
		return nil
	}

	state.State = "COMPLETED"
	_ = workflow.ExecuteActivity(actCtx, act.UpdateTransferStatus, input.TransferID, "SUCCESS", "").Get(ctx, nil)
	_ = workflow.ExecuteActivity(actCtx, act.CleanupOutbox, input.TransferID).Get(ctx, nil)
	return nil
}
