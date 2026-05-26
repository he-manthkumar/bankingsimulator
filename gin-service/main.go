package main

import (
	"log"

	"banking/gin-service/db"
	"banking/gin-service/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db.Connect()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "gin-service running"})
	})

	r.POST("/process-transfer", handlers.ProcessTransfer)
	r.GET("/transfer/:id", handlers.GetTransferStatus)
	r.GET("/transfers", handlers.GetAllTransfers)

	log.Println("Gin Transfer Service running on :8081")
	r.Run(":8081")
}
