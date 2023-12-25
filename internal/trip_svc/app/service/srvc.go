package service

import (
	"net/http"
	"taxi/internal/trip_svc/app/repository/trip"
	"taxi/pkg/models"
)

type Service struct {
	tripRepo TripRepository
}

func NewService(tripRepo TripRepository) *Service {
	return &Service{
		tripRepo: tripRepo,
	}
}

func (s *Service) Get(id int) (*models.Trip, error) {
	return s.tripRepo.Get(id)
}

func (s *Service) CreateTrip(event trip.Event) error {
	resp, err := http.Get("http://localhost:8089/offers/")
	return s.tripRepo.Add(trip)
}
