package workflows

import (
	"errors"
	"testing"

	"banking/gin-service/temporal/activities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
)

func newInput(mode string) SettlementWorkflowInput {
	return SettlementWorkflowInput{
		TransferID:   uuid.New().String(),
		FromAccount:  uuid.New().String(),
		ToAccount:    uuid.New().String(),
		Amount:       1000.0,
		TransferMode: mode,
		Tpin:         "1234",
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	testSuite := &testsuite.WorkflowTestSuite{}
	return testSuite.NewTestWorkflowEnvironment()
}

// ── Happy path tests ─────────────────────────────────────────────────────────

func TestSettlementWorkflow_shouldCompleteSuccessfullyForNEFT(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")
	var act *activities.SettlementActivity

	env.OnActivity(act.DebitAccount, mock.Anything, activities.DebitInput{
		AccountID: input.FromAccount, Amount: input.Amount,
		TransferRef: input.TransferID, Tpin: input.Tpin,
	}).Return(activities.DebitInput{}, nil)
	env.OnActivity(act.CreditAccount, mock.Anything, activities.CreditInput{
		AccountID: input.ToAccount, Amount: input.Amount, TransferRef: input.TransferID,
	}).Return(activities.CreditInput{}, nil)
	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "SUCCESS", "").Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("outbox-ready", nil)
	}, 0)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestSettlementWorkflow_shouldCompleteSuccessfullyForRTGS(t *testing.T) {
	env := newEnv(t)
	input := newInput("RTGS")
	var act *activities.SettlementActivity

	env.OnActivity(act.DebitAccount, mock.Anything, mock.Anything).Return(activities.DebitInput{}, nil)
	env.OnActivity(act.CreditAccount, mock.Anything, mock.Anything).Return(activities.CreditInput{}, nil)
	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "SUCCESS", "").Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("outbox-ready", nil)
	}, 0)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestSettlementWorkflow_shouldCompleteSuccessfullyForIMPS(t *testing.T) {
	env := newEnv(t)
	input := newInput("IMPS")
	var act *activities.SettlementActivity

	// IMPS has zero delay — outbox signal fires, debit + credit proceed immediately
	env.OnActivity(act.DebitAccount, mock.Anything, mock.Anything).Return(activities.DebitInput{}, nil)
	env.OnActivity(act.CreditAccount, mock.Anything, mock.Anything).Return(activities.CreditInput{}, nil)
	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "SUCCESS", "").Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("outbox-ready", nil)
	}, 0)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

// ── Failure path: debit fails ────────────────────────────────────────────────

func TestSettlementWorkflow_shouldFailCleanlyWhenDebitFails(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")
	var act *activities.SettlementActivity

	// Debit fails (e.g. wrong TPIN) — nothing was committed, no compensation needed
	env.OnActivity(act.DebitAccount, mock.Anything, mock.Anything).Return(
		activities.DebitInput{}, errors.New("invalid tpin for account"),
	)
	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "FAILED", mock.AnythingOfType("string")).Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("outbox-ready", nil)
	}, 0)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	// Workflow returns nil — failure is recorded via UpdateTransferStatus activity, not as workflow error
	assert.NoError(t, env.GetWorkflowError())
}

// ── Saga compensation: debit committed, credit fails ─────────────────────────

func TestSettlementWorkflow_shouldCompensateWhenCreditFails(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")
	var act *activities.SettlementActivity

	// Debit succeeds — money leaves sender's account (committed in Spring Boot)
	env.OnActivity(act.DebitAccount, mock.Anything, mock.Anything).Return(activities.DebitInput{}, nil)

	// Credit fails — receiver never gets the money
	env.OnActivity(act.CreditAccount, mock.Anything, activities.CreditInput{
		AccountID: input.ToAccount, Amount: input.Amount, TransferRef: input.TransferID,
	}).Return(activities.CreditInput{}, errors.New("receiver account not found"))

	// Saga compensation: CreditAccount(sender) — returns the money
	env.OnActivity(act.CreditAccount, mock.Anything, activities.CreditInput{
		AccountID:   input.FromAccount,
		Amount:      input.Amount,
		TransferRef: input.TransferID + "-refund",
	}).Return(activities.CreditInput{}, nil)

	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "COMPENSATED", mock.AnythingOfType("string")).Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("outbox-ready", nil)
	}, 0)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	// Workflow returns nil — saga ran cleanly, money returned to sender
	assert.NoError(t, env.GetWorkflowError())
}

// ── Cancellation ─────────────────────────────────────────────────────────────

func TestSettlementWorkflow_shouldCancelWhenUpdateReceivedInWaitingState(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")
	var act *activities.SettlementActivity

	env.OnActivity(act.UpdateTransferStatus, mock.Anything, input.TransferID, "CANCELLED", mock.AnythingOfType("string")).Return(nil)
	env.OnActivity(act.CleanupOutbox, mock.Anything, input.TransferID).Return(nil)

	// Send cancel update while still in WAITING state (before the 30s timer)
	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow("requestCancellation", "", nil)
	}, 1*1000*1000*1000) // 1 second in nanoseconds

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

// ── Unsupported mode ─────────────────────────────────────────────────────────

func TestSettlementWorkflow_shouldReturnErrorForUnsupportedTransferMode(t *testing.T) {
	env := newEnv(t)
	env.ExecuteWorkflow(SettlementWorkflow, newInput("WIRE"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}