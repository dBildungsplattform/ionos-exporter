package main

import (
	"ionos-exporter/internal"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	mutex           = &sync.RWMutex{} // Mutex to sync access to the Daatcenter map
	exporterPort    string            // Port to be used for exposing the metrics
	ionos_api_cycle int32             // Cycle time in seconds to query the IONOS API for changes, not th ePrometheus scraping intervall
)

func main() {
	exporterPort = internal.GetEnv("IONOS_EXPORTER_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(internal.GetEnv("IONOS_EXPORTER_API_CYCLE", "900"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	go internal.CollectResources(mutex, ionos_api_cycle)
	internal.StartPrometheus(mutex)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Print(os.Environ())
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))
}
