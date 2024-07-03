package internal

import (
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
	mutex            *sync.RWMutex
	coresMetric      *prometheus.GaugeVec
	ramMetric        *prometheus.GaugeVec
	serverMetric     *prometheus.GaugeVec
	dcCoresMetric    *prometheus.GaugeVec
	dcRamMetric      *prometheus.GaugeVec
	dcServerMetric   *prometheus.GaugeVec
	dcDCMetric       *prometheus.GaugeVec
	nlbsMetric       *prometheus.GaugeVec
	albsMetric       *prometheus.GaugeVec
	natsMetric       *prometheus.GaugeVec
	dcDCNLBMetric    *prometheus.GaugeVec
	dcDCALBMetric    *prometheus.GaugeVec
	dcDCNATMetric    *prometheus.GaugeVec
	dcNLBRulesMetric *prometheus.GaugeVec
	dcALBRulesMetric *prometheus.GaugeVec
	dcTotalIpsMetric prometheus.Gauge
}

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func NewIonosCollector(m *sync.RWMutex) *ionosCollector {
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
		dcTotalIpsMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ionos_total_number_of_ips",
			Help: "Shows the number of Ips in a IONOS",
		}),
	}
}

// Each and every collector must implement the Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
// func (collector *ionosCollector) Describe(ch chan<- *prometheus.Desc) {
func (collector *ionosCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	collector.coresMetric.Describe(ch)
	collector.ramMetric.Describe(ch)
	collector.serverMetric.Describe(ch)
	collector.dcCoresMetric.Describe(ch)
	collector.dcRamMetric.Describe(ch)
	collector.dcServerMetric.Describe(ch)
	collector.dcDCMetric.Describe(ch)
	collector.nlbsMetric.Describe(ch)
	collector.albsMetric.Describe(ch)
	collector.natsMetric.Describe(ch)
	collector.dcDCNLBMetric.Describe(ch)
	collector.dcDCALBMetric.Describe(ch)
	collector.dcDCNATMetric.Describe(ch)
	collector.dcALBRulesMetric.Describe(ch)
	collector.dcNLBRulesMetric.Describe(ch)
	collector.dcTotalIpsMetric.Describe(ch)
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
	collector.albsMetric.Reset()
	collector.natsMetric.Reset()
	collector.nlbsMetric.Reset()
	// fmt.Println("Here are the metrics in ionosCollector", IonosDatacenters)
	for dcName, dcResources := range IonosDatacenters {
		//Write latest value for each metric in the prometheus metric channel.
		collector.coresMetric.WithLabelValues(dcName).Set(float64(dcResources.Cores))
		collector.ramMetric.WithLabelValues(dcName).Set(float64(dcResources.Ram / 1024)) // MB -> GB
		collector.serverMetric.WithLabelValues(dcName).Set(float64(dcResources.Servers))
		collector.nlbsMetric.WithLabelValues(dcName, dcResources.NLBName, dcResources.NLBRuleName).Set(float64(dcResources.NLBs))
		collector.albsMetric.WithLabelValues(dcName, dcResources.ALBName, dcResources.ALBRuleName).Set(float64(dcResources.ALBs))
		collector.natsMetric.WithLabelValues(dcName).Set(float64(dcResources.NATs))
		collector.dcTotalIpsMetric.Set(float64(dcResources.TotalIPs))

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
	collector.nlbsMetric.Collect(ch)
	collector.albsMetric.Collect(ch)
	collector.natsMetric.Collect(ch)
	collector.dcDCNLBMetric.Collect(ch)
	collector.dcDCALBMetric.Collect(ch)
	collector.dcDCNATMetric.Collect(ch)
	collector.dcNLBRulesMetric.Collect(ch)
	collector.dcALBRulesMetric.Collect(ch)
	collector.dcTotalIpsMetric.Collect(ch)
}
