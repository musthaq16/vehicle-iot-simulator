package osrm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Coordinate holds lat/lon
type Coordinate struct {
	Lat float64
	Lon float64
}

// Parse string like "12.9716,77.5946" into Coordinate
func ParseCoord(input string) (Coordinate, error) {
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		return Coordinate{}, fmt.Errorf("invalid coordinate: %s", input)
	}

	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return Coordinate{}, fmt.Errorf("invalid lat/lon: %s", input)
	}

	return Coordinate{Lat: lat, Lon: lon}, nil
}

// OSRM response format
type osrmResponse struct {
	Routes []struct {
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}

// GetRoute returns list of lat/lon from OSRM API
func GetRoute(baseURL string, source, target Coordinate) ([]Coordinate, error) {

	url := fmt.Sprintf("%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=full&geometries=geojson",
		baseURL, source.Lon, source.Lat, target.Lon, target.Lat)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSRM returned %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var parsed osrmResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %v", err)
	}

	var coords []Coordinate
	for _, pair := range parsed.Routes[0].Geometry.Coordinates {
		coords = append(coords, Coordinate{Lon: pair[0], Lat: pair[1]})
	}

	return coords, nil
}
