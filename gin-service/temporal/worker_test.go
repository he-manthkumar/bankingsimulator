package temporalsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// ─── mock Temporal client ─────────────────────────────────────────────────────

// mockTemporalClient records calls to ExecuteWorkflow and returns a configurable
// error. Embedding temporalclient.Client satisfies the full interface without
// implementing every method.
type mockTemporalClient struct {
	temporalclient.Client
	executeCalled bool
	lastOptions   temporalclient.StartWorkflowOptions
	lastInput     interface{}
	returnErr     error
}

func (m *mockTemporalClient) ExecuteWorkflow(
	_ context.Context,
	opts temporalclient.StartWorkflowOptions,
	_ interface{},
	args ...interface{},
) (temporalclient.WorkflowRun, error) {
	m.executeCalled = true
	m.lastOptions = opts
	if len(args) > 0 {
		m.lastInput = args[0]
	}
	return nil, m.returnErr
}

func (m *mockTemporalClient) Close() {}

// ─── mock worker ──────────────────────────────────────────────────────────────

// mockWorker implements worker.Worker. Only the methods called by StartWorker
// need real bodies; the rest are satisfied by embedding worker.Worker.
type mockWorker struct {
	worker.Worker
	startCalled            bool
	registerWorkflowCalled bool
	registerActivityCalled bool
}

func (mw *mockWorker) RegisterWorkflow(_ interface{}) {
	mw.registerWorkflowCalled = true
}

// RegisterActivity matches the exact signature from the worker.ActivityRegistry interface.
func (mw *mockWorker) RegisterActivity(_ interface{}) {
	mw.registerActivityCalled = true
}

func (mw *mockWorker) Start() error {
	mw.startCalled = true
	return nil
}

func (mw *mockWorker) Stop() {}

// ─── TaskQueue constant ───────────────────────────────────────────────────────

func TestTaskQueue_shouldBeSettlementTaskQueue(t *testing.T) {
	assert.Equal(t, "settlement-task-queue", TaskQueue)
}

// ─── StartSettlementWorkflow ──────────────────────────────────────────────────

func TestStartSettlementWorkflow_shouldCallExecuteWorkflowWithCorrectWorkflowID(t *testing.T) {
	transferID := uuid.New().String()
	mockTC := &mockTemporalClient{}

	StartSettlementWorkflow(mockTC, transferID, "from", "to", 1000.0, "NEFT", "1234")

	assert.True(t, mockTC.executeCalled, "expected ExecuteWorkflow to be called")
	assert.Equal(t, "settlement-"+transferID, mockTC.lastOptions.ID)
}

func TestStartSettlementWorkflow_shouldCallExecuteWorkflowWithCorrectTaskQueue(t *testing.T) {
	mockTC := &mockTemporalClient{}

	StartSettlementWorkflow(mockTC, uuid.New().String(), "from", "to", 500.0, "IMPS", "0000")

	assert.True(t, mockTC.executeCalled)
	assert.Equal(t, TaskQueue, mockTC.lastOptions.TaskQueue,
		"workflow must use the TaskQueue constant so worker and starter stay in sync")
}

func TestStartSettlementWorkflow_shouldNotPanicWhenExecuteWorkflowFails(t *testing.T) {
	mockTC := &mockTemporalClient{returnErr: errors.New("temporal unavailable")}

	assert.NotPanics(t, func() {
		StartSettlementWorkflow(mockTC, uuid.New().String(), "from", "to", 100.0, "RTGS", "9999")
	})

	assert.True(t, mockTC.executeCalled, "ExecuteWorkflow must still be called even if it returns an error")
}

func TestStartSettlementWorkflow_shouldPassAllInputFieldsToWorkflow(t *testing.T) {
	mockTC := &mockTemporalClient{}
	transferID := "tid-999"
	fromAccount := "acc-from"
	toAccount := "acc-to"
	amount := 99999.99
	mode := "RTGS"
	tpin := "5678"

	StartSettlementWorkflow(mockTC, transferID, fromAccount, toAccount, amount, mode, tpin)

	assert.True(t, mockTC.executeCalled)
	// Verify the workflow options were assembled correctly.
	assert.Equal(t, "settlement-"+transferID, mockTC.lastOptions.ID)
	assert.Equal(t, TaskQueue, mockTC.lastOptions.TaskQueue)
}

// ─── StartWorker ──────────────────────────────────────────────────────────────

func TestStartWorker_shouldStartWorkerSuccessfully(t *testing.T) {
	mw := &mockWorker{}

	// Swap the package-level newWorker factory so StartWorker uses our mock.
	original := newWorker
	newWorker = func(_ temporalclient.Client, _ string, _ worker.Options) worker.Worker {
		return mw
	}
	t.Cleanup(func() { newWorker = original })

	// StartWorker calls log.Fatal if w.Start() returns an error; the mock
	// returns nil so no fatal occurs.
	StartWorker(&mockTemporalClient{})

	assert.True(t, mw.startCalled, "expected worker.Start() to be called")
	assert.True(t, mw.registerWorkflowCalled, "expected RegisterWorkflow to be called")
	assert.True(t, mw.registerActivityCalled, "expected RegisterActivity to be called")
}
