package main

import (
	"log"
	"sync"
	"time"

	"github.com/musthaq16/vehicle-iot-simulator/internal/config"
	"github.com/musthaq16/vehicle-iot-simulator/internal/simulator"
)

type routeManager struct {
	mu           sync.Mutex
	activeRoutes map[string]struct{} // Tracks running routes by vehicle_id
	wg           sync.WaitGroup
	stopChan     chan struct{}
}

func main() {
	// Load initial config
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	manager := &routeManager{
		activeRoutes: make(map[string]struct{}),
		stopChan:     make(chan struct{}),
	}

	// Start initial routes
	manager.startRoutes(cfg)

	// Start config watcher
	go manager.watchConfigChanges("config.yaml")

	// Block until shutdown signal
	manager.wg.Wait()
}

func (rm *routeManager) startRoutes(cfg *config.AppConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, route := range cfg.Routes {
		// Check if route is already running
		if _, exists := rm.activeRoutes[route.VehicleID]; !exists {
			rm.activeRoutes[route.VehicleID] = struct{}{}
			rm.wg.Add(1)

			go func(r config.RouteConfig) {
				defer rm.wg.Done()
				defer func() {
					rm.mu.Lock()
					delete(rm.activeRoutes, r.VehicleID)
					rm.mu.Unlock()
				}()
				simulator.RunVehicleSimulator(cfg.OSRM.BaseUrl, r, cfg.Simulator.FrequencySeconds)
			}(route)
		}
	}
}

func (rm *routeManager) watchConfigChanges(configPath string) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				log.Printf("Error reloading config: %v", err)
				continue
			}
			rm.startRoutes(cfg)

		case <-rm.stopChan:
			return
		}
	}
}

// Add graceful shutdown logic as needed
