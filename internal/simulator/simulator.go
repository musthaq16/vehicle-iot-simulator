package simulator

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"github.com/musthaq16/vehicle-iot-simulator/internal/config"
	"github.com/musthaq16/vehicle-iot-simulator/internal/osrm"
	"github.com/musthaq16/vehicle-iot-simulator/types"
)

// RunVehicleSimulator simulates a single vehicle sending coordinates
func RunVehicleSimulator(baseURL string, routeCfg config.RouteConfig, intervalSec int, address string) {

	// 1. Establish TCP connection
	conn, err := net.Dial("tcp", address) // Replace with your server details
	if err != nil {
		log.Printf("[%s] TCP connection failed: %v", routeCfg.VehicleID, err)
		return
	}
	defer conn.Close()

	// 2. Prepare and send login packet with IMEI
	loginPacket, err := createLoginPacket(routeCfg.Imei)
	if err != nil {
		log.Printf("[%s] Login packet creation failed: %v", routeCfg.VehicleID, err)
		return
	}

	_, err = conn.Write(loginPacket)
	if err != nil {
		log.Printf("[%s] Login packet send failed: %v", routeCfg.VehicleID, err)
		return
	}

	log.Printf("[%s] Login packet sent: %X", routeCfg.VehicleID, loginPacket)

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

	state := types.VehicleState{
		CurrentSpeed: 0.0,
	}

	log.Printf("[%s] Starting route with %d points", routeCfg.VehicleID, len(points))
	fmt.Println()
	// 4. Process route points with position packets
	PacketTemplate := "000000000000009E8E0100000190DA491CD8003DE3607E00C846B4000F00F50C000B0000001F000F00EF0100F001001505004501007100001E00001F4000205600251400272700326500352702F70400F60000FC00000C00B5000E00B60008004235A70018000B00430000004400000024065200280E09002A00FA002B0000003127C700333506000400F10000CD1900C7000001AF00100020E9DC000C000354A6000000000100001CDB"

	// Simulate GPS emission every intervalSec seconds
	for i, pt := range points {
		stopHit := false

		// Check if current point is a stop location
		for idx, stop := range routeCfg.Stops {

			stopLatLon, err := osrm.ParseCoord(stop.Location)
			if err != nil {
				continue
			}

			dist := haversineDistance(pt.Lat, pt.Lon, stopLatLon.Lat, stopLatLon.Lon)
			if dist == 0 { // 50 meters threshold
				log.Printf("[%s] Stop hit at (%.6f, %.6f), pausing %d seconds",
					routeCfg.VehicleID, pt.Lat, pt.Lon, stop.Duration)

				for wait := 0; wait < stop.Duration; wait += intervalSec {
					packetHex, err := generatePacket(PacketTemplate, pt.Lat, pt.Lon, 0.0, state.Odometer)
					if err != nil {
						log.Printf("[%s] Stop packet generation failed: %v", routeCfg.VehicleID, err)
						break
					}

					packetBytes, err := hex.DecodeString(packetHex)
					if err != nil {
						log.Printf("[%s] Hex decode failed during stop: %v", routeCfg.VehicleID, err)
						break
					}

					if _, err := conn.Write(packetBytes); err != nil {
						log.Printf("[%s] Stop packet send failed: %v", routeCfg.VehicleID, err)
						break
					}

					time.Sleep(time.Duration(intervalSec) * time.Second)
				}
				// Remove the stop from slice so it's not checked again
				routeCfg.Stops = append(routeCfg.Stops[:idx], routeCfg.Stops[idx+1:]...)

				stopHit = true
				break
			}
		}

		// If we hit a stop and already sent packets, skip normal processing
		if stopHit {
			continue
		}

		// Calculate realistic speed
		if i > 0 {
			// Calculate distance in meters
			distance := haversineDistance(state.LastLat, state.LastLon, pt.Lat, pt.Lon) * 1000

			// Calculate speed in km/h (distance / time * 3.6)
			requiredSpeed := (distance / float64(intervalSec)) * 3.6
			fmt.Println("req speed", requiredSpeed)

			state.Odometer += distance

			// Generate random speed between 0 and requiredSpeed
			// randomSpeed := rand.Float64() * requiredSpeed

			// state.CurrentSpeed = randomSpeed + 1
			state.CurrentSpeed = requiredSpeed

			log.Printf("[%s] Speed: %.1f km/h (Distance: %.2f m, Interval: %ds)",
				routeCfg.VehicleID, state.CurrentSpeed, distance, intervalSec)
		}

		// Update state
		state.LastLat = pt.Lat
		state.LastLon = pt.Lon

		log.Printf("[%s] Point %d: %.6f, %.6f", routeCfg.VehicleID, i+1, pt.Lat, pt.Lon)
		fmt.Println()
		// Generate position packet with current timestamp and coordinates
		positionPacketHex, err := generatePacket(PacketTemplate, pt.Lat, pt.Lon, state.CurrentSpeed, state.Odometer)
		if err != nil {
			log.Printf("[%s] Position packet generation failed: %v", routeCfg.VehicleID, err)
			continue
		}

		// Convert hex string to bytes
		positionPacketBytes, err := hex.DecodeString(positionPacketHex)
		if err != nil {
			log.Printf("[%s] Hex decode failed: %v", routeCfg.VehicleID, err)
			continue
		}

		// Send position packet
		if _, err := conn.Write(positionPacketBytes); err != nil {
			log.Printf("[%s] Position packet send failed: %v", routeCfg.VehicleID, err)
			continue
		}
		// fmt.Println("the packet is", string(positionPacketHex))
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}

	log.Printf("[%s] Route completed", routeCfg.VehicleID)
}

