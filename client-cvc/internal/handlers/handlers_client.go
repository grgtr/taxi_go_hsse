// internal/handlers/handlers_client.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"net/http"
	"taxi/internal/models"
	"taxi/internal/mongodb"
	kfk "taxi/pkg/kafka"
	"time"
)

// Router returns a new ServeMux with registered handlers for the client service.
func Router(db *mongodb.Database) *chi.Mux {
	router := chi.NewRouter()

	// Register handlers with MongoDB connection
	router.Post("/trips", createTripHandler(db))
	router.Get("/trips", getTripsHandler(db))
	router.Get("/trips/{trip_id}", getTripByIDHandler(db))
	router.Post("/trip/{trip_id}/cancel", cancelTripHandler(db))

	return router
}

func getTripsHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user_id := r.Header.Get("user_id")
		trips, err := db.GetTrips(user_id) // Implement this function in your mongodb package
		if err != nil {
			http.Error(w, "Error getting trips from MongoDB", http.StatusInternalServerError)
			return
		}

		// Convert trips to JSON
		response, err := json.Marshal(trips)
		if err != nil {
			http.Error(w, "Error encoding trips to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

type TripRequest struct {
	OfferID string `json:"offer_id"`
}

type Order struct {
	From     Location `json:"from"`
	To       Location `json:"to"`
	ClientID string   `json:"client_id"`
	Price    Price    `json:"price"`
}
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Price struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

func createTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//var trip mongodb.Trip
		var trip TripRequest
		//user_id := r.Header.Get("user_id")
		err := json.NewDecoder(r.Body).Decode(&trip)
		fmt.Println(r)
		fmt.Println(r.Header.Get("user_id"))
		fmt.Println(r.Body)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error decoding JSON request", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resp, err := http.Get("http://offering:8099/offers/" + trip.OfferID)
		if err != nil {
			http.Error(w, "Error getting offer", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Error reading response body", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var order Order
		err = json.Unmarshal(bytes, &order)
		fmt.Println(order)
		if err != nil {
			http.Error(w, "Error decoding JSON request", http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()
		// Validate and insert trip into MongoDB
		offer := mongodb.Trip{
			OfferID:  trip.OfferID,
			ClientID: r.Header.Get("user_id"),
			From: mongodb.LatLngLiteral{
				Lat: order.From.Lat,
				Lng: order.From.Lng,
			},
			To: mongodb.LatLngLiteral{
				Lat: order.To.Lat,
				Lng: order.To.Lng,
			},
			Price: mongodb.Money{
				Amount:   order.Price.Amount,
				Currency: order.Price.Currency,
			},
			Status: "DRIVER_SEARCH",
		}
		//result.InsertedID.(primitive.ObjectID).Hex()
		result, err := db.CreateTrip(&offer) // Implement this function in your mongodb package
		//"data": {
		//	"offer_id": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6InN0cmluZyIsImZyb20iOnsibGF0IjowLCJsbmciOjB9LCJ0byI6eyJsYXQiOjAsImxuZyI6MH0sImNsaWVudF9pZCI6InN0cmluZyIsInByaWNlIjp7ImFtb3VudCI6OTkuOTUsImN1cnJlbmN5IjoiUlVCIn19.fg0Bv2ONjT4r8OgFqJ2tpv67ar7pUih2LhDRCRhWW3c"
		//}
		offerData := models.CommandCreateData{
			OfferId: trip.OfferID,
		}
		jsonData, err := json.Marshal(offerData)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		request := models.Request{
			Id:              result.InsertedID.(primitive.ObjectID).Hex(),
			Source:          "/client",
			Type:            "trip.command.create",
			DataContentType: "application/jsonapplication/json",
			Time:            time.Now(),
			Data:            jsonData,
		}
		toTrip, err := kfk.ConnectKafka(context.Background(), "kafka:9092", "driver-client-trip-topic", 0)
		jsonRequest, err := json.Marshal(request)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = kfk.SendToTopic(toTrip, jsonRequest)
		if err != nil {
			fmt.Println(err)
		}
		result.InsertedID.(primitive.ObjectID).Hex()

		if err != nil {
			http.Error(w, "Error creating trip in MongoDB", http.StatusInternalServerError)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//fmt.Println("Created trip in MongoDB")

		w.WriteHeader(http.StatusOK)
	}
}

func getTripByIDHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "trip_id")

		trip, err := db.GetTripByID(tripID) // Implement this function in your mongodb package
		if err != nil {
			http.Error(w, "Incorrect trip ID", http.StatusBadRequest)
			return
		}

		// Convert trip to JSON
		response, err := json.Marshal(trip)
		if err != nil {
			http.Error(w, "Trip not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

func cancelTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trip_id := chi.URLParam(r, "trip_id")
		reason := r.URL.Query().Get("reason")
		//user_id := r.Header.Get("user_id")
		offerData := models.CommandCancelData{
			TripId: trip_id,
			Reason: reason,
		}
		jsonData, err := json.Marshal(offerData)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		request := models.Request{
			Id:              trip_id,
			Source:          "/client",
			Type:            "trip.command.cancel",
			DataContentType: "application/jsonapplication/json",
			Time:            time.Now(),
			Data:            jsonData,
		}
		toTrip, err := kfk.ConnectKafka(context.Background(), "kafka:9092", "driver-client-trip-topic", 0)
		if err != nil {
			fmt.Println(err)
			return
		}
		jsonRequest, err := json.Marshal(request)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = kfk.SendToTopic(toTrip, jsonRequest)
		if err != nil {
			fmt.Println(err)
		}

		err = db.UpdateTripStatus(trip_id, "CANCELED")
		if err != nil {
			http.Error(w, "Error canceling trip in MongoDB", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
