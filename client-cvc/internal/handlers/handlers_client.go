// internal/handlers/handlers_client.go
package handlers

import (
	"encoding/json"
	"net/http"
	"taxi/client-cvc/internal/mongodb"

	"github.com/go-chi/chi/v5"
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
		trips, err := db.GetTrips() // Implement this function in your mongodb package
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

func createTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var trip mongodb.Trip
		err := json.NewDecoder(r.Body).Decode(&trip)
		if err != nil {
			http.Error(w, "Error decoding JSON request", http.StatusBadRequest)
			return
		}

		// Validate and insert trip into MongoDB
		err = db.CreateTrip(&trip) // Implement this function in your mongodb package
		if err != nil {
			http.Error(w, "Error creating trip in MongoDB", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func getTripByIDHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "trip_id")

		trip, err := db.GetTripByID(tripID) // Implement this function in your mongodb package
		if err != nil {
			http.Error(w, "Error getting trip from MongoDB", http.StatusInternalServerError)
			return
		}

		// Convert trip to JSON
		response, err := json.Marshal(trip)
		if err != nil {
			http.Error(w, "Error encoding trip to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

func cancelTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "trip_id")

		// Your cancel trip logic using MongoDB
		err := db.CancelTrip(tripID) // Implement this function in your mongodb package
		if err != nil {
			http.Error(w, "Error canceling trip in MongoDB", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
