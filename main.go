package main

import (
	"flag"
	"ionos-exporter/internal"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
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
		err := godotenv.Load(*envFile)
		if err != nil {
			log.Fatalf("Error loading specified env file: %v\n", err)
		}
	}

	exporterPort = internal.GetEnv("IONOS_EXPORTER_APPLICATION_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(internal.GetEnv("IONOS_EXPORTER_API_CYCLE", "200"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	go internal.CollectResources(m, ionos_api_cycle)

	// Contract Limits Exporter
	if internal.Must(internal.GetBoolEnv("IONOS_EXPORTER_CONTRACT_LIMITS_ENABLED", true)) {
		contractLimitsCollector := internal.NewContractLimitsCollector()
		go contractLimitsCollector.StartScrape(ionos_api_cycle)
		prometheus.MustRegister(contractLimitsCollector)
	}

	if internal.Must(internal.GetBoolEnv("IONOS_EXPORTER_S3_ENABLED", false)) {
		go internal.S3CollectResources(m, ionos_api_cycle)
	}

	if internal.Must(internal.GetBoolEnv("IONOS_EXPORTER_POSTGRES_ENABLED", false)) {
		go internal.PostgresCollectResources(m, *configPath, ionos_api_cycle)
	}

	internal.PrintDCResources(m)
	internal.StartPrometheus(m)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))

}
