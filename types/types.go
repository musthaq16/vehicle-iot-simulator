package types

// Add this struct to track previous point data
type VehicleState struct {
	LastLat      float64
	LastLon      float64
	CurrentSpeed float64 // km/h
	Odometer     float64 // meters (cumulative distance)
}

// Coordinate holds lat/lon
type Coordinate struct {
	Lat float64
	Lon float64
}

type VehicleProgress struct {
	LastLat   float64 `json:"last_lat"`
	LastLon   float64 `json:"last_lon"`
	Odometer  float64 `json:"odometer"`
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	LastIndex int     `json:"last_index"`
}
