package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "text/template"
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

// Prometheus metrics
var (
    gatewayOnlineStatus = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gateway_online_status",
            Help: "Shows whether the gateway is online: 1 for online, 0 for offline",
        },
        []string{"name", "latitude", "longitude"},
    )
)

// Initialize Prometheus metrics
func init() {
    prometheus.MustRegister(gatewayOnlineStatus)
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

// FetchAndParseGatewayStatus fetches the JSON data from the URL and parses the 'online' status
func FetchAndParseGatewayStatus(gateway Gateway) bool {
    for _, check := range gateway.Checks {
        log.Printf("Fetching data for %s from URL: %s", gateway.Name, check.URL)
        
        resp, err := http.Get(check.URL)
        if err != nil {
            log.Printf("Failed to fetch data from URL: %s, error: %v", check.URL, err)
            return false
        }
        defer resp.Body.Close()

        log.Printf("Successfully fetched data from URL: %s", check.URL)

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            log.Printf("Failed to read response body from URL: %s, error: %v", check.URL, err)
            return false
        }

        log.Printf("Parsing JSON data for %s...", gateway.Name)
        var result map[string]interface{}
        if err := json.Unmarshal(body, &result); err != nil {
            log.Printf("Failed to parse JSON from URL: %s, error: %v", check.URL, err)
            return false
        }

        if val, ok := result["online"]; ok {
            if online, ok := val.(bool); ok {
                log.Printf("Gateway %s online status: %v", gateway.Name, online)
                return online
            }
        } else if val, ok := result[gateway.Name]; ok {
            if onlineStatus, ok := val.(map[string]interface{})["online"]; ok {
                if online, ok := onlineStatus.(bool); ok {
                    log.Printf("Gateway %s online status: %v", gateway.Name, online)
                    return online
                }
            }
        }

        log.Printf("No 'online' status found for %s in the fetched data", gateway.Name)
    }

    return false
}

// UpdateGatewayStatus updates the Prometheus metrics with the gateway's online status
func UpdateGatewayStatus(gateway Gateway) {
    online := FetchAndParseGatewayStatus(gateway)

    gatewayOnlineStatus.With(prometheus.Labels{
        "name":      gateway.Name,
        "latitude":  fmt.Sprintf("%f", gateway.Location.Latitude),
        "longitude": fmt.Sprintf("%f", gateway.Location.Longitude),
    }).Set(boolToFloat64(online))

    log.Printf("Updated Prometheus metrics for gateway %s, online status: %v", gateway.Name, online)
}

// Convert bool to float64 for Prometheus Gauge
func boolToFloat64(value bool) float64 {
    if value {
        return 1.0
    }
    return 0.0
}

// MonitorGateways runs periodically to update the gateway statuses
func MonitorGateways(gatewaysFile *GatewaysFile) {
    for {
        for _, gateway := range gatewaysFile.Gateways {
            UpdateGatewayStatus(gateway)
        }
        time.Sleep(1 * time.Minute)
    }
}

// CreateDashboardFile creates the dashboard JSON for each gateway
func CreateDashboardFile(gateway Gateway) error {
    const dashboardTemplate = `
{
  "id": null,
  "title": "Gateway Monitor Dashboard",
  "tags": [],
  "timezone": "browser",
  "schemaVersion": 16,
  "version": 1,
  "panels": [
    {
      "type": "stat",
      "title": "Gateway Uptime (Last 7 Days)",
      "targets": [
        {
          "expr": "rate(up{job='gateway'}[1w])",
          "legendFormat": "{{.Name}} uptime",
          "refId": "A"
        }
      ],
      "options": {
        "reduceOptions": {
          "calcs": ["mean"],
          "fields": "",
          "values": false
        },
        "orientation": "auto",
        "textMode": "value"
      },
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "thresholds": {
            "mode": "percentage",
            "steps": [
              {"color": "red", "value": 0},
              {"color": "green", "value": 95}
            ]
          }
        }
      },
      "gridPos": {
        "h": 6,
        "w": 8,
        "x": 0,
        "y": 0
      }
    },
    {
      "type": "status-history",
      "title": "Gateway Status History",
      "targets": [
        {
          "expr": "up{job='gateway', instance='{{.Name}}'}",
          "refId": "A"
        }
      ],
      "options": {
        "reduceOptions": {
          "calcs": ["lastNotNull"],
          "fields": "",
          "values": false
        },
        "orientation": "auto",
        "showValue": "auto"
      },
      "fieldConfig": {
        "defaults": {
          "thresholds": {
            "steps": [
              {"color": "red", "value": 0},
              {"color": "orange", "value": 1},
              {"color": "green", "value": 2}
            ]
          }
        }
      },
      "datasource": "Prometheus",
      "gridPos": {
        "h": 6,
        "w": 8,
        "x": 8,
        "y": 0
      }
    },
    {
      "type": "geomap",
      "title": "Gateway Geomap",
      "targets": [
        {
          "expr": "up{job='gateway', instance='{{.Name}}'}",
          "refId": "A"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "thresholds": {
            "steps": [
              {"color": "red", "value": 0},
              {"color": "orange", "value": 1},
              {"color": "green", "value": 2}
            ]
          }
        }
      },
      "options": {
        "layers": [
          {
            "type": "point",
            "url": "",
            "label": "Gateways",
            "source": {
              "type": "table"
            },
            "colorField": "value"
          }
        ],
        "mapView": {
          "lat": {{.Location.Latitude}},
          "lon": {{.Location.Longitude}},
          "zoom": 6
        }
      },
      "gridPos": {
        "h": 10,
        "w": 16,
        "x": 0,
        "y": 6
      }
    }
  ]
}
`
    log.Printf("Creating dashboard for %s", gateway.Name)

    filePath := fmt.Sprintf("/var/lib/grafana/dashboards/%s-dashboard.json", gateway.Name)
    file, err := os.Create(filePath)
    if err != nil {
        return fmt.Errorf("failed to create dashboard file: %v", err)
    }
    defer file.Close()

    tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
    if err != nil {
        return fmt.Errorf("failed to parse template: %v", err)
    }

    if err := tmpl.Execute(file, gateway); err != nil {
        return fmt.Errorf("failed to execute template: %v", err)
    }

    log.Printf("Dashboard created for %s", gateway.Name)
    return nil
}

func main() {
    log.Println("Go-backend starting...")

    gatewaysFile, err := LoadGatewaysConfig("config/gateways.json")
    if err != nil {
        log.Fatalf("Failed to load gateways.json: %v", err)
    }

    // Generate dashboards for each gateway
    for _, gateway := range gatewaysFile.Gateways {
        if err := CreateDashboardFile(gateway); err != nil {
            log.Fatalf("Error creating dashboard for %s: %v", gateway.Name, err)
        }
    }

    // Start monitoring the gateways in the background
    go MonitorGateways(gatewaysFile)

    // Expose Prometheus metrics
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":9100", nil)) // Serve metrics on port 9100
		
}
