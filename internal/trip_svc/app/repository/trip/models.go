package trip

type CreateCmd struct {
	OfferId string `json:"offer_id"`
}

type AcceptTrip struct {
	TripId   string `json:"trip_id"`
	DriverId string `json:"driver_id"`
}

type StartedTrip struct {
	TripId string `json:"trip_id"`
}

type EndedTrip struct {
	TripId string `json:"trip_id"`
}
