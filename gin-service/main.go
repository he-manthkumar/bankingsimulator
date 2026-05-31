package main

import (
	"log"
	"os"

	"banking/gin-service/db"
	"banking/gin-service/handlers"
	temporalsetup "banking/gin-service/temporal"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	temporalclient "go.temporal.io/sdk/client"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db.Connect()

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	tc, err := temporalclient.Dial(temporalclient.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Printf("Warning: could not connect to Temporal at %s (%v) — falling back to goroutine settlement",
			temporalHost, err)
	} else {
		defer tc.Close()

		handlers.TemporalClient = tc

		go temporalsetup.StartWorker(tc)
		log.Println("Temporal client connected:", temporalHost)
	}

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
