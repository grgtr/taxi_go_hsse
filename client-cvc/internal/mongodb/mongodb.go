// internal/mongodb/mongodb.go
package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Trip represents a trip in MongoDB.
type Trip struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	OfferID  string             `bson:"offer_id"`
	ClientID string             `bson:"client_id"`
	From     LatLngLiteral      `bson:"from"`
	To       LatLngLiteral      `bson:"to"`
	Price    Money              `bson:"price"`
	Status   string             `bson:"status"`
	// Add other fields as needed
}

// LatLngLiteral represents latitude and longitude in MongoDB.
type LatLngLiteral struct {
	Lat float64 `bson:"lat"`
	Lng float64 `bson:"lng"`
}

// Money represents an amount and currency in MongoDB.
type Money struct {
	Amount   float64 `bson:"amount"`
	Currency string  `bson:"currency"`
}

// Database represents a MongoDB database connection.
type Database struct {
	client *mongo.Client
	dbName string
}

// NewDatabase creates a new MongoDB database connection.
func NewDatabase(uri, dbName string) (*Database, error) {
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

	log.Println("Connected to MongoDB")

	return &Database{
		client: client,
		dbName: dbName,
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

// GetTrips retrieves a list of trips from MongoDB.
func (db *Database) GetTrips(user_id string) ([]Trip, error) {
	collection := db.client.Database(db.dbName).Collection("trips")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define filter to get trips for a specific client_id
	filter := bson.M{"client_id": user_id}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var trips []Trip
	if err := cursor.All(ctx, &trips); err != nil {
		return nil, err
	}

	return trips, nil
}

// CreateTrip inserts a new trip into MongoDB.
func (db *Database) CreateTrip(trip *Trip) (*mongo.InsertOneResult, error) {
	collection := db.client.Database(db.dbName).Collection("trips")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, trip)
	if err != nil {
		return &mongo.InsertOneResult{}, err
	}

	fmt.Printf("Inserted new trip with ID %v\n", result.InsertedID)
	return result, nil
}

// GetTripByID retrieves a trip by ID from MongoDB.
func (db *Database) GetTripByID(tripID string) (*Trip, error) {
	collection := db.client.Database(db.dbName).Collection("trips")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return nil, err
	}

	var trip Trip
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&trip)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

// CancelTrip cancels a trip by ID in MongoDB.
func (db *Database) CancelTrip(tripID string) error {
	collection := db.client.Database(db.dbName).Collection("trips")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return err
	}

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	fmt.Printf("Deleted %v document(s)\n", result.DeletedCount)
	return nil
}
