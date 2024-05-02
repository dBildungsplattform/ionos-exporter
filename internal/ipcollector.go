package internal

import (
	"context"
	"fmt"
	"os"

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
	file, _ := os.Create("Ipsoutput.txt")

	defer file.Close()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = file

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	newIonosIPResources := make(map[string]IonosIPResources)
	// newIonosIPResources := make(map[string]IonosIPResources)
	ipBlocks, _, err := apiClient.IPBlocksApi.IpblocksGet(context.Background()).Depth(3).Execute()

	if err != nil {
		fmt.Println("Problem with the API Client")
	}

	totalIPs = 0
	for _, ips := range *ipBlocks.Items {
		totalIPs += *ips.Properties.Size
		fmt.Println("Hey this is the size of IPs", totalIPs)

		newIonosIPResources[*ips.Properties.Name] = IonosIPResources{

			TotalIPs: totalIPs,
		}
	}

	fmt.Println("Heyo")

}
