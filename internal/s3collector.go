package internal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

const (
	objectPerPage = 100
	maxConcurrent = 10
)

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

func S3CollectResources(m *sync.RWMutex, cycletime int32) {
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
		"de":           {"de", "", "+", "https://s3-eu-central-1.ionoscloud.com"},
		"eu-central-2": {"eu-central-2", "", "+", "https://s3-eu-central-2.ionoscloud.com"},

		// Add more endpoints as needed
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrent)

	for {
		for endpoint, config := range endpoints {
			if _, exists := IonosS3Buckets[endpoint]; exists {
				continue
			}

			client, err := createS3ServiceClient(config.Region, config.AccessKey, config.SecretKey, config.Endpoint)

			if err != nil {
				fmt.Printf("Erropr creating service client for endpoint %s: %v\n", endpoint, err)
				continue
			}

			fmt.Println("Using service client for endpoint:", endpoint)

			result, err := client.ListBuckets(nil)

			if err != nil {
				fmt.Println("Error while Listing Buckets", err)
				continue
			}

			wg.Add(len(result.Buckets))

			for _, bucket := range result.Buckets {
				semaphore <- struct{}{}
				go func(bucketName string) {
					defer func() {
						<-semaphore
						wg.Done()
					}()

					processBucket(client, bucketName)
				}(*bucket.Name)
			}

		}
		wg.Wait()
		CalculateS3Totals(m)
		fmt.Println("This is end of before sleep")

		time.Sleep(time.Duration(cycletime) * time.Second)
	}

}

func processBucket(client *s3.S3, bucketName string) {
	var (
		totalGetMethods    int32 = 0
		totalPutMethods    int32 = 0
		totalGetMethodSize int64 = 0
		totalPutMethodSize int64 = 0
	)

	continuationToken := ""

	for {
		objectList, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(bucketName),
			Prefix:            aws.String("logs/"),
			ContinuationToken: aws.String(continuationToken),
			MaxKeys:           aws.Int64(objectPerPage),
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case "NoSuchBucket":
					fmt.Printf("bucket %s does not exist\n", bucketName)
				default:
					fmt.Printf("error listing objects in bucket %s: %s\n", bucketName, aerr.Message())
				}
			}
			return
		}

		if len(objectList.Contents) == 0 {
			fmt.Printf("bucket %s does not contain any objects with the 'logs/' prefix\n", bucketName)
			return
		}

		for _, object := range objectList.Contents {
			downloadInput := &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
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
			sizeGet, err := strconv.ParseInt(sizeStrGet, 10, 64)

			if err != nil {
				fmt.Println("Error parsing GET size:", err)
				continue
			}
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

		if !aws.BoolValue(objectList.IsTruncated) {
			break
		}

		continuationToken = *objectList.NextContinuationToken

	}
	IonosS3Buckets[bucketName] = IonosS3Resources{
		Name:               bucketName,
		GetMethods:         totalGetMethods,
		PutMethods:         totalPutMethods,
		TotalGetMethodSize: int32(totalGetMethodSize),
		TotalPutMethodSize: int32(totalPutMethodSize),
	}
}

func CalculateS3Totals(m *sync.RWMutex) {
	var (
		getMethodTotal     int32
		putMethodTotal     int32
		getMethodSizeTotal int64
		putMethodSizeTotal int64
	)
	for _, s3Resources := range IonosS3Buckets {
		getMethodTotal += s3Resources.GetMethods
		putMethodTotal += s3Resources.PutMethods
		getMethodSizeTotal += int64(s3Resources.TotalGetMethodSize)
		putMethodSizeTotal += int64(s3Resources.TotalPutMethodSize)
	}
	m.Lock()
	defer m.Unlock()

	TotalGetMethods = getMethodTotal
	TotalPutMethods = putMethodTotal
	TotalGetMethodSize = getMethodSizeTotal
	TotalPutMethodSize = putMethodSizeTotal
}
