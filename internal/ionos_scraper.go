package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
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
	Cores                int32  // Amount of CPU cores in the whole DC, regardless whether it is a VM or Kubernetscluster
	Ram                  int32  // Amount of RAM in the whole DC, regardless whether it is a VM or Kubernetscluster
	Servers              int32  // Amount of servers in the whole DC
	DCId                 string // UUID od the datacenter
	NLBs                 int32  //Number of Networkloadbalancers
	ALBs                 int32  //Number of Applicationloadbalanceers
	NATs                 int32  //Number of NAT Gateways
	NLBRules             int32  //Number of NLB Rules
	ALBRules             int32  //Number of ALB Rueles
	ALBName              string //ALB Name
	NLBName              string //NLB Name
	NLBRuleName          string //Rule name of NLB
	ALBRuleName          string //Rule name of ALB
	IPName               string //IP Name
	TotalIPs             int32  //Number of total IP-s
	TotalAPICallFailures int32
}

func CollectResources(m *sync.RWMutex, cycletime int32) {

	// err := godotenv.Load(".env")
	// if err != nil {
	// 	fmt.Println("Error loading .env file")
	// }
	cfgENV := ionoscloud.NewConfigurationFromEnv()

	// cfg.Debug = true
	cfgENV.Debug = false
	apiClient := ionoscloud.NewAPIClient(cfgENV)

	totalAPICallFailures := 0
	for {
		datacenters, resp, err := apiClient.DataCentersApi.DatacentersGet(context.Background()).Depth(depth).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `DataCentersApi.DatacentersGet``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
			totalAPICallFailures++
			continue
		}
		// fmt.Println("DATACENTER", datacenters)
		newIonosDatacenters := make(map[string]IonosDCResources)
		for _, datacenter := range *datacenters.Items {
			var (
				coresTotalDC         int32 = 0
				ramTotalDC           int32 = 0
				serverTotalDC        int32 = 0
				nlbTotalDC           int32 = 0
				nlbTotalRulesDC      int32 = 0
				albTotalRulesDC      int32 = 0
				albTotalDC           int32 = 0
				natTotalDC           int32 = 0
				albNames             string
				nlbNames             string
				albRuleNames         string
				nlbRuleNames         string
				totalIPs             int32 = 0
				totalAPICallFailures int32 = 0
			)
			servers, resp, err := apiClient.ServersApi.DatacentersServersGet(context.Background(), *datacenter.Id).Depth(depth).Execute()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error when calling `ServersApi.DatacentersServersGet``: %v\n", err)
				fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
				totalAPICallFailures++
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
				DCId:                 *datacenter.Id,
				Cores:                coresTotalDC,
				Ram:                  ramTotalDC,
				Servers:              serverTotalDC,
				NLBs:                 nlbTotalDC,
				ALBs:                 albTotalDC,
				NATs:                 natTotalDC,
				NLBRules:             nlbTotalRulesDC,
				ALBRules:             albTotalRulesDC,
				ALBName:              albNames,
				NLBName:              nlbNames,
				ALBRuleName:          albRuleNames,
				NLBRuleName:          nlbRuleNames,
				TotalIPs:             totalIPs,
				TotalAPICallFailures: totalAPICallFailures,
			}

		}

		m.Lock()
		IonosDatacenters = newIonosDatacenters
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

/*
Retrieves a list of NAT Gateways which are associated with specific datanceter using the ionoscloud API Client

Parameters:
apiClient: An instance of APIClient for making API Requests
datacenter Pointer to an ionoscloud.Datacenter object representing the target datacenter.

Returns:
- *ionoscloud.NatGateways: A pointer to ionoscloud.NatGateways which has NAT List or an error if it fails
If successful, it returns a pointer to the fetched NATs, otherwise it returns nil and an error message.
*/
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

/*
Retrieves a list of Network Load Balancers (NLB) which are associated with specific datanceter using the ionoscloud API Client

Parameters:
apiClient: An instance of APIClient for making API Requests
datacenter Pointer to an ionoscloud.Datacenter object representing the target datacenter.

Returns:
- *ionoscloud.NetworkLoadBalancers: A pointer to ionoscloud.ApplicationLoadbalancers which has ALB List or an error if it fails
If successful, it returns a pointer to the fetched ALBs, otherwise it returns nil and an error message.
*/
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

/*
retrievers a list of IP Blocks from ionoscloud API

Parameters:
  - apiClient: An instance of ionoscloud.APIClient

Returns:

- pointer to ionoscloud.IpBlocks containing the fetched IP blocks, or nil if there are no items
in the resource.
- error: An error if there was an issue making the API call or if no IP blocks were found.
*/
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

/*
Retrieves a list of Application Load Balancers (ALB) which are associated with specific datanceter using the ionoscloud API Client

Parameters:
apiClient: An instance of APIClient for making API Requests
datacenter Pointer to an ionoscloud.Datacenter object representing the target datacenter.

Returns:
- *ionoscloud.ApplicationLoadBalancers: A pointer to ionoscloud.ApplicationLoadbalancers which has ALB List or an error if it fails
If successful, it returns a pointer to the fetched ALBs, otherwise it returns nil and an error message.
*/
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

/*
Calculates total number of IP addresses from a list of IP Blocks

Parameters:
- ipBlocks: A pointer to ionoscloud.IpBlocks containing a list of IP blocks to process.

Returns:
- The total number of Ip addresses summed from all IP Blocks
*/
func processIPBlocks(ipBlocks *ionoscloud.IpBlocks) int32 {

	var totalIPs int32

	for _, ips := range *ipBlocks.Items {
		if ips.Properties != nil && ips.Properties.Size != nil {
			totalIPs += *ips.Properties.Size
		} else {
			fmt.Println("Ip Properties or Ip Properties Size is nil")
		}
	}
	return totalIPs
}

/*
process a list of Network Load Balancers to extract information about NLB names
and total forwarding rules across all NLBs.

Parameter:
  - a pointer to the NetworkLoadbalaners containig a list of NLBs to process

Returns:
  - string: names of loadbalancers
  - int32: total number of forwarding rules

If any NLB or its associated forwarding rules are nil, they are skipped during processing.
*/
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

/*
process a list of Application Load Balancers ALBs to extract information about ALB names and
total forwarding rules across al ALBs

Parameters:
  - a pointer to ApplicationLoadBalancers containing a list of ALBs to process

Returns:
  - string: names of application loadbalancers
  - int32: total number of forwarding rules
*/
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
