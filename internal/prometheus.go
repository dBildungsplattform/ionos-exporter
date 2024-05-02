package internal

import (
	"io"
	"net/http"
	"os"
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

type lbCollector struct {
	mutex            *sync.RWMutex
	nlbsMetric       *prometheus.GaugeVec
	albsMetric       *prometheus.GaugeVec
	natsMetric       *prometheus.GaugeVec
	dcDCNLBMetric    *prometheus.GaugeVec
	dcDCALBMetric    *prometheus.GaugeVec
	dcDCNATMetric    *prometheus.GaugeVec
	dcNLBRulesMetric *prometheus.GaugeVec
	dcALBRulesMetric *prometheus.GaugeVec
}

type s3Collector struct {
	mutex                            *sync.RWMutex
	s3TotalGetMethodSizeMetric       *prometheus.GaugeVec
	s3TotalPutMethodSizeMetric       *prometheus.GaugeVec
	s3TotalNumberOfGetRequestsMetric *prometheus.GaugeVec
	s3TotalNumberOfPutRequestsMetric *prometheus.GaugeVec
}

var mutex *sync.RWMutex

func newLBCollector(m *sync.RWMutex) *lbCollector {
	mutex = m
	return &lbCollector{
		mutex: &sync.RWMutex{},
		nlbsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_networkloadbalancer_amount",
			Help: "Shows the number of active Network Loadbalancers in an IONOS datacenter",
		}, []string{"datacenter", "nlb_name", "nlb_rules_name"}),
		albsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_applicationloadbalancer_amount",
			Help: "Shows the number of active Application Loadbalancers in an IONOS datacenter",
		}, []string{"datacenter", "alb_name", "alb_rules_name"}),
		natsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_nat_gateways_amount",
			Help: "Shows the number of NAT Gateways in an IONOS datacenter",
		}, []string{"datacenter"}),
		dcDCNLBMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_networkloadbalancer_amount",
			Help: "Shows the total number of Network Loadbalancers in IONOS Account",
		}, []string{"account"}),
		dcDCALBMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_applicationbalancer_amount",
			Help: "Shows the total number of Application Loadbalancers in IONOS Account",
		}, []string{"account"}),
		dcDCNATMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_nat_gateways_amount",
			Help: "Shows the total number of NAT Gateways in IONOS Account",
		}, []string{"account"}),
		dcNLBRulesMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_number_of_nlb_rules",
			Help: "Shows the total number of NLB Rules in IONOS Account",
		}, []string{"nlb_rules"}),
		dcALBRulesMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_total_nmumber_of_alb_rules",
			Help: "Shows the total number of ALB Rules in IONOS Account",
		}, []string{"alb_rules"}),
	}
}

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newIonosCollector(m *sync.RWMutex) *ionosCollector {
	mutex = m
	return &ionosCollector{
		mutex: &sync.RWMutex{},
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

func newS3Collector(m *sync.RWMutex) *s3Collector {
	mutex = m
	return &s3Collector{
		mutex: &sync.RWMutex{},
		s3TotalGetMethodSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_size_of_get_requests_in_bytes",
			Help: "Gives the total size of s3 GET HTTP Request in Bytes",
		}, []string{"bucket_name"}),
		s3TotalPutMethodSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_size_of_put_requests_in_bytes",
			Help: "Gives the total size of s3 PUT HTTP Request in Bytes",
		}, []string{"bucket_name"}),
		s3TotalNumberOfGetRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_get_requests",
			Help: "Gives the total number of S3 GET HTTP Requests",
		}, []string{"bucket_name"}),
		s3TotalNumberOfPutRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_put_requests",
			Help: "Gives the total number of S3 PUT HTTP Requests",
		}, []string{"bucket_name"}),
	}
}

