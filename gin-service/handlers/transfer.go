package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"banking/gin-service/client"
	"banking/gin-service/db"
	"banking/gin-service/models"
	temporalsetup "banking/gin-service/temporal"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	ProcessTransferWithClient(c, client.NewCoreBankingClient())
}

func ProcessTransferWithClient(c *gin.Context, bankingClient client.BankingClient) {
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
		FromAccount:  fromID,
		ToAccount:    toID,
		Amount:       req.Amount,
		TransferMode: req.TransferMode,
		Status:       getInitialStatus(req.TransferMode),
	}

	ctx := c.Request.Context()
	if err := db.Repo.Create(ctx, &transfer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.TransferMode == "IMPS" {
		result, err := bankingClient.SettleTransfer(
			transfer.FromAccount.String(),
			transfer.ToAccount.String(),
			transfer.Amount,
			transfer.TransferMode,
			req.Tpin,
		)
		if err != nil {
			log.Printf("Failed to settle IMPS transfer: %v", err)
			db.Repo.UpdateStatus(context.Background(), transfer.ID, "FAILED")
			transfer.Status = "FAILED"
		} else {
			db.Repo.UpdateStatus(context.Background(), transfer.ID, result.Status)
			transfer.Status = result.Status
		}
		c.JSON(http.StatusAccepted, gin.H{
			"message":  fmt.Sprintf("%s transfer completed", req.TransferMode),
			"transfer": transfer,
		})
	} else {
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
			log.Printf("Warning: Temporal not connected — using goroutine fallback for %s transfer %s",
				req.TransferMode, transfer.ID)
			go simulateSettlement(transfer, bankingClient, req.Tpin)
		}
		c.JSON(http.StatusAccepted, gin.H{
			"message":  fmt.Sprintf("%s transfer initiated", req.TransferMode),
			"transfer": transfer,
		})
	}
}

func GetTransferStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transfer ID"})
		return
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

func GetAllTransfers(c *gin.Context) {
	transfers, err := db.Repo.FindAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transfers"})
		return
	}
	c.JSON(http.StatusOK, transfers)
}

func getInitialStatus(mode string) string {
	switch mode {
	case "IMPS":
		return "SUCCESS"
	case "NEFT":
		return "PENDING"
	case "RTGS":
		return "PROCESSING"
	default:
		return "PENDING"
	}
}

func simulateSettlement(transfer models.Transfer, bankingClient client.BankingClient, tpin string) {
	switch transfer.TransferMode {
	case "NEFT":
		time.Sleep(30 * time.Second)
	case "RTGS":
		time.Sleep(15 * time.Second)
	}

	result, err := bankingClient.SettleTransfer(
		transfer.FromAccount.String(),
		transfer.ToAccount.String(),
		transfer.Amount,
		transfer.TransferMode,
		tpin,
	)
	if err != nil {
		log.Printf("Failed to notify Spring Boot: %v", err)
		db.Repo.UpdateStatus(context.Background(), transfer.ID, "FAILED")
		log.Printf("%s transfer %s marked as FAILED", transfer.TransferMode, transfer.ID)
		return
	}

	db.Repo.UpdateStatus(context.Background(), transfer.ID, result.Status)
	log.Printf("%s transfer %s settled with status: %s", transfer.TransferMode, transfer.ID, result.Status)
}