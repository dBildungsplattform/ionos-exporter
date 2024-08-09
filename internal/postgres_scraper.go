package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	psql "github.com/ionos-cloud/sdk-go-dbaas-postgres"
	"github.com/joho/godotenv"
)

type IonosPostgresResources struct {
	ClusterName   string
	CPU           int32
	RAM           int32
	Storage       int32
	Owner         string
	DatabaseNames []string
	Telemetry     []TelemetryMetric
}

type TelemetryMetric struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

type TelemetryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []TelemetryMetric `json:"result"`
	} `json:"data"`
}

var (
	ClusterCoresTotal     int32 = 0
	ClusterRamTotal       int32 = 0
	ClusterTotal          int32 = 0
	IonosPostgresClusters       = make(map[string]IonosPostgresResources)
)

func PostgresCollectResources(m *sync.RWMutex, configPath, envFile string, cycletime int32) {
	err := godotenv.Load(envFile)
	if err != nil {
		fmt.Println("Error loading .env file (optional)")
	}

	cfgENV := psql.NewConfigurationFromEnv()
	apiClient := psql.NewAPIClient(cfgENV)

	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	for {
		processCluster(apiClient, m, config.Metrics)
		time.Sleep(time.Duration(cycletime) * time.Second)
	}
}

func processCluster(apiClient *psql.APIClient, m *sync.RWMutex, metrics []MetricConfig) {
	datacenters, err := fetchClusters(apiClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch clusters: %v\n", err)
	}
	if datacenters == nil || datacenters.Items == nil {
		fmt.Fprintf(os.Stderr, "datacenters or datacenters Items are nil\n")
		return
	}
	newIonosPostgresResources := make(map[string]IonosPostgresResources)

	for _, clusters := range *datacenters.Items {
		if clusters.Id == nil || clusters.Properties == nil {
			fmt.Fprintf(os.Stderr, "Cluster or Cluster Properties are nil\n")
			continue
		}
		clusterName := clusters.Properties.DisplayName
		if clusterName == nil {
			fmt.Fprintf(os.Stderr, "Cluster name is nil\n")
			continue
		}
		databaseNames, err := fetchDatabases(apiClient, *clusters.Id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch databases for cluster %s: %v\n", *clusters.Properties.DisplayName, err)
			continue
		}
		databaseOwner, err := fetchOwner(apiClient, *clusters.Id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch owner for database %s: %v\n", *clusters.Properties.DisplayName, err)
			continue
		}

		telemetryData := make([]TelemetryMetric, 0)

		for _, metricConfig := range metrics {
			telemetryResp, err := fetchTelemetryMetrics(os.Getenv("IONOS_TOKEN"), fmt.Sprintf("%s{postgres_cluster=\"%s\"}", metricConfig.Name, *clusters.Id))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch telemetry metrics for cluster %s: %v\n", *clusters.Id, err)
				continue
			}
			telemetryData = append(telemetryData, telemetryResp.Data.Result...)
		}

		newIonosPostgresResources[*clusters.Properties.DisplayName] = IonosPostgresResources{
			ClusterName:   *clusters.Properties.DisplayName,
			CPU:           *clusters.Properties.Cores,
			RAM:           *clusters.Properties.Ram,
			Storage:       *clusters.Properties.StorageSize,
			DatabaseNames: databaseNames,
			Owner:         databaseOwner,
			Telemetry:     telemetryData,
		}
	}
	m.Lock()
	IonosPostgresClusters = newIonosPostgresResources
	m.Unlock()

}

func fetchClusters(apiClient *psql.APIClient) (*psql.ClusterList, error) {
	datacenters, resp, err := apiClient.ClustersApi.ClustersGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling ClustersApi: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if datacenters.Items == nil {
		return nil, fmt.Errorf("no items in resource")
	}

	return &datacenters, nil
}

func fetchDatabases(apiClient *psql.APIClient, clusterID string) ([]string, error) {
	databases, resp, err := apiClient.DatabasesApi.DatabasesList(context.Background(), clusterID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling DatabasesApi: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if databases.Items == nil {
		return nil, fmt.Errorf("no databases found for cluster %s", clusterID)
	}

	var databaseNames []string

	for _, db := range *databases.Items {
		if db.Properties != nil && db.Properties.Name != nil {
			databaseNames = append(databaseNames, *db.Properties.Name)
		}
	}
	return databaseNames, nil
}

func fetchOwner(apiClient *psql.APIClient, clusterID string) (string, error) {
	databases, resp, err := apiClient.DatabasesApi.DatabasesList(context.Background(), clusterID).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling DatabasesApi: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return "", err
	}

	if databases.Items == nil {
		return "", fmt.Errorf("no databases found for cluster %s", clusterID)
	}
	var owner = ""
	for _, db := range *databases.Items {
		if db.Properties != nil && db.Properties.Name != nil {
			owner = *db.Properties.Owner
		}
	}
	return owner, nil
}

func fetchTelemetryMetrics(apiToken, query string) (*TelemetryResponse, error) {
	req, err := http.NewRequest("GET", "https://dcd.ionos.com/telemetry/api/v1/query_range", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", time.Now().Add(-time.Hour).Format(time.RFC3339))
	q.Add("end", time.Now().Format(time.RFC3339))
	q.Add("step", "60")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var telemetryResp TelemetryResponse
	if err := json.NewDecoder(resp.Body).Decode(&telemetryResp); err != nil {
		fmt.Printf("Fialed to decode json response: %v\n", err)
		return nil, err
	}

	return &telemetryResp, nil
}
