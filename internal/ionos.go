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
	Cores       int32  // Amount of CPU cores in the whole DC, regardless whether it is a VM or Kubernetscluster
	Ram         int32  // Amount of RAM in the whole DC, regardless whether it is a VM or Kubernetscluster
	Servers     int32  // Amount of servers in the whole DC
	DCId        string // UUID od the datacenter
	NLBs        int32
	ALBs        int32
	NATs        int32
	NLBRules    int32
	ALBRules    int32
	ALBName     string
	NLBName     string
	NLBRuleName string
	ALBRuleName string
	IPName      string
	TotalIPs    int32
}

func CollectResources(m *sync.RWMutex, cycletime int32) {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	cfgENV := ionoscloud.NewConfigurationFromEnv()

	// cfg.Debug = true
	cfgENV.Debug = false
	apiClient := ionoscloud.NewAPIClient(cfgENV)

	for {
		datacenters, resp, err := apiClient.DataCentersApi.DatacentersGet(context.Background()).Depth(depth).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `DataCentersApi.DatacentersGet``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			os.Exit(1)
		}
		// fmt.Println("DATACENTER", datacenters)
		newIonosDatacenters := make(map[string]IonosDCResources)
		for _, datacenter := range *datacenters.Items {
			var (
				coresTotalDC    int32 = 0
				ramTotalDC      int32 = 0
				serverTotalDC   int32 = 0
				nlbTotalDC      int32 = 0
				nlbTotalRulesDC int32 = 0
				albTotalRulesDC int32 = 0
				albTotalDC      int32 = 0
				natTotalDC      int32 = 0
				albNames        string
				nlbNames        string
				albRuleNames    string
				nlbRuleNames    string
				totalIPs        int32 = 0
			)
			servers, resp, err := apiClient.ServersApi.DatacentersServersGet(context.Background(), *datacenter.Id).Depth(depth).Execute()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error when calling `ServersApi.DatacentersServersGet``: %v\n", err)
				fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			}
			albList, _, err := apiClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersGet(context.Background(), *datacenter.Id).Depth(3).Execute()
			if err != nil {
				fmt.Printf("Error retrieving ALBs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}
			nlbList, _, err := apiClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersGet(context.Background(), *datacenter.Id).Depth(3).Execute()
			if err != nil {
				fmt.Printf("Error retrieving NLBs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}
			natList, _, _ := apiClient.NATGatewaysApi.DatacentersNatgatewaysGet(context.Background(), *datacenter.Id).Depth(3).Execute()
			if err != nil {
				fmt.Printf("Error retrieving NATs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}

			ipBlocks, _, err := apiClient.IPBlocksApi.IpblocksGet(context.Background()).Depth(3).Execute()

			if err != nil {
				fmt.Println("Problem with the API Client")
			}

			for _, ips := range *ipBlocks.Items {
				totalIPs += *ips.Properties.Size
			}

			for _, nlbRulesAndLabels := range *nlbList.Items {
				if nlbRulesAndLabels.Properties != nil && nlbRulesAndLabels.Properties.Name != nil {
					nlbNames = *nlbRulesAndLabels.Properties.Name
				}

				nlbForwardingRules := nlbRulesAndLabels.Entities.Forwardingrules
				if nlbForwardingRules != nil && nlbForwardingRules.Items != nil {
					nlbTotalRulesDC = int32(len(*nlbForwardingRules.Items))
					for _, ruleItems := range *nlbForwardingRules.Items {
						if ruleItems.Properties != nil && ruleItems.Properties.Name != nil {
							nlbRuleNames = *ruleItems.Properties.Name
						}
					}
				}
			}

			for _, albRulesAndLabels := range *albList.Items {
				if albRulesAndLabels.Properties != nil && albRulesAndLabels.Properties.Name != nil {
					albNames = *albRulesAndLabels.Properties.Name
				}
				forwardingRules := albRulesAndLabels.Entities.Forwardingrules
				if forwardingRules != nil && forwardingRules.Items != nil {
					albTotalRulesDC = int32(len(*forwardingRules.Items))

					for _, ruleItems := range *forwardingRules.Items {
						if ruleItems.Properties != nil && ruleItems.Properties.HttpRules != nil {
							for _, ruleName := range *ruleItems.Properties.HttpRules {
								if ruleName.Name != nil {
									albRuleNames = *ruleName.Name
								}
							}
						}
					}
				}
			}
			nlbTotalDC = int32(len(*nlbList.Items))
			albTotalDC = int32(len(*albList.Items))
			natTotalDC = int32(len(*natList.Items))
			serverTotalDC = int32(len(*servers.Items))

			for _, server := range *servers.Items {
				coresTotalDC += *server.Properties.Cores
				ramTotalDC += *server.Properties.Ram
			}

			newIonosDatacenters[*datacenter.Properties.Name] = IonosDCResources{
				DCId:        *datacenter.Id,
				Cores:       coresTotalDC,
				Ram:         ramTotalDC,
				Servers:     serverTotalDC,
				NLBs:        nlbTotalDC,
				ALBs:        albTotalDC,
				NATs:        natTotalDC,
				NLBRules:    nlbTotalRulesDC,
				ALBRules:    albTotalRulesDC,
				ALBName:     albNames,
				NLBName:     nlbNames,
				ALBRuleName: albRuleNames,
				NLBRuleName: nlbRuleNames,
				TotalIPs:    totalIPs,
			}

		}

		m.Lock()
		IonosDatacenters = newIonosDatacenters
		m.Unlock()
		// IPCollectResources(apiClient)
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
