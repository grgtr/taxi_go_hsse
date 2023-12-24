package trip

import (
	"context"
	"gorm.io/gorm"
	"taxi/pkg/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r Repository) Get(ctx context.Context, id int32) (*models.Trip, error) {
	var trip models.Trip
	r.db.First(&trip, id)
	return &trip, nil
}
