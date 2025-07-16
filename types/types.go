package types

import "time"

// Add this struct to track previous point data
type vehicleState struct {
	lastLat  float64
	lastLon  float64
	lastTime time.Time
}

// Coordinate holds lat/lon
type Coordinate struct {
	Lat float64
	Lon float64
}
