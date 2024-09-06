package main

import (
    "log"
    "os"
    "text/template"
)

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
      "title": "Gateway Uptime",
      "targets": [
        {
          "expr": "up",
          "legendFormat": "instance",
          "refId": "A"
        }
      ],
      "datasource": "Prometheus"
    }
  ]
}
`


func createDashboardFile() error {
    log.Println("Starting to create dashboard file...")
		file, err := os.Create("/var/lib/grafana/dashboards/gateway-dashboard.json")
    if err != nil {
        log.Fatalf("Failed to create dashboard file: %v", err)
        return err
    }
    defer file.Close()

    tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
    if err != nil {
        log.Fatalf("Failed to parse template: %v", err)
        return err
    }

    log.Println("Executing template...")
    if err := tmpl.Execute(file, nil); err != nil {
        log.Fatalf("Failed to execute template: %v", err)
    }

    log.Println("Dashboard file created successfully.")
    return nil
}

func main() {
    log.Println("Go-backend starting...")
    err := createDashboardFile()
    if err != nil {
        log.Fatalf("Error: %v", err)
    }

    log.Println("Go-backend finished.")
}
