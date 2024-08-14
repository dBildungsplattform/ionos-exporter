package internal

import (
	"io"
	"net/http"
	"sync"

	//"time"

	"github.com/prometheus/client_golang/prometheus"
)

// var mutex *sync.RWMutex

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
	dcMutex := &sync.RWMutex{}
	s3Mutex := &sync.RWMutex{}
	pgMutex := &sync.RWMutex{}

	ionosCollector := NewIonosCollector(dcMutex)
	s3Collector := NewS3Collector(s3Mutex)
	pgCollector := NewPostgresCollector(pgMutex)

	prometheus.MustRegister(ionosCollector)
	prometheus.MustRegister(s3Collector)
	prometheus.MustRegister(pgCollector)
	prometheus.MustRegister(HttpRequestsTotal)
}

var HttpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "http_requests_total",
		Help:        "Total number of HTTP requests",
		ConstLabels: prometheus.Labels{"server": "api"},
	},
	[]string{"route", "method"},
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// PrintDCTotals(mutex)
	HttpRequestsTotal.WithLabelValues("/healthcheck", r.Method).Inc()
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}
