package internal

import (
	"context"
	"fmt"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

var (
	nlbNames        string
	albNames        string
	nlbTotalRulesDC int32
	nlbRuleNames    string
	albTotalRulesDC int32
	albRuleNames    string

	IonosLoadbalancers = make(map[string]IonosLBResources)
)

type IonosLBResources struct {
	NLBs        int32
	ALBs        int32
	NATs        int32
	NLBRules    int32
	ALBRules    int32
	ALBName     string
	NLBName     string
	NLBRuleName string
	ALBRuleName string
}

func LoadbalancerCollector(apiClient *ionoscloud.APIClient) {
	// fmt.Println("Hey this is the Loadbalancer Collector")

	// file, _ := os.Create("LoadBalancerOutput.txt")

	// defer file.Close()

	// oldStdout := os.Stdout
	// defer func() { os.Stdout = oldStdout }()
	// os.Stdout = file
	datacenter, _, _ := apiClient.DataCentersApi.DatacentersGet(context.Background()).Depth(3).Execute()

	newIonosLBResources := make(map[string]IonosLBResources)
	for _, datacenter := range *datacenter.Items {

		var (
			nlbTotalDC      int32 = 0
			nlbTotalRulesDC int32 = 0
			albTotalRulesDC int32 = 0
			albTotalDC      int32 = 0
			natTotalDC      int32 = 0
			albNames        string
			nlbNames        string
			albRuleNames    string
			nlbRuleNames    string
		)

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

		newIonosLBResources[*datacenter.Properties.Name] = IonosLBResources{
			NLBs:        nlbTotalDC,
			ALBs:        albTotalDC,
			NATs:        natTotalDC,
			NLBRules:    nlbTotalRulesDC,
			ALBRules:    albTotalRulesDC,
			ALBName:     albNames,
			NLBName:     nlbNames,
			ALBRuleName: albRuleNames,
			NLBRuleName: nlbRuleNames,
		}
	}
	IonosLoadbalancers = newIonosLBResources
}
