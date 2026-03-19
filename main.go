package main

import (
	"flag"
	"ionos-exporter/internal"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	m               = &sync.RWMutex{} // Mutex to sync access to the Datacenter map
	exporterPort    string            // Port to be used for exposing the metrics
	ionos_api_cycle int32             // Cycle time in seconds to query the IONOS API for changes, not th ePrometheus scraping intervall
)

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func main() {
	configPath := flag.String("config", "/etc/ionos-exporter/config.yaml", "Path to configuration file")
	envFile := flag.String("env", "", "Path to env file (optional)")
	flag.Parse()
	if *envFile != "" {
		if _, err := os.Stat(*configPath); os.IsNotExist(err) {
			log.Printf("Warning: config file not found at %s, continuing without it", *configPath)
		}
	}

	exporterPort = getEnvOrDefault("IONOS_EXPORTER_APPLICATION_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(getEnvOrDefault("IONOS_EXPORTER_API_CYCLE", "200"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	go internal.CollectResources(m, *envFile, ionos_api_cycle)

	// Enable / Disable S3 Exporter
	s3EnabledStr := getEnvOrDefault("IONOS_EXPORTER_S3_ENABLED", "false")
	s3Enabled, err := strconv.ParseBool(s3EnabledStr)
	if err != nil {
		log.Fatalf("Invalid value for IONOS_EXPORTER_S3_ENABLED=%q (expected true/false): %v", s3EnabledStr, err)
	}
	if s3Enabled {
		go internal.S3CollectResources(m, ionos_api_cycle)
	}

	// Enable / Disable Postgres Exporter
	postgresEnabledStr := getEnvOrDefault("IONOS_EXPORTER_POSTGRES_ENABLED", "false")
	postgresEnabled, err := strconv.ParseBool(postgresEnabledStr)
	if err != nil {
		log.Fatalf("Invalid value for IONOS_EXPORTER_POSTGRES_ENABLED=%q (expected true/false): %v", postgresEnabledStr, err)
	}
	if postgresEnabled {
		go internal.PostgresCollectResources(m, *configPath, *envFile, ionos_api_cycle)
	}

	internal.PrintDCResources(m)
	internal.StartPrometheus(m)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))

}
