package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/joho/godotenv"
)

var (
	CoresTotal       int32 = 0
	RamTotal         int32 = 0
	ServerTotal      int32 = 0
	DataCenters      int32 = 0
	IonosDatacenters       = make(map[string]IonosDCResources) //Key is the name of the datacenter
	depth            int32 = 1
)

type IonosDCResources struct {
	Cores   int32  // Amount of CPU cores in the whole DC, regardless whether it is a VM or Kubernetscluster
	Ram     int32  // Amount of RAM in the whole DC, regardless whether it is a VM or Kubernetscluster
	Servers int32  // Amount of servers in the whole DC
	DCId    string // UUID od the datacenter
}

func CollectResources(m *sync.RWMutex, cycletime int32) {

	file, _ := os.Create("ionosoutput.txt")

	defer file.Close()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = file

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	// username := os.Getenv("IONOS_USERNAME")
	// password := os.Getenv("IONOS_PASSWORD")
	// cfg := ionoscloud.NewConfiguration(username, password, "", "")
	cfgENV := ionoscloud.NewConfigurationFromEnv()

	// cfg.Debug = true
	cfgENV.Debug = true
	apiClient := ionoscloud.NewAPIClient(cfgENV)

	for {
		datacenters, resp, err := apiClient.DataCentersApi.DatacentersGet(context.Background()).Depth(depth).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `DataCentersApi.DatacentersGet``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			os.Exit(1)
		}
		fmt.Println("DATACENTER", datacenters)
		newIonosDatacenters := make(map[string]IonosDCResources)
		for _, datacenter := range *datacenters.Items {
			var (
				coresTotalDC  int32 = 0
				ramTotalDC    int32 = 0
				serverTotalDC int32 = 0
			)
			servers, resp, err := apiClient.ServersApi.DatacentersServersGet(context.Background(), *datacenter.Id).Depth(depth).Execute()
			//fmt.Println("SERVERS", servers)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error when calling `ServersApi.DatacentersServersGet``: %v\n", err)
				fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			}
			serverTotalDC = int32(len(*servers.Items))

			for _, server := range *servers.Items {
				coresTotalDC += *server.Properties.Cores
				ramTotalDC += *server.Properties.Ram
			}

			newIonosDatacenters[*datacenter.Properties.Name] = IonosDCResources{
				DCId:    *datacenter.Id,
				Cores:   coresTotalDC,
				Ram:     ramTotalDC,
				Servers: serverTotalDC,
			}

		}

		m.Lock()
		IonosDatacenters = newIonosDatacenters
		LoadbalancerCollector(apiClient)
		IPCollectResources(apiClient)
		S3CollectResources()
		m.Unlock()
		CalculateDCTotals(m)
		time.Sleep(time.Duration(cycletime) * time.Second)
	}
}

func CalculateDCTotals(m *sync.RWMutex) {
	var (
		serverTotal      int32
		ramTotal         int32
		coresTotal       int32
		datacentersTotal int32
	)
	m.RLock()
	for _, dcResources := range IonosDatacenters {
		serverTotal += dcResources.Servers
		ramTotal += dcResources.Ram
		coresTotal += dcResources.Cores
	}
	datacentersTotal = int32(len(IonosDatacenters))
	m.RUnlock()
	m.Lock()
	ServerTotal = serverTotal
	RamTotal = ramTotal
	CoresTotal = coresTotal
	DataCenters = datacentersTotal
	m.Unlock()
}
func PrintDCResources(m *sync.RWMutex) {
	m.RLock()
	defer m.RUnlock()
	for dcName, dcResources := range IonosDatacenters {
		fmt.Fprintf(os.Stdout, "%s:\n    - UUID: %s\n", dcName, dcResources.DCId)
		fmt.Fprintf(os.Stdout, "    - Servers: %d\n", dcResources.Servers)
		fmt.Fprintf(os.Stdout, "%s:\n    - Cores: %d\n", dcName, dcResources.Cores)
		fmt.Fprintf(os.Stdout, "    - Ram: %d GB\n", dcResources.Ram/1024)
	}
}
func PrintDCTotals(m *sync.RWMutex) {
	m.RLock()
	defer m.RUnlock()
	log.Printf("Total - Datacenters: %d\n", DataCenters)
	log.Printf("Total - Servers: %d\n", ServerTotal)
	log.Printf("Total - Cores: %d\n", CoresTotal)
	log.Printf("Total - Ram: %d GB\n", RamTotal/1024)
}

//problemen mit ionos log bucket konnte nicht testen richtig
//noch problemen mit aktuallisierung von log data wenn welche geloescht werden
//problem sa paralelizacijom. logove mogu kalkulisati kako treba
//ali ne tako brzo
