package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/musthaq16/vehicle-iot-simulator/types"
)

// Ensure folder exists
func ensureStateDir() string {
	dir := "./state"
	os.MkdirAll(dir, os.ModePerm)
	return dir
}

func GetVehicleState(vehicleID string) (*types.VehicleProgress, error) {
	path := filepath.Join(ensureStateDir(), vehicleID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state types.VehicleProgress
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveVehicleState(vehicleID string, state *types.VehicleProgress) error {
	path := filepath.Join(ensureStateDir(), vehicleID+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
