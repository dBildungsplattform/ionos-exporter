package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type PostgresqlV2Cluster struct {
	ID      string
	Name    string
	Version string
}

type PostgresqlV2ItemProperties struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PostgresqlV2Item struct {
	ID         string                     `json:"id"`
	Properties PostgresqlV2ItemProperties `json:"properties"`
}

type PostgresqlV2Collection struct {
	Items []PostgresqlV2Item `json:"items"`
}

var (
	postgresqlV2Clusters   []PostgresqlV2Cluster
	postgresqlV2FetchError float64
	postgresqlV2Mutex      sync.RWMutex
)

func PostgresqlV2CollectResources(cycletime int32) {
	for {
		scrapePostgresqlV2()
		time.Sleep(time.Duration(cycletime) * time.Second)
	}
}

func scrapePostgresqlV2() {
	token := os.Getenv("IONOS_TOKEN")

	endpoints := []string{
		"https://postgresql.de-txl.ionos.com/v2/clusters",
		"https://postgresql.de-fra.ionos.com/v2/clusters",
	}

	clusters := make([]PostgresqlV2Cluster, 0)
	fetchError := 0.0

	for _, endpoint := range endpoints {
		collection, err := fetchPostgresqlV2Clusters(token, endpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", endpoint, err)
			fetchError = 1.0
			continue
		}
		for idx := range collection.Items {
			item := collection.Items[idx]

			clusterID := "UNKNOWN"
			if item.ID != "" {
				clusterID = item.ID
			}
			clusterName := "UNKNOWN"
			if item.Properties.Name != "" {
				clusterName = item.Properties.Name
			}
			clusterVersion := "UNKNOWN"
			if item.Properties.Version != "" {
				clusterVersion = item.Properties.Version
			}

			clusters = append(clusters, PostgresqlV2Cluster{
				ID:      clusterID,
				Name:    clusterName,
				Version: clusterVersion,
			})
		}
	}

	postgresqlV2Mutex.Lock()
	postgresqlV2Clusters = clusters
	postgresqlV2FetchError = fetchError
	postgresqlV2Mutex.Unlock()
}

func fetchPostgresqlV2Clusters(token, url string) (*PostgresqlV2Collection, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var collection PostgresqlV2Collection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, err
	}

	return &collection, nil
}
