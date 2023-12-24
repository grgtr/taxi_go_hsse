package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func NewGorm(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("Can't open connection to db")
	}
	return db
}
