package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Gateway represents the structure of each gateway in the gateways.json file
type Gateway struct {
	Name     string `json:"name"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Checks []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"checks"`
}

// GatewaysFile represents the JSON structure for gateways.json
type GatewaysFile struct {
	Gateways []Gateway `json:"gateways"`
}

// Prometheus metrics for link status and location
var (
	gatewayLinkStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_link_status",
			Help: "Shows link status: 1 for online, 0 for offline",
		},
		[]string{"gateway_name", "link_url"},
	)

	// Metric: gateway_location with latitude and longitude as labels
	gatewayLocation = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_location",
			Help: "Gateway location with latitude and longitude as labels",
		},
		[]string{"gateway_name", "latitude", "longitude"},
	)
)

// Fetch interval in minutes (configurable)
var fetchInterval time.Duration

// Initialize Prometheus metrics and set fetch interval
func init() {
	// Register the link status and location metrics
	prometheus.MustRegister(gatewayLinkStatus)
	prometheus.MustRegister(gatewayLocation)

	// Get fetch interval from environment variable (default 1 minute)
	interval, err := strconv.Atoi(os.Getenv("FETCH_INTERVAL"))
	if err != nil || interval <= 0 {
		fetchInterval = 1 * time.Minute // default value
	} else {
		fetchInterval = time.Duration(interval) * time.Minute
	}
}

// LoadGatewaysConfig loads the gateway configuration from the JSON file
func LoadGatewaysConfig(filePath string) (*GatewaysFile, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var gateways GatewaysFile
	if err := json.Unmarshal(data, &gateways); err != nil {
		return nil, err
	}

	return &gateways, nil
}

// FetchGatewayLinkStatus fetches the JSON data from a URL and checks if it's "online"
func FetchGatewayLinkStatus(url string) bool {
	log.Printf("Fetching data from URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch data from URL: %s, error: %v", url, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-200 response from URL: %s, status: %d", url, resp.StatusCode)
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from URL: %s, error: %v", url, err)
		return false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to parse JSON from URL: %s, error: %v", url, err)
		return false
	}

	// Assuming 'online' is the field in the JSON that indicates status
	if online, ok := result["online"].(bool); ok {
		return online
	}

	log.Printf("No 'online' status found in response from URL: %s", url)
	return false
}

// UpdateGatewayStatus updates Prometheus metrics for a gateway's link status and location
func UpdateGatewayStatus(gateway Gateway) {
	// Record latitude and longitude in Prometheus as labels in gateway_location metric
	gatewayLocation.With(prometheus.Labels{
		"gateway_name": gateway.Name,
		"latitude":     fmt.Sprintf("%f", gateway.Location.Latitude),
		"longitude":    fmt.Sprintf("%f", gateway.Location.Longitude),
	}).Set(1)

	log.Printf("Updated gateway_location metric for gateway %s with latitude %f and longitude %f", gateway.Name, gateway.Location.Latitude, gateway.Location.Longitude)

	// Check link status for each link in the gateway and update Prometheus
	for _, check := range gateway.Checks {
		var status float64
		if FetchGatewayLinkStatus(check.URL) {
			status = 1 // Link is online
		} else {
			status = 0 // Link is offline
		}

		// Update Prometheus metric for link status
		gatewayLinkStatus.With(prometheus.Labels{
			"gateway_name": gateway.Name,
			"link_url":     check.URL,
		}).Set(status)

		log.Printf("Updated Prometheus metrics for gateway %s, link %s, status: %f", gateway.Name, check.URL, status)
	}
}

// MonitorGateways runs periodically to update the gateway statuses
func MonitorGateways(gatewaysFile *GatewaysFile) {
	for {
		for _, gateway := range gatewaysFile.Gateways {
			UpdateGatewayStatus(gateway)
		}
		log.Printf("Sleeping for %v before next fetch...", fetchInterval)
		time.Sleep(fetchInterval)
	}
}

func main() {
	log.Println("Go-backend starting...")

	gatewaysFile, err := LoadGatewaysConfig("config/gateways.json")
	if err != nil {
		log.Fatalf("Failed to load gateways.json: %v", err)
	}

	// Start monitoring the gateways in the background
	go MonitorGateways(gatewaysFile)

	// Expose Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9100", nil)) // Serve metrics on port 9100
}
