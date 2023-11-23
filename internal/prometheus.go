package internal

import (
	"io"
	"net/http"
	"sync"

	//"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type ionosCollector struct {
	mutex          *sync.RWMutex
	coresMetric    *prometheus.GaugeVec
	ramMetric      *prometheus.GaugeVec
	serverMetric   *prometheus.GaugeVec
	dcCoresMetric  *prometheus.GaugeVec
	dcRamMetric    *prometheus.GaugeVec
	dcServerMetric *prometheus.GaugeVec
	dcDCMetric     *prometheus.GaugeVec
}

var mutex *sync.RWMutex

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newIonosCollector(m *sync.RWMutex) *ionosCollector {
	mutex = m
	return &ionosCollector{
		mutex: m,
		coresMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dc_cores_amount",
			Help: "Shows the number of currently active cores in an IONOS datacenter",
		}, []string{"datacenter"}),
		ramMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dc_ram_gb",
			Help: "Shows the number of currently active RAM in an IONOS datacenter",
		}, []string{"datacenter"}),
		serverMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dc_server_amount",
			Help: "Shows the number of currently active servers in an IONOS datacenter",
		}, []string{"datacenter"}),
		dcCoresMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_cores_amount",
			Help: "Shows the number of currently active cores of an IONOS account",
		}, []string{"account"}),
		dcRamMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_ram_gb",
			Help: "Shows the number of currently active RAM of an IONOS account",
		}, []string{"account"}),
		dcServerMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_server_amount",
			Help: "Shows the number of currently active servers of an IONOS account",
		}, []string{"account"}),
		dcDCMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_datacenter_amount",
			Help: "Shows the number of datacenters of an IONOS account",
		}, []string{"account"}),
	}
}

// Each and every collector must implement the Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
func (collector *ionosCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	collector.coresMetric.Describe(ch)
	collector.ramMetric.Describe(ch)
	collector.serverMetric.Describe(ch)
	collector.dcCoresMetric.Describe(ch)
	collector.dcRamMetric.Describe(ch)
	collector.dcServerMetric.Describe(ch)
	collector.dcDCMetric.Describe(ch)
}

// Collect implements required collect function for all promehteus collectors
func (collector *ionosCollector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	// Reset metrics in case a datacenter was removed
	collector.coresMetric.Reset()
	collector.ramMetric.Reset()
	collector.serverMetric.Reset()
	for dcName, dcResources := range IonosDatacenters {
		//Write latest value for each metric in the prometheus metric channel.
		collector.coresMetric.WithLabelValues(dcName).Set(float64(dcResources.Cores))
		collector.ramMetric.WithLabelValues(dcName).Set(float64(dcResources.Ram / 1024)) // MB -> GB
		collector.serverMetric.WithLabelValues(dcName).Set(float64(dcResources.Servers))
	}
	collector.dcCoresMetric.WithLabelValues("SVS").Set(float64(CoresTotal))
	collector.dcRamMetric.WithLabelValues("SVS").Set(float64(RamTotal / 1024)) // MB -> GB
	collector.dcServerMetric.WithLabelValues("SVS").Set(float64(ServerTotal))
	collector.dcDCMetric.WithLabelValues("SVS").Set(float64(DataCenters))

	collector.coresMetric.Collect(ch)
	collector.ramMetric.Collect(ch)
	collector.serverMetric.Collect(ch)
	collector.dcCoresMetric.Collect(ch)
	collector.dcRamMetric.Collect(ch)
	collector.dcServerMetric.Collect(ch)
	collector.dcDCMetric.Collect(ch)
}
func (collector *ionosCollector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func StartPrometheus(m *sync.RWMutex) {
	ic := newIonosCollector(m)
	prometheus.MustRegister(ic)
	prometheus.MustRegister(httpRequestsTotal)

}

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "http_requests_total",
		Help:        "Total number of HTTP requests",
		ConstLabels: prometheus.Labels{"server": "api"},
	},
	[]string{"route", "method"},
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	PrintDCTotals(mutex)
	httpRequestsTotal.WithLabelValues("/healthcheck", r.Method).Inc()
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}
