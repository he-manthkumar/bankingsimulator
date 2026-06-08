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
		TransferID:    uuid.New().String(),
		FromAccount:   uuid.New().String(),
		ToAccount:     uuid.New().String(),
		Amount:        1000.0,
		TransferMode:  mode,
		Tpin:          "1234",
		CorrelationID: uuid.New().String(),
	}
}

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	testSuite := &testsuite.WorkflowTestSuite{}
	return testSuite.NewTestWorkflowEnvironment()
}


func expectedActivity(input SettlementWorkflowInput) activities.SettlementInput {
	return activities.SettlementInput{
		TransferID:    input.TransferID,
		FromAccount:   input.FromAccount,
		ToAccount:     input.ToAccount,
		Amount:        input.Amount,
		TransferMode:  input.TransferMode,
		Tpin:          input.Tpin,
		CorrelationID: input.CorrelationID,
	}
}

func TestSettlementWorkflow_shouldCompleteSuccessfullyForNEFT(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer,
		mock.AnythingOfType("*context.valueCtx"),       
		expectedActivity(input),  
	).Return(nil)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}


func TestSettlementWorkflow_shouldMarkNEFTAsFailedWhenActivityReturnsError(t *testing.T) {
	env := newEnv(t)
	input := newInput("NEFT")

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer,
		mock.AnythingOfType("*context.valueCtx"),       
		expectedActivity(input),
	).Return(errors.New("spring boot down"))

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}


func TestSettlementWorkflow_shouldCompleteSuccessfullyForRTGS(t *testing.T) {
	env := newEnv(t)
	input := newInput("RTGS")

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer,
		mock.AnythingOfType("*context.valueCtx"),
		expectedActivity(input),
	).Return(nil)

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestSettlementWorkflow_shouldMarkRTGSAsFailedWhenActivityReturnsError(t *testing.T) {
	env := newEnv(t)
	input := newInput("RTGS")

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer,
		mock.AnythingOfType("*context.valueCtx"),
		expectedActivity(input),
	).Return(errors.New("core banking unavailable"))

	env.ExecuteWorkflow(SettlementWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}


func TestSettlementWorkflow_shouldReturnErrorForUnsupportedTransferMode(t *testing.T) {
	env := newEnv(t)
	env.ExecuteWorkflow(SettlementWorkflow, newInput("IMPS"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}