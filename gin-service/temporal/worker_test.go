package temporalsetup

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type mockTemporalClient struct {
	temporalclient.Client
	executeCalled  bool
	lastWorkflowID string
	returnErr      error
}

func (m *mockTemporalClient) ExecuteWorkflow(
	_ context.Context,
	opts temporalclient.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (temporalclient.WorkflowRun, error) {
	m.executeCalled = true
	m.lastWorkflowID = opts.ID
	return nil, m.returnErr
}

func (m *mockTemporalClient) Close() {}

type mockWorker struct {
    worker.Worker                  
    startCalled            bool
    registerWorkflowCalled bool
    registerActivityCalled bool
}

func (m *mockWorker) RegisterWorkflow(_ interface{}) {
	m.registerWorkflowCalled = true 
	}
func (m *mockWorker) RegisterActivity(_ interface{}) { 
	m.registerActivityCalled = true 
	}
func (m *mockWorker) Start() error{ 
	m.startCalled = true; return nil 
	}

func TestStartWorker_shouldRegisterWorkflowAndActivityAndStart(t *testing.T) {
    mock := &mockWorker{}

    original := newWorker
    newWorker = func(_ temporalclient.Client, _ string, _ worker.Options) worker.Worker {
        return mock
    }
    defer func() { newWorker = original }()

    StartWorker(nil)

    assert.True(t, mock.registerWorkflowCalled, "should register workflow")
    assert.True(t, mock.registerActivityCalled, "should register activity")
    assert.True(t, mock.startCalled, "should call Start()")
}

func TestStartSettlementWorkflow_shouldCallExecuteWorkflowWithCorrectID(t *testing.T) {
	transferID := uuid.New().String()
	mockTC := &mockTemporalClient{}

	StartSettlementWorkflow(
		mockTC,
		transferID,
		uuid.New().String(),
		uuid.New().String(),
		1000.0,
		"NEFT",
		"1234",
	)

	assert.True(t, mockTC.executeCalled)
	assert.Equal(t, "settlement-"+transferID, mockTC.lastWorkflowID,
		"workflow ID should be deterministic and prefixed with 'settlement-'")
}

func TestStartSettlementWorkflow_shouldNotPanicWhenTemporalReturnsError(t *testing.T) {
	mockTC := &mockTemporalClient{returnErr: assert.AnError}

	assert.NotPanics(t, func() {
		StartSettlementWorkflow(
			mockTC,
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
			500.0,
			"RTGS",
			"5678",
		)
	})
}

type capturingTemporalClient struct {
	temporalclient.Client
	lastTaskQueue string
}

func (c *capturingTemporalClient) ExecuteWorkflow(
	_ context.Context,
	opts temporalclient.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (temporalclient.WorkflowRun, error) {
	c.lastTaskQueue = opts.TaskQueue
	return nil, nil
}

func (c *capturingTemporalClient) Close() {}

func TestStartSettlementWorkflow_shouldUseTaskQueueConstant(t *testing.T) {
	var capturedQueue string

	capturingClient := &capturingTemporalClient{}
	StartSettlementWorkflow(
		capturingClient,
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		250.0,
		"NEFT",
		"0000",
	)

	capturedQueue = capturingClient.lastTaskQueue
	assert.Equal(t, TaskQueue, capturedQueue,
		"workflow must use the TaskQueue constant so worker and starter stay in sync")
}
