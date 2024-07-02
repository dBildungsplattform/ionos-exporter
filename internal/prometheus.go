package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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

type postgresCollector struct {
	mutex                               *sync.RWMutex
	postgresTotalRamMetric              *prometheus.GaugeVec
	postgresTotalCPUMetric              *prometheus.GaugeVec
	postgresTotalStorageMetric          *prometheus.GaugeVec
	postgresTransactionRateMetric       *prometheus.GaugeVec
	postgresTotalStorageBytesMetric     *prometheus.GaugeVec
	postgresAvailableStorageBytesMetric *prometheus.GaugeVec
	postgresDiskIOMetric                *prometheus.GaugeVec
	postgresCpuRateMetric               *prometheus.GaugeVec
	postgresLoadMetric                  *prometheus.GaugeVec
	postgresTotalMemoryAvailableBytes   *prometheus.GaugeVec
}

type s3Collector struct {
	mutex                             *sync.RWMutex
	s3TotalGetRequestSizeMetric       *prometheus.GaugeVec
	s3TotalGetResponseSizeMetric      *prometheus.GaugeVec
	s3TotalPutRequestSizeMetric       *prometheus.GaugeVec
	s3TotalPutResponseSizeMetric      *prometheus.GaugeVec
	s3TotalPostRequestSizeMetric      *prometheus.GaugeVec
	s3TotalPostResponseSizeMetric     *prometheus.GaugeVec
	s3TotalHeadRequestSizeMetric      *prometheus.GaugeVec
	s3TotalHeadResponseSizeMetric     *prometheus.GaugeVec
	s3TotalNumberOfGetRequestsMetric  *prometheus.GaugeVec
	s3TotalNumberOfPutRequestsMetric  *prometheus.GaugeVec
	s3TotalNumberOfPostRequestsMetric *prometheus.GaugeVec
	s3TotalNumberOfHeadRequestsMetric *prometheus.GaugeVec
}

// var mutex *sync.RWMutex

// You must create a constructor for you collector that
// initializes every descriptor and returns a pointer to the collector
func newIonosCollector(m *sync.RWMutex) *ionosCollector {
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

// s3collector func returns all the metrics as gauges
func newS3Collector(m *sync.RWMutex) *s3Collector {
	return &s3Collector{
		mutex: m,
		s3TotalGetRequestSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_get_request_size_in_bytes",
			Help: "Gives the total size of s3 GET Request in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalGetResponseSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_get_response_size_in_bytes",
			Help: "Gives the total size of s3 GET Response in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalPutRequestSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_put_request_size_in_bytes",
			Help: "Gives the total size of s3 PUT Request in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalPutResponseSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_put_response_size_in_bytes",
			Help: "Gives the total size of s3 PUT Response in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalPostRequestSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_post_request_size_in_bytes",
			Help: "Gives the total size of s3 POST Request in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalPostResponseSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_post_response_size_in_bytes",
			Help: "Gives the total size of s3 POST Response in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalHeadRequestSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_head_request_size_in_bytes",
			Help: "Gives the total size of s3 HEAD Request in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalHeadResponseSizeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_head_response_size_in_bytes",
			Help: "Gives the total size of s3 HEAD Response in Bytes in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalNumberOfGetRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_get_requests",
			Help: "Gives the total number of S3 GET HTTP Requests in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalNumberOfPutRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_put_requests",
			Help: "Gives the total number of S3 PUT HTTP Requests in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalNumberOfPostRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_post_requests",
			Help: "Gives the total number of S3 Post Requests in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
		s3TotalNumberOfHeadRequestsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "s3_total_number_of_head_requests",
			Help: "Gives the total number of S3 HEAD HTTP Requests in one Bucket",
		}, []string{"bucket", "method", "region", "owner", "enviroment", "namespace", "tenant"}),
	}
}

func newPostgresCollector(m *sync.RWMutex) *postgresCollector {
	return &postgresCollector{
		mutex: m,
		postgresTotalRamMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_total_ram_in_cluster",
			Help: "Gives the total ammount of allocated RAM in cluster",
		}, []string{"clusterName", "owner", "db"}),
		postgresTotalCPUMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_total_cpu_in_cluster",
			Help: "Gives a total amount of CPU Cores in Cluster",
		}, []string{"clusterName", "owner", "db"}),
		postgresTotalStorageMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_total_storage_in_cluster",
			Help: "Gives a total amount of Storage in Cluster",
		}, []string{"clusterName", "owner", "db"}),
		postgresTransactionRateMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_transactions:rate2m",
			Help: "Gives a Transaction Rate in postgres cluster in 2m",
		}, []string{"clusterName"}),
		postgresTotalStorageBytesMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_total_storage_metric",
			Help: "Gives a Total Storage Metric in Bytes",
		}, []string{"clusterName"}),
		postgresAvailableStorageBytesMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_available_storage_metric",
			Help: "Gives a Available Storage Metric in Bytes",
		}, []string{"clusterName"}),
		postgresCpuRateMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgress_cpu_rate5m",
			Help: "Gives a CPU Rate (Average Utilization) over the past 5 Minutes",
		}, []string{"clusterName"}),
		postgresDiskIOMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_disk_io_time_weighted_seconds_rate5m",
			Help: "The rate of disk I/O time, in seconds, over a five-minute period.",
		}, []string{"clusterName"}),
		postgresLoadMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_load5",
			Help: "Linux load average for the last 5 minutes.",
		}, []string{"clusterName"}),
		postgresTotalMemoryAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_dbaas_postgres_memory_available_bytes",
			Help: "Available memory in bytes",
		}, []string{"clusterName"}),
	}
}

