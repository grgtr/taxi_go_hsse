package service

import (
	"taxi/pkg/models"
)

type TripRepository interface { //TODO
	Get(id int) (models.Trip, error)
	Add(trip models.Trip) error
}
