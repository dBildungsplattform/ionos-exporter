package internal

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/prometheus/client_golang/prometheus"
)

type KubernetesCollector struct {
	mutex             sync.RWMutex
	clusters          []clusterData
	fetchError        float64
	descClusterState  *prometheus.Desc
	descnodepoolState *prometheus.Desc
	descNodeState     *prometheus.Desc
	descFetchError    *prometheus.Desc
}

type clusterData struct {
	id        string
	name      string
	state     string
	nodepools []nodepoolData
}

type nodepoolData struct {
	id    string
	name  string
	state string
	nodes []nodeData
}

type nodeData struct {
	id    string
	name  string
	state string
}

func NewKubernetesCollector() *KubernetesCollector {
	return &KubernetesCollector{
		descClusterState: prometheus.NewDesc(
			"ionos_k8s_cluster_state",
			"Kubernetes cluster state",
			[]string{"cluster_id", "cluster_name", "state"},
			nil,
		),
		descnodepoolState: prometheus.NewDesc(
			"ionos_k8s_nodepool_state",
			"Kubernetes nodepool state",
			[]string{"cluster_id", "cluster_name", "nodepool_id", "nodepool_name", "state"},
			nil,
		),
		descNodeState: prometheus.NewDesc(
			"ionos_k8s_node_state",
			"Kubernetes node state",
			[]string{"cluster_id", "cluster_name", "nodepool_id", "node_id", "node_name", "state"},
			nil,
		),
		descFetchError: prometheus.NewDesc(
			"ionos_k8s_fetch_error",
			"Error during K8s metrics fetch",
			nil,
			nil,
		),
	}
}

func (c *KubernetesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.descClusterState
	ch <- c.descnodepoolState
	ch <- c.descNodeState
	ch <- c.descFetchError
}

func (c *KubernetesCollector) StartScrape(fetchInterval int32) {
	cfgENV := ionoscloud.NewConfigurationFromEnv()
	apiClient := ionoscloud.NewAPIClient(cfgENV)

	for {
		c.scrape(apiClient)
		time.Sleep(time.Duration(fetchInterval) * time.Second)
	}
}

func (c *KubernetesCollector) scrape(apiClient *ionoscloud.APIClient) {
	// Use depth 3 to get the nodepools states as well
	clustersResp, resp, err := apiClient.KubernetesApi.K8sGet(context.Background()).Depth(3).Execute()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.fetchError = 0
	c.clusters = nil

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `KubernetesApi.K8sGet`: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %+v\n", resp)
		c.fetchError = 1
		return
	}

	if clustersResp.Items == nil {
		fmt.Fprintf(os.Stderr, "No Items in response: %+v\n", resp)
		c.fetchError = 1
		return
	}

	for idx := range *clustersResp.Items {
		cluster := (*clustersResp.Items)[idx]

		clusterID := "UNKNOWN"
		if cluster.Id != nil {
			clusterID = *cluster.Id
		}
		clusterName := "UNKNOWN"
		if cluster.Properties != nil && cluster.Properties.Name != nil {
			clusterName = *cluster.Properties.Name
		}
		state := "UNKNOWN"
		if cluster.Metadata != nil && cluster.Metadata.State != nil {
			state = *cluster.Metadata.State
		}

		clusterData := clusterData{
			id:    clusterID,
			name:  clusterName,
			state: state,
		}

		if cluster.Entities != nil && cluster.Entities.Nodepools != nil && cluster.Entities.Nodepools.Items != nil {
			for nodepoolIdx := range *cluster.Entities.Nodepools.Items {
				nodepool := (*cluster.Entities.Nodepools.Items)[nodepoolIdx]

				nodepoolID := "UNKNOWN"
				if nodepool.Id != nil {
					nodepoolID = *nodepool.Id
				}
				nodepoolName := "UNKNOWN"
				if nodepool.Properties != nil && nodepool.Properties.Name != nil {
					nodepoolName = *nodepool.Properties.Name
				}
				nodepoolState := "UNKNOWN"
				if nodepool.Metadata != nil && nodepool.Metadata.State != nil {
					nodepoolState = *nodepool.Metadata.State
				}

				nodepoolData := nodepoolData{
					id:    nodepoolID,
					name:  nodepoolName,
					state: nodepoolState,
				}

				nodesResp, _, nodesErr := apiClient.KubernetesApi.K8sNodepoolsNodesGet(context.Background(), clusterID, nodepoolID).Depth(2).Execute()
				if nodesErr != nil {
					fmt.Fprintf(os.Stderr, "Error fetching nodes for nodepool %s: %v\n", nodepoolID, nodesErr)
					c.fetchError = 1
				} else if nodesResp.Items != nil {
					for nodeIdx := range *nodesResp.Items {
						node := (*nodesResp.Items)[nodeIdx]

						nodeID := "UNKNOWN"
						if node.Id != nil {
							nodeID = *node.Id
						}
						nodeName := "UNKNOWN"
						if node.Properties != nil && node.Properties.Name != nil {
							nodeName = *node.Properties.Name
						}
						nodeState := "UNKNOWN"
						if node.Metadata != nil && node.Metadata.State != nil {
							nodeState = *node.Metadata.State
						}

						nodepoolData.nodes = append(nodepoolData.nodes, nodeData{
							id:    nodeID,
							name:  nodeName,
							state: nodeState,
						})
					}
				}
				clusterData.nodepools = append(clusterData.nodepools, nodepoolData)
			}
		}
		c.clusters = append(c.clusters, clusterData)
	}
}

func (c *KubernetesCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, cluster := range c.clusters {
		ch <- prometheus.MustNewConstMetric(
			c.descClusterState,
			prometheus.GaugeValue, 1,
			cluster.id, cluster.name, cluster.state,
		)

		for _, nodepool := range cluster.nodepools {
			ch <- prometheus.MustNewConstMetric(
				c.descnodepoolState,
				prometheus.GaugeValue, 1,
				cluster.id, cluster.name, nodepool.id, nodepool.name, nodepool.state,
			)

			for _, node := range nodepool.nodes {
				ch <- prometheus.MustNewConstMetric(
					c.descNodeState,
					prometheus.GaugeValue, 1,
					cluster.id, cluster.name, nodepool.id, node.id, node.name, node.state,
				)
			}
		}
	}

	ch <- prometheus.MustNewConstMetric(
		c.descFetchError,
		prometheus.GaugeValue, c.fetchError,
	)
}
