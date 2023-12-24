// internal/handlers/handlers_client.go
package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"taxi/client-cvc/internal/mongodb"
)

// Router returns a new ServeMux with registered handlers for the client service.
func Router(db *mongodb.Database) *chi.Mux {
	router := chi.NewRouter()
	router.Post("/trips", createTripHandler(db))
	router.Get("/trips", getTripsHandler(db))
	router.Get("/trips/{trip_id}", getTripByIDHandler(db))
	router.Post("/trip/{trip_id}/cancel", cancelTripHandler(db))
	return router
}

func getTripsHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("getTripsHandler")
		w.Write([]byte("getTripsHandler"))
		w.WriteHeader(http.StatusOK)
	}
}

func createTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("createTripHandler")
		w.Write([]byte("createTripHandler"))
		w.WriteHeader(http.StatusOK)
		// Your implementation for handling POST /trips
	}
}

func getTripByIDHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("getTripByIDHandler")
		w.Write([]byte("getTripByIDHandler"))
		w.WriteHeader(http.StatusOK)
		// Your implementation for handling GET /trips/{trip_id}
	}
}

func cancelTripHandler(db *mongodb.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("cancelTripHandler")
		w.Write([]byte("cancelTripHandler"))
		w.WriteHeader(http.StatusOK)
		// Your implementation for handling POST /trip/{trip_id}/cancel
	}
}