func (collector *lbCollector) Describe(ch chan<- *prometheus.Desc) {

	collector.nlbsMetric.Describe(ch)
	collector.albsMetric.Describe(ch)
	collector.natsMetric.Describe(ch)
	collector.dcDCNLBMetric.Describe(ch)
	collector.dcDCALBMetric.Describe(ch)
	collector.dcDCNATMetric.Describe(ch)
	collector.dcALBRulesMetric.Describe(ch)
	collector.dcNLBRulesMetric.Describe(ch)

}

func (collector *lbCollector) Collect(ch chan<- prometheus.Metric) {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	collector.albsMetric.Reset()
	collector.natsMetric.Reset()
	collector.nlbsMetric.Reset()
	for lbName, lbResources := range IonosLoadbalancers {
		collector.nlbsMetric.WithLabelValues(lbName, lbResources.NLBName, lbResources.NLBRuleName).Set(float64(lbResources.NLBs))
		collector.albsMetric.WithLabelValues(lbName, lbResources.ALBName, lbResources.ALBRuleName).Set(float64(lbResources.ALBs))
		collector.natsMetric.WithLabelValues(lbName).Set(float64(lbResources.NATs))
	}

	collector.nlbsMetric.Collect(ch)
	collector.albsMetric.Collect(ch)
	collector.natsMetric.Collect(ch)
	collector.dcDCNLBMetric.Collect(ch)
	collector.dcDCALBMetric.Collect(ch)
	collector.dcDCNATMetric.Collect(ch)
	collector.dcNLBRulesMetric.Collect(ch)
	collector.dcALBRulesMetric.Collect(ch)
}

func (collector *s3Collector) Describe(ch chan<- *prometheus.Desc) {
	collector.s3TotalGetMethodSizeMetric.Describe(ch)
	collector.s3TotalPutMethodSizeMetric.Describe(ch)
	collector.s3TotalNumberOfGetRequestsMetric.Describe(ch)
	collector.s3TotalNumberOfPutRequestsMetric.Describe(ch)

}
func (collector *s3Collector) Collect(ch chan<- prometheus.Metric) {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	collector.s3TotalGetMethodSizeMetric.Reset()
	collector.s3TotalPutMethodSizeMetric.Reset()
	collector.s3TotalNumberOfGetRequestsMetric.Reset()
	collector.s3TotalNumberOfPutRequestsMetric.Reset()

	for s3Name, s3Resources := range IonosS3Buckets {
		collector.s3TotalGetMethodSizeMetric.WithLabelValues(s3Name).Set(float64(s3Resources.TotalGetMethodSize))
		collector.s3TotalPutMethodSizeMetric.WithLabelValues(s3Name).Set(float64(s3Resources.TotalPutMethodSize))
		collector.s3TotalNumberOfGetRequestsMetric.WithLabelValues(s3Name).Set(float64(s3Resources.GetMethods))
		collector.s3TotalNumberOfPutRequestsMetric.WithLabelValues(s3Name).Set(float64(s3Resources.PutMethods))

	}

	collector.s3TotalGetMethodSizeMetric.Collect(ch)
	collector.s3TotalPutMethodSizeMetric.Collect(ch)
	collector.s3TotalNumberOfGetRequestsMetric.Collect(ch)
	collector.s3TotalNumberOfPutRequestsMetric.Collect(ch)

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
	account := os.Getenv("IONOS_ACCOUNT")
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

	collector.dcCoresMetric.WithLabelValues(account).Set(float64(CoresTotal))
	collector.dcRamMetric.WithLabelValues(account).Set(float64(RamTotal / 1024)) // MB -> GB
	collector.dcServerMetric.WithLabelValues(account).Set(float64(ServerTotal))
	collector.dcDCMetric.WithLabelValues(account).Set(float64(DataCenters))

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

func (collector *s3Collector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func (collector *lbCollector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func StartPrometheus(m *sync.RWMutex) {
	ic := newIonosCollector(m)
	s3c := newS3Collector(m)
	lbc := newLBCollector(m)
	prometheus.MustRegister(ic)
	prometheus.MustRegister(s3c)
	prometheus.MustRegister(lbc)
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
