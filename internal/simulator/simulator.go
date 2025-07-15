package simulator

import (
	"fmt"
	"log"
	"time"

	"github.com/musthaq16/vehicle-iot-simulator/internal/config"
	"github.com/musthaq16/vehicle-iot-simulator/internal/osrm"
)

// RunVehicleSimulator simulates a single vehicle sending coordinates
func RunVehicleSimulator(baseURL string, routeCfg config.RouteConfig, intervalSec int) {
	src, err := osrm.ParseCoord(routeCfg.Source)
	if err != nil {
		log.Printf("[%s] Invalid source: %v", routeCfg.VehicleID, err)
		return
	}
	dst, err := osrm.ParseCoord(routeCfg.Target)
	if err != nil {
		log.Printf("[%s] Invalid target: %v", routeCfg.VehicleID, err)
		return
	}

	points, err := osrm.GetRoute(baseURL, src, dst)
	if err != nil {
		log.Printf("[%s] Route fetch failed: %v", routeCfg.VehicleID, err)
		return
	}

	log.Printf("[%s] Starting route with %d points", routeCfg.VehicleID, len(points))
	fmt.Println()

	// Simulate GPS emission every intervalSec seconds
	for i, pt := range points {
		log.Printf("[%s] Point %d: %.6f, %.6f", routeCfg.VehicleID, i+1, pt.Lat, pt.Lon)
		fmt.Println()
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}

	log.Printf("[%s] Route completed", routeCfg.VehicleID)
}
