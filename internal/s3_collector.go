package internal

import (
	"sync"

	//"time"

	"github.com/prometheus/client_golang/prometheus"
)

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

func NewS3Collector(m *sync.RWMutex) *s3Collector {
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
		tags := TagsForPrometheus[s3Name]
		// if !ok {
		// 	// fmt.Printf("No tags found for bucket %s\n", s3Name)
		// }
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
