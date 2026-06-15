package models

import "time"

type OutboxPayload struct {
	FromAccount  string  `bson:"from_account"  json:"from_account"`
	ToAccount    string  `bson:"to_account"    json:"to_account"`
	Amount       float64 `bson:"amount"        json:"amount"`
	TransferMode string  `bson:"transfer_mode" json:"transfer_mode"`
}

type OutboxEntry struct {
	ID            string        `bson:"_id"            json:"id"`
	TransferID    string        `bson:"transfer_id"    json:"transfer_id"`
	CorrelationID string        `bson:"correlation_id" json:"correlation_id"`
	Action        string        `bson:"action"         json:"action"`
	Status        string        `bson:"status"         json:"status"`
	Payload       OutboxPayload `bson:"payload"        json:"payload"`
	CreatedAt     time.Time     `bson:"created_at"     json:"created_at"`
	Attempts      int           `bson:"attempts"       json:"attempts"`
}
