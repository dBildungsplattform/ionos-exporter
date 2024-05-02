package internal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	TotalGetMethods    int32 = 0
	TotalGetMethodSize int64 = 0
	TotalPutMethodSize int64 = 0
	TotalPutMethods    int32 = 0
	IonosS3Buckets           = make(map[string]IonosS3Resources)
)

type IonosS3Resources struct {
	Name               string
	GetMethods         int32
	PutMethods         int32
	HeadMethods        int32
	PostMethods        int32
	TotalGetMethodSize int32
	TotalPutMethodSize int32
}

func createS3ServiceClient(region, accessKey, secretKey, endpoint string) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:    aws.String(endpoint),
	})
	if err != nil {
		return nil, fmt.Errorf("error establishing session with AWS S3 Endpoint: %s", err)
	}
	return s3.New(sess), nil
}

func S3CollectResources() {
	// accessKey := os.Getenv("IONOS_ACCESS_KEY")
	// secretKey := os.Getenv("IONOS_SECRET_KEY")

	file, _ := os.Create("S3ioutput.txt")
	defer file.Close()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = file
	//TODO YAML konfiguration
	// Define endpoint configurations
	endpoints := map[string]struct {
		Region, AccessKey, SecretKey, Endpoint string
	}{
		"de":           {"de", "00e556b6437d8a8d1776", "LbypY0AmotQCDDckTz+cAPFI7l0eQvSFeQ1WxKtw", "https://s3-eu-central-1.ionoscloud.com"},
		"eu-central-2": {"eu-central-2", "00e556b6437d8a8d1776", "LbypY0AmotQCDDckTz+cAPFI7l0eQvSFeQ1WxKtw", "https://s3-eu-central-2.ionoscloud.com"},
		// Add more endpoints as needed
	}

	bucketCounts := make(map[string]struct {
		totalGetMethods    int32
		totalPutMethods    int32
		totalGetMethodSize int64
		totalPutMethodSize int64
	})
	newS3IonosResources := make(map[string]IonosS3Resources)
	serviceClients := make(map[string]*s3.S3)

	// var totalLineCount int = 0
	// Create service clients for each endpoint

	for endpoint, config := range endpoints {
		if _, exists := serviceClients[endpoint]; exists {
			continue
		}
		client, err := createS3ServiceClient(config.Region, config.AccessKey, config.SecretKey, config.Endpoint)
		if err != nil {
			fmt.Printf("Error creating service client for endpoint %s: %v\n", endpoint, err)
			continue
		}
		serviceClients[endpoint] = client

		fmt.Println("Using service client for endpoint: %s\n", endpoint)
		// serviceClient := s3.New(sess)

		result, err := client.ListBuckets(nil)
		if err != nil {
			fmt.Println("Problem with the Listing of the Buckets")
		}

		for _, buckets := range result.Buckets {
			var (
				totalGetMethods    int32 = 0
				totalPutMethods    int32 = 0
				totalGetMethodSize int64 = 0
				totalPutMethodSize int64 = 0
			)

			objectList, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
				Bucket: aws.String(*buckets.Name),
				Prefix: aws.String("logs/"),
			})
			if err != nil {
				fmt.Println("Could not use the service client to list objects")
				continue
			}
			if len(objectList.Contents) == 0 {
				continue
			}
			for _, object := range objectList.Contents {
				downloadInput := &s3.GetObjectInput{
					Bucket: aws.String(*buckets.Name),
					Key:    aws.String(*object.Key),
				}

				result, err := client.GetObject(downloadInput)
				if err != nil {
					fmt.Println("Error downloading object", err)
					continue
				}
				defer result.Body.Close()

				logContent, err := io.ReadAll(result.Body)
				if err != nil {
					fmt.Println("Error reading log content:", err)
					continue
				}
				fields := strings.Fields(string(logContent))

				bucketMethod := fields[9]

				sizeStrGet := fields[14]
				sizeStrPut := fields[16]
				if bucketMethod == "PUT" {
					fmt.Println("This si the PUT Method")
				}
				sizeGet, err := strconv.ParseInt(sizeStrGet, 10, 64)
				sizePut, err := strconv.ParseInt(sizeStrPut, 10, 64)

				if err != nil {
					fmt.Println("Error parsing PUT size:", err)
					continue
				}
				switch bucketMethod {
				case "\"GET":
					totalGetMethods++
					totalGetMethodSize += sizeGet
				case "\"PUT":
					totalPutMethods++
					totalPutMethodSize += sizePut
				default:
				}
			}
			bucketCounts[*buckets.Name] = struct {
				totalGetMethods    int32
				totalPutMethods    int32
				totalGetMethodSize int64
				totalPutMethodSize int64
			}{
				totalGetMethods:    totalGetMethods,
				totalPutMethods:    totalPutMethods,
				totalGetMethodSize: totalGetMethodSize,
				totalPutMethodSize: totalPutMethodSize,
			}
			fmt.Println("This is the bucket Name", *buckets.Name)
		}

	}

	for bucketName, counts := range bucketCounts {
		newS3IonosResources[bucketName] = IonosS3Resources{
			Name:               bucketName,
			GetMethods:         counts.totalGetMethods,
			PutMethods:         counts.totalPutMethods,
			TotalGetMethodSize: int32(counts.totalGetMethodSize),
			TotalPutMethodSize: int32(counts.totalPutMethodSize),
		}
	}
	IonosS3Buckets = newS3IonosResources
	// time.Sleep(time.Duration(cycletime) * time.Second)
}

// CalculateS3Totals(m)

// func CalculateS3Totals(m *sync.RWMutex) {
// 	var (
// 		getMethodTotal     int32
// 		putMethodTotal     int32
// 		getMethodSizeTotal int64
// 		putMethodSizeTotal int64
// 	)
// 	for _, s3Resources := range IonosS3Buckets {
// 		getMethodTotal += s3Resources.GetMethods
// 		putMethodTotal += s3Resources.PutMethods
// 		getMethodSizeTotal += int64(s3Resources.TotalGetMethodSize)
// 		putMethodSizeTotal += int64(s3Resources.TotalPutMethodSize)
// 	}
// 	TotalGetMethods = getMethodTotal

// 	fmt.Println("Get method inside a calculate totals program", TotalGetMethods)
// 	TotalPutMethods = putMethodTotal
// 	TotalGetMethodSize = getMethodSizeTotal
// 	TotalPutMethodSize = putMethodSizeTotal
// }
