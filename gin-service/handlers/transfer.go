package handlers

import (
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

// TemporalClient is set by main.go after connecting to the Temporal server.
// If nil (e.g. during unit tests or local dev without Temporal), the handler
// falls back to the goroutine-based simulateSettlement.
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
		ID:           uuid.New(),
		FromAccount:  fromID,
		ToAccount:    toID,
		Amount:       req.Amount,
		TransferMode: req.TransferMode,
		Status:       getInitialStatus(req.TransferMode),
	}

	if result := db.DB.Create(&transfer); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
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
			db.DB.Model(&transfer).Update("status", "FAILED")
			transfer.Status = "FAILED"
		} else {
			db.DB.Model(&transfer).Update("status", result.Status)
			transfer.Status = result.Status
		}
		c.JSON(http.StatusAccepted, gin.H{
			"message":  fmt.Sprintf("%s transfer completed", req.TransferMode),
			"transfer": transfer,
		})
	} else {
		// NEFT / RTGS — use Temporal for durable, crash-safe settlement
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
			// Fallback: goroutine-based settlement (used in tests and local dev
			// when Temporal server is not running)
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

	var transfer models.Transfer
	if result := db.DB.First(&transfer, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transfer not found"})
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func GetAllTransfers(c *gin.Context) {
	var transfers []models.Transfer
	if result := db.DB.Order("created_at desc").Find(&transfers); result.Error != nil {
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

// simulateSettlement is the goroutine-based fallback used when Temporal is unavailable.
// Kept for backward compatibility with tests and local dev without a Temporal server.
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
		db.DB.Model(&transfer).Update("status", "FAILED")
		log.Printf("%s transfer %s marked as FAILED", transfer.TransferMode, transfer.ID)
		return
	}

	db.DB.Model(&transfer).Update("status", result.Status)
	log.Printf("%s transfer %s settled with status: %s", transfer.TransferMode, transfer.ID, result.Status)
}