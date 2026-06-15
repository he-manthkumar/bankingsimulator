package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"banking/gin-service/db"
	"banking/gin-service/models"
	temporalsetup "banking/gin-service/temporal"
	"banking/gin-service/temporal/workflows"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	temporalclient "go.temporal.io/sdk/client"
)

var TemporalClient temporalclient.Client

type TransferRequest struct {
	FromAccount  string  `json:"from_account" binding:"required"`
	ToAccount    string  `json:"to_account" binding:"required"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	TransferMode string  `json:"transfer_mode" binding:"required,oneof=NEFT RTGS IMPS"`
	Tpin         string  `json:"tpin" binding:"required"`
}

func ProcessTransfer(c *gin.Context) {
	ProcessTransferInternal(c)
}

func ProcessTransferInternal(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fromID, err := uuid.Parse(req.FromAccount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_account UUID"})
		return
	}
	toID, err := uuid.Parse(req.ToAccount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_account UUID"})
		return
	}

	transfer := models.Transfer{
		ID:           uuid.New(),
		FromAccount:  fromID,
		ToAccount:    toID,
		Amount:       req.Amount,
		TransferMode: req.TransferMode,
		Status:       "PENDING",
		CreatedAt:    time.Now(),
	}

	outboxEntry := models.OutboxEntry{
		ID:            uuid.New().String(),
		TransferID:    transfer.ID.String(),
		CorrelationID: "settlement-" + transfer.ID.String(),
		Action:        "SETTLE",
		Status:        "UNPROCESSED",
		Payload: models.OutboxPayload{
			FromAccount:  transfer.FromAccount.String(),
			ToAccount:    transfer.ToAccount.String(),
			Amount:       transfer.Amount,
			TransferMode: transfer.TransferMode,
		},
		CreatedAt: time.Now(),
		Attempts:  0,
	}

	ctx := c.Request.Context()
	session, err := db.MongoClient.StartSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start db session"})
		return
	}
	defer session.EndSession(ctx)

	_, txErr := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := db.Repo.CreateWithSession(sessCtx, session, &transfer); err != nil {
			return nil, fmt.Errorf("failed to create transfer: %w", err)
		}
		if err := db.OutboxRepo.CreateWithSession(sessCtx, session, &outboxEntry); err != nil {
			return nil, fmt.Errorf("failed to create outbox entry: %w", err)
		}
		return nil, nil
	})

	if txErr != nil {
		log.Printf("ProcessTransfer: transaction failed for %s transfer: %v", req.TransferMode, txErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist transfer"})
		return
	}

	if TemporalClient != nil {
		temporalsetup.StartSettlementWorkflow(
			TemporalClient,
			transfer.ID.String(),
			transfer.FromAccount.String(),
			transfer.ToAccount.String(),
			transfer.Amount,
			transfer.TransferMode,
			req.Tpin,
		)
	} else {
		log.Printf("Warning: Temporal not connected — %s transfer %s persisted but no workflow started",
			req.TransferMode, transfer.ID)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":  fmt.Sprintf("%s transfer initiated", req.TransferMode),
		"transfer": transfer,
	})
}

func GetTransferStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transfer ID"})
		return
	}

	if TemporalClient != nil {
		workflowID := "settlement-" + id.String()
		resp, queryErr := TemporalClient.QueryWorkflow(c.Request.Context(), workflowID, "", "getStatus")
		if queryErr == nil {
			var state workflows.WorkflowState
			if decodeErr := resp.Get(&state); decodeErr == nil {
				c.JSON(http.StatusOK, state)
				return
			}
		}
	}

	transfer, err := db.Repo.FindByID(c.Request.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "transfer not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transfer)
}

func CancelTransfer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transfer ID"})
		return
	}

	if TemporalClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workflow engine not available"})
		return
	}

	workflowID := "settlement-" + id.String()
	handle, err := TemporalClient.UpdateWorkflow(c.Request.Context(),
		temporalclient.UpdateWorkflowOptions{
			WorkflowID:   workflowID,
			UpdateName:   "requestCancellation",
			WaitForStage: temporalclient.WorkflowUpdateStageCompleted,
		},
	)
	if err != nil {
		log.Printf("CancelTransfer: failed to send update to workflow %s: %v", workflowID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reach workflow"})
		return
	}

	if updateErr := handle.Get(c.Request.Context(), nil); updateErr != nil {
		c.JSON(http.StatusConflict, gin.H{"error": updateErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancellation confirmed"})
}

func GetAllTransfers(c *gin.Context) {
	transfers, err := db.Repo.FindAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transfers"})
		return
	}
	c.JSON(http.StatusOK, transfers)
}