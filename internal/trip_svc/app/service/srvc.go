package service

import (
	"context"
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

func (s *Service) Get(ctx context.Context, id int) (*models.Trip, error) {
	return nil, nil //TODO
}
