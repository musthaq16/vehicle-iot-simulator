package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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

// WithSignalHandler creates a context that cancels on OS signals
func WithSignalHandler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived termination signal, saving state...")
		cancel()
	}()

	return ctx
}
