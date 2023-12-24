package mongodb

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database represents a MongoDB database connection.
type Database struct {
	client *mongo.Client
}

// NewDatabase creates a new MongoDB database connection.
func NewDatabase(uri string) (*Database, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &Database{
		client: client,
	}, nil
}

// Close closes the MongoDB database connection.
func (db *Database) Close() {
	if db.client != nil {
		if err := db.client.Disconnect(context.Background()); err != nil {
			log.Println("Error disconnecting from MongoDB:", err)
		}
	}
}
