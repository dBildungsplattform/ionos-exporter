package internal

import (
	"context"
	"fmt"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/joho/godotenv"
)

var (
	ipName   string
	totalIPs int32
	IonosIPs = make(map[string]IonosIPResources)
)

type IonosIPResources struct {
	IPName   string
	TotalIPs int32
}

func IPCollectResources(apiClient *ionoscloud.APIClient) {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	newIonosIPResources := make(map[string]IonosIPResources)
	ipBlocks, _, err := apiClient.IPBlocksApi.IpblocksGet(context.Background()).Depth(3).Execute()

	if err != nil {
		fmt.Println("Problem with the API Client")
	}

	totalIPs = 0
	for _, ips := range *ipBlocks.Items {
		totalIPs += *ips.Properties.Size

		newIonosIPResources[*ips.Properties.Name] = IonosIPResources{

			TotalIPs: totalIPs,
		}
	}

}
