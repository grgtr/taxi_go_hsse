package models

import (
	"gorm.io/gorm"
)

type price struct {
	currency string
	amount   int
}

type location struct {
	lat float64
	lng float64
}

type Trip struct {
	gorm.Model
	tripId  string
	offerId string
	cost    price
	status  string
	from    location
	to      location
}
