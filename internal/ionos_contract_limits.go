package internal

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/prometheus/client_golang/prometheus"
)

type ContractLimitsCollector struct {
	mutex        sync.RWMutex
	contractData *ionoscloud.Contracts
	promDescs    map[string]*prometheus.Desc
}

func NewContractLimitsCollector() *ContractLimitsCollector {
	return &ContractLimitsCollector{
		promDescs: make(map[string]*prometheus.Desc),
	}
}

// Uncheked Collector: Descriptions will be generated dynamically
func (c *ContractLimitsCollector) Describe(ch chan<- *prometheus.Desc) {}

func (c *ContractLimitsCollector) StartScrape(cycletime int32) {
	cfgENV := ionoscloud.NewConfigurationFromEnv()
	apiClient := ionoscloud.NewAPIClient(cfgENV)

	for {
		contracts, resp, err := apiClient.ContractResourcesApi.ContractsGet(context.Background()).Execute()
		c.mutex.Lock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `ContractResourcesApi.ContractsGet``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %+v\n", resp)
			c.contractData = nil
		} else {
			c.contractData = &contracts
		}
		c.mutex.Unlock()
		time.Sleep(time.Duration(cycletime) * time.Second)
	}
}

func (c *ContractLimitsCollector) Collect(ch chan<- prometheus.Metric) {
	fetchErrorMetric := 1.0
	c.mutex.RLock()
	//Ensure clean finish in case of errors
	defer func() {
		c.mutex.RUnlock()
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "Error while converting IONOS response to prometheus metrics: %v\n", err)
		}
		ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(
			"ionos_contract_fetch_error",
			"Error during fetch/generation of contract resource limits metrics",
			nil,
			nil,
		), prometheus.GaugeValue, fetchErrorMetric)
	}()

	if c.contractData != nil && c.contractData.Items != nil {
		for _, contract := range *c.contractData.Items {
			if contract.Properties == nil || contract.Properties.ResourceLimits == nil || contract.Properties.ContractNumber == nil {
				fmt.Fprintf(os.Stderr, "Contract is missing neccessary field, skip: %+v\n", contract)
				continue
			}
			contractId := fmt.Sprintf("%d", *contract.Properties.ContractNumber)
			values := reflect.ValueOf(*contract.Properties.ResourceLimits)
			names := values.Type()
			for i := 0; i < values.NumField(); i++ {
				name := names.Field(i).Name
				value := values.Field(i).Elem()
				if !value.CanInt(){
					fmt.Fprintf(os.Stderr, "Expected int for metrics, but %q is %v, skip.\n", name, value.Kind())
					continue
				}
				ch <- prometheus.MustNewConstMetric(c.getDesc(name), prometheus.GaugeValue, float64(value.Int()), contractId)
			}
		}
		fetchErrorMetric = 0
	}
}

func (c *ContractLimitsCollector) getDesc(name string) *prometheus.Desc {
	if desc, ok := c.promDescs[name]; ok {
		return desc
	} else {
		desc := prometheus.NewDesc(
			"ionos_contract_"+ToSnake(name),
			"Contract resource limits metrics via IONOS API. More details: https://api.ionos.com/docs/cloud/v6/#tag/Contract-resources/operation/contractsGet",
			[]string{"contract"},
			nil,
		)
		c.promDescs[name] = desc
		return desc
	}
}
