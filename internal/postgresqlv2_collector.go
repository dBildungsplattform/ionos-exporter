package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PostgresqlV2Collector struct {
	clusterMetric  *prometheus.GaugeVec
	descFetchError *prometheus.Desc
}

func NewPostgresqlV2Collector() *PostgresqlV2Collector {
	return &PostgresqlV2Collector{
		clusterMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ionos_postgresql_v2_cluster",
			Help: "PostgreSQL v2 cluster exposed by its id, name and version",
		}, []string{"id", "name", "version"}),
		descFetchError: prometheus.NewDesc(
			"ionos_postgresql_v2_fetch_error",
			"Error during PostgreSQL v2 metrics fetch",
			nil,
			nil,
		),
	}
}

func (collector *PostgresqlV2Collector) Describe(ch chan<- *prometheus.Desc) {
	collector.clusterMetric.Describe(ch)
	ch <- collector.descFetchError
}

func (collector *PostgresqlV2Collector) Collect(ch chan<- prometheus.Metric) {
	postgresqlV2Mutex.RLock()
	defer postgresqlV2Mutex.RUnlock()

	collector.clusterMetric.Reset()
	for _, cluster := range postgresqlV2Clusters {
		collector.clusterMetric.WithLabelValues(cluster.ID, cluster.Name, cluster.Version).Set(1)
	}
	collector.clusterMetric.Collect(ch)

	ch <- prometheus.MustNewConstMetric(
		collector.descFetchError,
		prometheus.GaugeValue, postgresqlV2FetchError,
	)
}
