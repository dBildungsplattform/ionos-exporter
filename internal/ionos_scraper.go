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
				continue
			}

			albList, err := fetchApplicationLoadbalancers(apiClient, &datacenter)
			if err != nil {
				fmt.Printf("Error retrieving ALBs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}
			nlbList, err := fetchNetworkLoadBalancers(apiClient, &datacenter)
			if err != nil {
				fmt.Printf("Error retrieving NLBs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}
			natList, err := fetchNATGateways(apiClient, &datacenter)
			if err != nil {
				fmt.Printf("Error retrieving NATs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}
			ipBlocks, err := fetchIPBlocks(apiClient)
			if err != nil {
				fmt.Printf("Error retrieving IPs for datacenter %s: %v\n", *datacenter.Properties.Name, err)
				continue
			}

			totalIPs = processIPBlocks(ipBlocks)
			nlbNames, nlbTotalRulesDC = processNetworkLoadBalancers(nlbList)
			albNames, albTotalRulesDC = processApplicationLoadBalancers(albList)

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
		// CalculateDCTotals(m)
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

func fetchNATGateways(apiClient *ionoscloud.APIClient, datacenter *ionoscloud.Datacenter) (*ionoscloud.NatGateways, error) {
	datacenterId := *datacenter.Id
	natList, resp, err := apiClient.NATGatewaysApi.DatacentersNatgatewaysGet(context.Background(), datacenterId).Depth(2).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling NATGateways API: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if natList.Items == nil {
		return nil, fmt.Errorf("no items in resource")
	}
	return &natList, nil
}

func fetchNetworkLoadBalancers(apiClient *ionoscloud.APIClient, datacenter *ionoscloud.Datacenter) (*ionoscloud.NetworkLoadBalancers, error) {
	datacenterId := *datacenter.Id
	nlbList, resp, err := apiClient.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersGet(context.Background(), datacenterId).Depth(2).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling NetworkLoadbalancers API: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if nlbList.Items == nil {
		return nil, fmt.Errorf("no items in resource")
	}

	return &nlbList, nil
}

func fetchIPBlocks(apiClient *ionoscloud.APIClient) (*ionoscloud.IpBlocks, error) {
	ipBlocks, resp, err := apiClient.IPBlocksApi.IpblocksGet(context.Background()).Depth(2).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling IPBlocks API: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if ipBlocks.Items == nil {
		return nil, fmt.Errorf("no items in resource")
	}

	return &ipBlocks, nil
}

func fetchApplicationLoadbalancers(apiClient *ionoscloud.APIClient, datacenter *ionoscloud.Datacenter) (*ionoscloud.ApplicationLoadBalancers, error) {
	datacenterId := *datacenter.Id
	albList, resp, err := apiClient.ApplicationLoadBalancersApi.DatacentersApplicationloadbalancersGet(context.Background(), datacenterId).Depth(2).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling ApplicationLoadBalancers API: %v\n", err)
		if resp != nil {
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Fprintf(os.Stderr, "No HTTP response received\n")
		}
		return nil, err
	}

	if albList.Items == nil {
		return nil, fmt.Errorf("no items in resource")
	}

	return &albList, nil
}

func processIPBlocks(ipBlocks *ionoscloud.IpBlocks) int32 {
	var totalIPs int32
	for _, ips := range *ipBlocks.Items {
		if ips.Properties != nil && ips.Properties.Size != nil {
			totalIPs += *ips.Properties.Size
		}
	}
	return totalIPs
}

func processNetworkLoadBalancers(nlbList *ionoscloud.NetworkLoadBalancers) (string, int32) {
	var (
		nlbNames        string
		nlbTotalRulesDC int32
	)

	for _, nlb := range *nlbList.Items {
		if nlb.Properties != nil && nlb.Properties.Name != nil {
			nlbNames = *nlb.Properties.Name
		}
		nlbForwardingRules := nlb.Entities.Forwardingrules
		if nlbForwardingRules != nil && nlbForwardingRules.Items != nil {
			nlbTotalRulesDC = int32(len(*nlbForwardingRules.Items))
			for _, rule := range *nlbForwardingRules.Items {
				if rule.Properties != nil && rule.Properties.Name != nil {
					nlbNames = *rule.Properties.Name
				}
			}
		}
	}
	return nlbNames, nlbTotalRulesDC
}

func processApplicationLoadBalancers(albList *ionoscloud.ApplicationLoadBalancers) (string, int32) {
	var (
		albNames        string
		albTotalRulesDC int32
	)

	for _, alb := range *albList.Items {
		if alb.Properties != nil && alb.Properties.Name != nil {
			albNames = *alb.Properties.Name
		}
		albForwardingRules := alb.Entities.Forwardingrules
		if albForwardingRules != nil && albForwardingRules.Items != nil {
			albTotalRulesDC = int32(len(*albForwardingRules.Items))
			for _, rule := range *albForwardingRules.Items {
				if rule.Properties != nil && rule.Properties.Name != nil {
					albNames = *rule.Properties.Name
				}
			}
		}
	}
	return albNames, albTotalRulesDC
}