// createLoginPacket generates the hex login packet with IMEI
func createLoginPacket(imei string) ([]byte, error) {
	// Validate IMEI length (typical IMEI is 15 digits)
	if len(imei) != 15 {
		return nil, fmt.Errorf("IMEI must be 15 digits")
	}

	// Convert IMEI string to hex ASCII representation
	imeiBytes := []byte(imei)

	// Create packet: 000F (2 bytes) + IMEI (15 bytes)
	packet := make([]byte, 2+15)

	// Set packet length prefix (000F in hex)
	packet[0] = 0x00
	packet[1] = 0x0F

	// Copy IMEI ASCII values
	copy(packet[2:], imeiBytes)

	return packet, nil
}
func generatePacket(template string, lat, lon, speed, odometer float64) (string, error) {
	// Decode the template packet
	packet, err := hex.DecodeString(template)
	if err != nil {
		return "", fmt.Errorf("failed to decode packet template: %v", err)
	}

	// Update timestamp (bytes 10-17)
	now := time.Now().UTC()
	binary.BigEndian.PutUint64(packet[10:18], uint64(now.UnixNano()/int64(time.Millisecond)))

	// Convert coordinates to int32 (scaled by 1e6 for precision)
	latInt := int32(lat * 1e7)
	lonInt := int32(lon * 1e7)

	// Print coordinate hex values
	latBytes := make([]byte, 4)
	lonBytes := make([]byte, 4)

	binary.BigEndian.PutUint32(lonBytes, uint32(lonInt))
	binary.BigEndian.PutUint32(latBytes, uint32(latInt))

	fmt.Printf("Latitude: %.6f → %d → %X\n", lat, latInt, latBytes)
	fmt.Printf("Longitude: %.6f → %d → %X\n", lon, lonInt, lonBytes)
	speedInt := uint16(speed) // Max 30.0 km/h
	speedBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(speedBytes, speedInt)
	fmt.Printf("speed: %d → %X\n", speedInt, speedBytes)

	odometerInt := uint32(odometer)
	odometerBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(odometerBytes, odometerInt)
	fmt.Printf("Odometer: %.2f m → %d → %X (4 bytes)\n", odometer, odometerInt, odometerBytes)

	// Inject coordinates
	binary.BigEndian.PutUint32(packet[19:23], uint32(lonInt)) // Longitude
	binary.BigEndian.PutUint32(packet[23:27], uint32(latInt)) // Latitude
	binary.BigEndian.PutUint16(packet[32:34], speedInt)       // Speed
	binary.BigEndian.PutUint32(packet[151:155], odometerInt)  // Odometer

	// Calculate CRC-16 (bytes 8 to len-4)
	crc := crc16IBM(packet[8 : len(packet)-4])

	// Convert CRC to 4-byte big-endian format
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, uint32(crc))

	// Replace last 4 bytes with CRC
	copy(packet[len(packet)-4:], crcBytes)

	fmt.Println("hex packet: ", hex.EncodeToString(packet))

	// Convert back to hex string
	return hex.EncodeToString(packet), nil
}

// Correct CRC-16/IBM implementation matching your examples
func crc16IBM(data []byte) uint16 {
	var crc uint16 = 0x0000      // Initial value
	polynomial := uint16(0xA001) // Reversed polynomial (0x8005 >> 1)

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// Haversine distance calculation (in kilometers)
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// 10:18
//000000000000009e8e0100000190da491cd8003de3607e00c846b4000f00f50c000b0000001f000f00ef0100f001001505004501007100001e00001f4000205600251400272700326500352702f70400f60000fc00000c00b5000e00b60008004235a70018000b00430000004400000024065200280e09002a00fa002b0000003127c700333506000400f10000cd1900c7000001af00100020e9dc000c000354a6000000000100001cdb //correct
//000000000000009e8e010000000001980d91c946e3607e00c846b4000f00f50c000b0000001f000f00ef0100f001001505004501007100001e00001f4000205600251400272700326500352702f70400f60000fc00000c00b5000e00b60008004235a70018000b00430000004400000024065200280e09002a00fa002b0000003127c700333506000400f10000cd1900c7000001af00100020e9dc000c000354a6000000000100001cdb // generate wrong
//000000000000009E8E01000001980D91C946003DE3607E00C846B4000F00F50C000B0000001F000F00EF0100F001001505004501007100001E00001F4000205600251400272700326500352702F70400F60000FC00000C00B5000E00B60008004235A70018000B00430000004400000024065200280E09002A00FA002B0000003127C700333506000400F10000CD1900C7000001AF00100020E9DC000C000354A6000000000100001CDB // original ah vara vendiyathu
