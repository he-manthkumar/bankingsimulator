package db

import (
	"context"
	"time"

	"banking/gin-service/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type OutboxRepository interface {

	CreateWithSession(ctx context.Context, session mongo.Session, entry *models.OutboxEntry) error
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string) error
	DeleteByTransferID(ctx context.Context, transferID string) error
	FindUnprocessed(ctx context.Context) ([]models.OutboxEntry, error)
}

var OutboxRepo OutboxRepository

type MongoOutboxRepo struct {
	Col *mongo.Collection
}

func (r *MongoOutboxRepo) CreateWithSession(ctx context.Context, session mongo.Session, entry *models.OutboxEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	_, err := r.Col.InsertOne(mongo.NewSessionContext(ctx, session), entry)
	return err
}

func (r *MongoOutboxRepo) MarkProcessed(ctx context.Context, id string) error {
	_, err := r.Col.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": "PROCESSED"}},
	)
	return err
}

func (r *MongoOutboxRepo) MarkFailed(ctx context.Context, id string) error {
	_, err := r.Col.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{"status": "FAILED"},
			"$inc": bson.M{"attempts": 1},
		},
	)
	return err
}

func (r *MongoOutboxRepo) DeleteByTransferID(ctx context.Context, transferID string) error {
	_, err := r.Col.DeleteOne(ctx, bson.M{"transfer_id": transferID})
	return err
}

func (r *MongoOutboxRepo) FindUnprocessed(ctx context.Context) ([]models.OutboxEntry, error) {
	cursor, err := r.Col.Find(ctx, bson.M{"status": "UNPROCESSED"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var entries []models.OutboxEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
