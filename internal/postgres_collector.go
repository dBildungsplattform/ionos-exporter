package internal

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

//"time"

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

func NewPostgresCollector(m *sync.RWMutex) *postgresCollector {
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
