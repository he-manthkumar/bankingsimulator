package db

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"banking/gin-service/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("transfer not found")

type TransferRepository interface {
	Create(ctx context.Context, t *models.Transfer) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Transfer, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	FindAll(ctx context.Context) ([]models.Transfer, error)
}

var Repo TransferRepository

type mongoTransfer struct {
	ID            string    `bson:"_id"`
	CorrelationID string    `bson:"correlation_id"`
	FromAccount   string    `bson:"from_account"`
	ToAccount     string    `bson:"to_account"`
	Amount        float64   `bson:"amount"`
	TransferMode  string    `bson:"transfer_mode"`
	Status        string    `bson:"status"`
	CreatedAt     time.Time `bson:"created_at"`
}

func toMongo(t *models.Transfer) mongoTransfer {
	return mongoTransfer{
		ID:            t.ID.String(),
		CorrelationID: t.CorrelationID,
		FromAccount:   t.FromAccount.String(),
		ToAccount:     t.ToAccount.String(),
		Amount:        t.Amount,
		TransferMode:  t.TransferMode,
		Status:        t.Status,
		CreatedAt:     t.CreatedAt,
	}
}

func fromMongo(m mongoTransfer) models.Transfer {
	return models.Transfer{
		ID:            uuid.MustParse(m.ID),
		CorrelationID: m.CorrelationID,
		FromAccount:   uuid.MustParse(m.FromAccount),
		ToAccount:     uuid.MustParse(m.ToAccount),
		Amount:        m.Amount,
		TransferMode:  m.TransferMode,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
	}
}

type MongoTransferRepo struct {
	Col *mongo.Collection
}

func (r *MongoTransferRepo) Create(ctx context.Context, t *models.Transfer) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := r.Col.InsertOne(ctx, toMongo(t))
	return err
}

func (r *MongoTransferRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Transfer, error) {
	var m mongoTransfer
	err := r.Col.FindOne(ctx, bson.M{"_id": id.String()}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t := fromMongo(m)
	return &t, nil
}

func (r *MongoTransferRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.Col.UpdateOne(ctx,
		bson.M{"_id": id.String()},
		bson.M{"$set": bson.M{"status": status}},
	)
	return err
}

func (r *MongoTransferRepo) FindAll(ctx context.Context) ([]models.Transfer, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.Col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var docs []mongoTransfer
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	transfers := make([]models.Transfer, len(docs))
	for i, d := range docs {
		transfers[i] = fromMongo(d)
	}
	return transfers, nil
}

func Connect() {
	uri := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGO_DB", "banking")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	col := client.Database(dbName).Collection("transfers")
	Repo = &MongoTransferRepo{Col: col}
	log.Printf("Connected to MongoDB at %s (db=%s)", uri, dbName)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}