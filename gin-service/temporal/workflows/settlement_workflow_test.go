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


func TestSettlementWorkflow_shouldCompleteSuccessfullyForNEFT(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(SettlementWorkflow, newInput("NEFT"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestSettlementWorkflow_shouldMarkNEFTAsFailedWhenActivityReturnsError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var act *activities.SettlementActivity

	env.OnActivity(act.SettleTransfer, mock.Anything, mock.Anything).
		Return(errors.New("spring boot down"))

	env.ExecuteWorkflow(SettlementWorkflow, newInput("NEFT"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}


func TestSettlementWorkflow_shouldCompleteSuccessfullyForRTGS(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(SettlementWorkflow, newInput("RTGS"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestSettlementWorkflow_shouldMarkRTGSAsFailedWhenActivityReturnsError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var act *activities.SettlementActivity
	env.OnActivity(act.SettleTransfer, mock.Anything, mock.Anything).
		Return(errors.New("core banking unavailable"))

	env.ExecuteWorkflow(SettlementWorkflow, newInput("RTGS"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}


func TestSettlementWorkflow_shouldReturnErrorForUnsupportedTransferMode(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(SettlementWorkflow, newInput("IMPS"))

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}
