package main

import (
	"ionos-exporter/internal"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	dcMutex         = &sync.RWMutex{} // Mutex to sync access to the Datacenter map
	s3Mutex         = &sync.RWMutex{}
	pgMutex         = &sync.RWMutex{}
	exporterPort    string // Port to be used for exposing the metrics
	ionos_api_cycle int32  // Cycle time in seconds to query the IONOS API for changes, not th ePrometheus scraping intervall
)

func main() {
	exporterPort = internal.GetEnv("IONOS_EXPORTER_APPLICATION_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(internal.GetEnv("IONOS_EXPORTER_API_CYCLE", "200"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	go internal.CollectResources(dcMutex, ionos_api_cycle)
	go internal.S3CollectResources(s3Mutex, ionos_api_cycle)
	go internal.PostgresCollectResources(pgMutex, ionos_api_cycle)
	startPrometheus()
	//internal.PrintDCResources(mutex)
	// internal.StartPrometheus(dcMutex)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))

}

func startPrometheus() {
	// Initialize and register collectors
	ionosCollector := internal.NewIonosCollector(dcMutex)
	s3Collector := internal.NewS3Collector(s3Mutex)
	pgCollector := internal.NewPostgresCollector(pgMutex)

	prometheus.MustRegister(ionosCollector)
	prometheus.MustRegister(s3Collector)
	prometheus.MustRegister(pgCollector)
	prometheus.MustRegister(internal.HttpRequestsTotal)
}
