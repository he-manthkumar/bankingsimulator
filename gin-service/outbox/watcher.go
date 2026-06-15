package outbox

import (
	"context"
	"log"
	"time"

	"banking/gin-service/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	temporalclient "go.temporal.io/sdk/client"
)

func StartOutboxWatcher(mongoClient *mongo.Client, tc temporalclient.Client, dbName string) {
	col := mongoClient.Database(dbName).Collection("outbox")
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "operationType", Value: "insert"},
			{Key: "fullDocument.status", Value: "UNPROCESSED"},
		}}},
	}

	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	for {
		if err := watch(col, pipeline, opts, tc); err != nil {
			log.Printf("Outbox watcher: stream error (%v) — reconnecting in 5s", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func watch(
	col *mongo.Collection,
	pipeline mongo.Pipeline,
	opts *options.ChangeStreamOptions,
	tc temporalclient.Client,
) error {
	ctx := context.Background()
	stream, err := col.Watch(ctx, pipeline, opts)
	if err != nil {
		return err
	}
	defer stream.Close(ctx)

	log.Println("Outbox watcher: change stream open, watching for UNPROCESSED entries")

	for stream.Next(ctx) {
		var event struct {
			FullDocument models.OutboxEntry `bson:"fullDocument"`
		}
		if err := stream.Decode(&event); err != nil {
			log.Printf("Outbox watcher: failed to decode event: %v", err)
			continue
		}

		entry := event.FullDocument
		if entry.TransferID == "" {
			log.Printf("Outbox watcher: skipping entry with empty TransferID")
			continue
		}

		go signalWorkflow(tc, entry)
	}

	return stream.Err()
}

func signalWorkflow(tc temporalclient.Client, entry models.OutboxEntry) {
	workflowID := "settlement-" + entry.TransferID

	err := tc.SignalWorkflow(
		context.Background(),
		workflowID,
		"", 
		"outbox-ready",
		nil,
	)
	if err != nil {
		log.Printf("Outbox watcher: failed to signal workflow %s: %v", workflowID, err)
		return
	}

	log.Printf("Outbox watcher: signalled workflow %s (transfer %s)", workflowID, entry.TransferID)
}
