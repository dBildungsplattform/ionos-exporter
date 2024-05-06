package main

import (
	"ionos-exporter/internal"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	dcMutex         = &sync.RWMutex{} // Mutex to sync access to the Datacenter map
	s3Mutex         = &sync.RWMutex{}
	exporterPort    string // Port to be used for exposing the metrics
	ionos_api_cycle int32  // Cycle time in seconds to query the IONOS API for changes, not th ePrometheus scraping intervall
)

func main() {
	//internal.CollectResources(mutex, ionos_api_cycle)
	//internal.BasicAuthExample()
	exporterPort = internal.GetEnv("IONOS_EXPORTER_APPLICATION_CONTAINER_PORT", "9100")
	if cycletime, err := strconv.ParseInt(internal.GetEnv("IONOS_EXPORTER_API_CYCLE", "300"), 10, 32); err != nil {
		log.Fatal("Cannot convert IONOS_API_CYCLE to int")
	} else {
		ionos_api_cycle = int32(cycletime)
	}
	// internal.IPCollectResources()
	go internal.CollectResources(dcMutex, ionos_api_cycle)
	go internal.S3CollectResources(s3Mutex, ionos_api_cycle)

	//internal.PrintDCResources(mutex)
	internal.StartPrometheus(dcMutex)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/healthcheck", http.HandlerFunc(internal.HealthCheck))
	log.Fatal(http.ListenAndServe(":"+exporterPort, nil))

}
