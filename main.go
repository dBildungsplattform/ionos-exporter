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

func main() {
	configPath := flag.String("config", "/etc/ionos-exporter/config.yaml", "Path to configuration file")
	envFile := flag.String("env", "", "Path to env file (optional)")
	flag.Parse()
	if *envFile != "" {
		if _, err := os.Stat(*configPath); os.IsNotExist(err) {
			log.Printf("Warning: config file not found at %s, continuing without it", *configPath)
		}
	}

	exporterPort = internal.GetEnv("IONOS_EXPORTER_APPLICATION_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(internal.GetEnv("IONOS_EXPORTER_API_CYCLE", "200"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	go internal.CollectResources(m, *envFile, ionos_api_cycle)
	if s3_enabled, err := strconv.ParseBool(internal.GetEnv("IONOS_EXPORTER_S3_ENABLED", "false")); s3_enabled == true {
		if err != nil {
			log.Fatal("Cannot convert IONOS_EXPORTER_S3_ENABLED value to bool")
		}
		go internal.S3CollectResources(m, ionos_api_cycle)
	}
	go internal.PostgresCollectResources(m, *configPath, *envFile, ionos_api_cycle)

	internal.PrintDCResources(m)
	internal.StartPrometheus(m)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))

}
