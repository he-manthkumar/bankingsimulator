package outbox

import (
	"context"
	"errors"
	"testing"

	"banking/gin-service/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	temporalclient "go.temporal.io/sdk/client"
)

// ─── mock Temporal client ─────────────────────────────────────────────────────

// mockTemporalClient records calls made to SignalWorkflow and returns a
// configurable error. Embed temporalclient.Client so the struct satisfies the
// full interface without implementing every method.
type mockTemporalClient struct {
	temporalclient.Client
	signalCalled   bool
	lastWorkflowID string
	lastSignalName string
	returnErr      error
}

func (m *mockTemporalClient) SignalWorkflow(
	_ context.Context,
	workflowID, runID, signalName string,
	arg interface{},
) error {
	m.signalCalled = true
	m.lastWorkflowID = workflowID
	m.lastSignalName = signalName
	return m.returnErr
}

// ─── signalWorkflow unit tests ────────────────────────────────────────────────

// TestSignalWorkflow_shouldCallSignalWorkflowWithCorrectWorkflowID verifies that
// signalWorkflow constructs the workflow ID as "settlement-<transferID>".
func TestSignalWorkflow_shouldCallSignalWorkflowWithCorrectWorkflowID(t *testing.T) {
	transferID := uuid.New().String()
	mockTC := &mockTemporalClient{}

	entry := models.OutboxEntry{
		ID:         uuid.New().String(),
		TransferID: transferID,
		Status:     "UNPROCESSED",
	}

	signalWorkflow(mockTC, entry)

	assert.True(t, mockTC.signalCalled, "expected SignalWorkflow to be called")
	assert.Equal(t, "settlement-"+transferID, mockTC.lastWorkflowID)
}

// TestSignalWorkflow_shouldCallSignalWorkflowWithOutboxReadySignalName verifies
// that the signal name sent to Temporal is exactly "outbox-ready".
func TestSignalWorkflow_shouldCallSignalWorkflowWithOutboxReadySignalName(t *testing.T) {
	transferID := uuid.New().String()
	mockTC := &mockTemporalClient{}

	entry := models.OutboxEntry{
		ID:         uuid.New().String(),
		TransferID: transferID,
		Status:     "UNPROCESSED",
	}

	signalWorkflow(mockTC, entry)

	assert.True(t, mockTC.signalCalled, "expected SignalWorkflow to be called")
	assert.Equal(t, "outbox-ready", mockTC.lastSignalName)
}

// TestSignalWorkflow_shouldNotPanicWhenTemporalClientReturnsError verifies that
// a Temporal transport error is logged and swallowed — it must never cause a
// panic or propagate up to the caller.
func TestSignalWorkflow_shouldNotPanicWhenTemporalClientReturnsError(t *testing.T) {
	transferID := uuid.New().String()
	mockTC := &mockTemporalClient{returnErr: errors.New("temporal server unavailable")}

	entry := models.OutboxEntry{
		ID:         uuid.New().String(),
		TransferID: transferID,
		Status:     "UNPROCESSED",
	}

	assert.NotPanics(t, func() {
		signalWorkflow(mockTC, entry)
	})

	// SignalWorkflow should still have been called despite the error.
	assert.True(t, mockTC.signalCalled)
}

// TestSignalWorkflow_shouldUseTransferIDFromOutboxEntry verifies that the
// workflow ID is derived from the entry's TransferID field, not a hardcoded
// value, so different entries produce different workflow IDs.
func TestSignalWorkflow_shouldUseTransferIDFromOutboxEntry(t *testing.T) {
	transferID1 := uuid.New().String()
	transferID2 := uuid.New().String()

	mockTC1 := &mockTemporalClient{}
	signalWorkflow(mockTC1, models.OutboxEntry{ID: uuid.New().String(), TransferID: transferID1})

	mockTC2 := &mockTemporalClient{}
	signalWorkflow(mockTC2, models.OutboxEntry{ID: uuid.New().String(), TransferID: transferID2})

	assert.Equal(t, "settlement-"+transferID1, mockTC1.lastWorkflowID)
	assert.Equal(t, "settlement-"+transferID2, mockTC2.lastWorkflowID)
	assert.NotEqual(t, mockTC1.lastWorkflowID, mockTC2.lastWorkflowID,
		"each entry must produce a unique workflowID")
}