func (collector *postgresCollector) Describe(ch chan<- *prometheus.Desc) {
	collector.postgresTotalCPUMetric.Describe(ch)
	collector.postgresTotalRamMetric.Describe(ch)
	collector.postgresTotalStorageMetric.Describe(ch)
	collector.postgresTransactionRateMetric.Describe(ch)
	collector.postgresTotalStorageBytesMetric.Describe(ch)
	collector.postgresAvailableStorageBytesMetric.Describe(ch)
	collector.postgresCpuRateMetric.Describe(ch)
	collector.postgresDiskIOMetric.Describe(ch)
	collector.postgresLoadMetric.Describe(ch)
	collector.postgresTotalMemoryAvailableBytes.Describe(ch)
}

func (collector *postgresCollector) Collect(ch chan<- prometheus.Metric) {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	metricsMutex.Lock()
	collector.postgresTotalCPUMetric.Reset()
	collector.postgresTotalRamMetric.Reset()
	collector.postgresTotalStorageMetric.Reset()
	metricsMutex.Unlock()

	for postgresName, postgresResources := range IonosPostgresClusters {

		for _, telemetry := range postgresResources.Telemetry {
			for _, value := range telemetry.Values {
				if len(value) != 2 {
					fmt.Printf("Unexpected value length: %v\n", value)
					continue
				}
				metricValue, ok := value[1].(float64)
				if !ok {
					strValue, ok := value[1].(string)
					if !ok {
						fmt.Printf("Unexpected type for metric %s value: %v\n", telemetry.Values, value[1])
						continue
					}

					var err error
					metricValue, err = strconv.ParseFloat(strValue, 64)
					if err != nil {
						fmt.Printf("Failed to parse metric value: %v\n", err)
						continue
					}
				}
				// fmt.Println("Telemetry Metric", telemetry.Metric)
				switch telemetry.Metric["__name__"] {
				case "ionos_dbaas_postgres_transactions:rate2m":
					collector.postgresTransactionRateMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_storage_total_bytes":
					collector.postgresTotalStorageBytesMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_storage_available_bytes":
					collector.postgresAvailableStorageBytesMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_cpu_rate5m":
					collector.postgresCpuRateMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_disk_io_time_weighted_seconds_rate5m":
					collector.postgresDiskIOMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_load5":
					collector.postgresLoadMetric.WithLabelValues(postgresName).Set(float64(metricValue))
				case "ionos_dbaas_postgres_memory_available_bytes":
					collector.postgresTotalMemoryAvailableBytes.WithLabelValues(postgresName).Set(float64(metricValue))
				default:
					// fmt.Printf("Unrecognised metric: %s\n", telemetry.Metric["__name__"])
					continue
				}
			}
		}

		for _, dbName := range postgresResources.DatabaseNames {

			collector.postgresTotalCPUMetric.WithLabelValues(postgresName, postgresResources.Owner, dbName).Set(float64(postgresResources.CPU))
			collector.postgresTotalRamMetric.WithLabelValues(postgresName, postgresResources.Owner, dbName).Set(float64(postgresResources.RAM))
			collector.postgresTotalStorageMetric.WithLabelValues(postgresName, postgresResources.Owner, dbName).Set(float64(postgresResources.Storage))
		}

	}
	collector.postgresTotalCPUMetric.Collect(ch)
	collector.postgresTotalRamMetric.Collect(ch)
	collector.postgresTotalStorageMetric.Collect(ch)
	collector.postgresTransactionRateMetric.Collect(ch)
	collector.postgresTotalStorageBytesMetric.Collect(ch)
	collector.postgresAvailableStorageBytesMetric.Collect(ch)
	collector.postgresCpuRateMetric.Collect(ch)
	collector.postgresDiskIOMetric.Collect(ch)
	collector.postgresLoadMetric.Collect(ch)
	collector.postgresTotalMemoryAvailableBytes.Collect(ch)
}
func (collector *s3Collector) Describe(ch chan<- *prometheus.Desc) {
	collector.s3TotalGetRequestSizeMetric.Describe(ch)
	collector.s3TotalGetResponseSizeMetric.Describe(ch)
	collector.s3TotalPutRequestSizeMetric.Describe(ch)
	collector.s3TotalPutResponseSizeMetric.Describe(ch)
	collector.s3TotalPostRequestSizeMetric.Describe(ch)
	collector.s3TotalPostResponseSizeMetric.Describe(ch)
	collector.s3TotalHeadRequestSizeMetric.Describe(ch)
	collector.s3TotalHeadResponseSizeMetric.Describe(ch)
	collector.s3TotalNumberOfGetRequestsMetric.Describe(ch)
	collector.s3TotalNumberOfPutRequestsMetric.Describe(ch)
	collector.s3TotalNumberOfPostRequestsMetric.Describe(ch)
	collector.s3TotalNumberOfHeadRequestsMetric.Describe(ch)

}

