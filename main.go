package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	/* // Get current time
	now := time.Now().UTC()

	// Convert to epoch milliseconds
	epochMillis := now.UnixNano() / int64(time.Millisecond)

	// Convert to big-endian binary
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint64(epochMillis)); err != nil {
		fmt.Println("Error writing binary:", err)
		return
	}

	// Convert binary to hex string
	hexStr := hex.EncodeToString(buf.Bytes())

	// Output
	fmt.Println("Current Time        :", now.Format(time.RFC3339Nano))
	fmt.Println("Epoch (ms)          :", epochMillis)
	fmt.Println("Big-endian Hex      :", hexStr)

	return */

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

// startRoutes starts all configured routes as goroutines
func (rm *routeManager) startRoutes(cfg *config.AppConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	// Do not defer cancel here; cancel when signal received or routes complete

	// Channel to catch interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to signal completion
	done := make(chan struct{})

	// Goroutine to handle shutdown
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, cancelling simulations...", sig)
		cancel() // Cancel the context to stop all simulations

		// Wait for goroutines with timeout
		waitChan := make(chan struct{})
		go func() {
			rm.wg.Wait()
			close(waitChan)
		}()

		select {
		case <-waitChan:
			log.Println("All routes stopped, states saved")
		case <-time.After(5 * time.Second):
			log.Println("Timeout waiting for routes to stop, forcing exit")
		}
		close(done)
	}()

	for _, route := range cfg.Routes {
		if _, exists := rm.activeRoutes[route.VehicleID]; !exists {
			rm.activeRoutes[route.VehicleID] = struct{}{}
			rm.wg.Add(1)

			go func(r config.RouteConfig) {
				defer rm.wg.Done()
				defer func() {
					rm.mu.Lock()
					fmt.Println("removedddd", rm.activeRoutes, r.VehicleID)
					delete(rm.activeRoutes, r.VehicleID)
					rm.mu.Unlock()
				}()
				simulator.RunVehicleSimulator(ctx, cfg.OSRM.BaseUrl, r, cfg.Simulator.FrequencySeconds, cfg.Simulator.Client)
			}(route)
		}
	}

	// Wait for either all routes to complete or a signal
	log.Println("Started all routes, waiting for completion or signal...")
	select {
	case <-done:
		log.Println("Exiting due to signal or completion")
	case <-ctx.Done():
		log.Println("Context cancelled externally, waiting for routes to stop...")
		rm.wg.Wait()
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