func (collector *s3Collector) Collect(ch chan<- prometheus.Metric) {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	metricsMutex.Lock()
	collector.s3TotalGetRequestSizeMetric.Reset()
	collector.s3TotalGetResponseSizeMetric.Reset()
	collector.s3TotalPutRequestSizeMetric.Reset()
	collector.s3TotalPutResponseSizeMetric.Reset()
	collector.s3TotalPostRequestSizeMetric.Reset()
	collector.s3TotalPostResponseSizeMetric.Reset()
	collector.s3TotalHeadRequestSizeMetric.Reset()
	collector.s3TotalHeadResponseSizeMetric.Reset()
	collector.s3TotalNumberOfGetRequestsMetric.Reset()
	collector.s3TotalNumberOfPutRequestsMetric.Reset()
	collector.s3TotalNumberOfPostRequestsMetric.Reset()
	collector.s3TotalNumberOfHeadRequestsMetric.Reset()

	defer metricsMutex.Unlock()

	for s3Name, s3Resources := range IonosS3Buckets {

		region := s3Resources.Regions
		owner := s3Resources.Owner
		tags, ok := TagsForPrometheus[s3Name]
		if !ok {
			fmt.Printf("No tags found for bucket %s\n", s3Name)
			continue
		}
		//tags of buckets change to tags you have defined on s3 buckets
		enviroment := tags["Enviroment"]
		namespace := tags["Namespace"]
		tenant := tags["Tenant"]

		for method, requestSize := range s3Resources.RequestSizes {
			switch method {
			case MethodGET:
				collector.s3TotalGetRequestSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(requestSize))
			case MethodPOST:
				collector.s3TotalPostRequestSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(requestSize))
			case MethodHEAD:
				collector.s3TotalHeadRequestSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(requestSize))
			case MethodPUT:
				collector.s3TotalPutRequestSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(requestSize))
			}

		}
		for method, responseSize := range s3Resources.ResponseSizes {
			switch method {
			case MethodGET:
				collector.s3TotalGetResponseSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodPOST:
				collector.s3TotalPostResponseSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodHEAD:
				collector.s3TotalHeadResponseSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodPUT:
				collector.s3TotalPutResponseSizeMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			}
		}

		for method, responseSize := range s3Resources.Methods {
			switch method {
			case MethodGET:
				collector.s3TotalNumberOfGetRequestsMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodPOST:
				collector.s3TotalNumberOfPostRequestsMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodHEAD:
				collector.s3TotalNumberOfHeadRequestsMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			case MethodPUT:
				collector.s3TotalNumberOfPutRequestsMetric.WithLabelValues(s3Name, method, region, owner, enviroment, namespace, tenant).Set(float64(responseSize))
			}
		}
	}

	collector.s3TotalGetRequestSizeMetric.Collect(ch)
	collector.s3TotalGetResponseSizeMetric.Collect(ch)
	collector.s3TotalPutRequestSizeMetric.Collect(ch)
	collector.s3TotalPutResponseSizeMetric.Collect(ch)
	collector.s3TotalPostRequestSizeMetric.Collect(ch)
	collector.s3TotalPostResponseSizeMetric.Collect(ch)
	collector.s3TotalHeadRequestSizeMetric.Collect(ch)
	collector.s3TotalHeadResponseSizeMetric.Collect(ch)
	collector.s3TotalNumberOfGetRequestsMetric.Collect(ch)
	collector.s3TotalNumberOfPutRequestsMetric.Collect(ch)
	collector.s3TotalNumberOfPostRequestsMetric.Collect(ch)
	collector.s3TotalNumberOfHeadRequestsMetric.Collect(ch)
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
func (collector *ionosCollector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func (collector *s3Collector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func (collector *postgresCollector) GetMutex() *sync.RWMutex {
	return collector.mutex
}

func StartPrometheus(m *sync.RWMutex) {

	ic := newIonosCollector(m)
	s3c := newS3Collector(m)
	pc := newPostgresCollector(m)
	prometheus.MustRegister(ic)
	prometheus.MustRegister(s3c)
	prometheus.MustRegister(pc)
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
	// PrintDCTotals(mutex)
	httpRequestsTotal.WithLabelValues("/healthcheck", r.Method).Inc()
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}
