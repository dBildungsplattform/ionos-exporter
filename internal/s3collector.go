package internal

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type EndpointConfig struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

var (
	// Global totals
	TotalMetrics = Metrics{}
	// IonosS3Buckets
	IonosS3Buckets = make(map[string]Metrics)
)

type Metrics struct {
	Methods       map[string]int32
	RequestSizes  map[string]int64
	ResponseSizes map[string]int64
}

const (
	MethodGET  = "GET"
	MethodPUT  = "PUT"
	MethodPOST = "POST"
	MethodHEAD = "HEAD"
)

const (
	objectPerPage = 100
	maxConcurrent = 10
)

var metricsMutex sync.Mutex

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
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	file, _ := os.Create("S3ioutput.txt")
	defer file.Close()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = file
	fmt.Println("ACESSEKEY", accessKey)
	if accessKey == "" || secretKey == "" {
		fmt.Println("AWS credentials are not set in the environment variables.")
		return
	}
	endpoints := map[string]EndpointConfig{
		"de": {
			Region:    "de",
			AccessKey: accessKey,
			SecretKey: secretKey,
			Endpoint:  "https://s3-eu-central-1.ionoscloud.com",
		},
		"eu-central-2": {
			Region:    "eu-central-2",
			AccessKey: accessKey,
			SecretKey: secretKey,
			Endpoint:  "https://s3-eu-central-2.ionoscloud.com",
		},
	}

	semaphore := make(chan struct{}, maxConcurrent)
	for {
		var wg sync.WaitGroup
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

			for _, bucket := range result.Buckets {
				bucketName := *bucket.Name
				wg.Add(1)
				if err := GetHeadBucket(client, bucketName); err != nil {
					if reqErr, ok := err.(awserr.RequestFailure); ok && reqErr.StatusCode() == 403 {
						wg.Done()
						continue
					}
					fmt.Println("Error checking the bucket head:", err)
					wg.Done()
					continue
				}

				semaphore <- struct{}{}
				go func(bucketName string) {
					defer func() {
						<-semaphore
						wg.Done()
					}()
					processBucket(client, bucketName)
				}(*bucket.Name)
				// wg.Wait() //when we want sequentiel here wait for bucket to finish
			}

		}
		fmt.Println("Before the wait")
		wg.Wait()
		fmt.Println("After the wait")
		fmt.Println("This is before sleep")
		time.Sleep(time.Duration(cycletime) * time.Second)
	}

}

func processBucket(client *s3.S3, bucketName string) {
	// var logEntryRegex = regexp.MustCompile(`(?)(GET|PUT|HEAD|POST) .+? (\d+) (\d+)`)
	// var logEntryRegex = regexp.MustCompile(`(\w+) \/[^"]*" \d+ \S+ (\d+) - \d+ (\d+)`)
	var logEntryRegex = regexp.MustCompile(`(GET|PUT|HEAD|POST) \/[^"]*" \d+ \S+ (\d+|-) (\d+|-) \d+ (\d+|-)`)

	// fmt.Println("Regex Pattern:", logEntryRegex.String())

	metrics := Metrics{
		Methods:       make(map[string]int32),
		RequestSizes:  make(map[string]int64),
		ResponseSizes: make(map[string]int64),
	}

	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
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
					if awserr, ok := err.(awserr.Error); ok {
						if awserr.Code() == "AccessDenied" {
							fmt.Println("ACCESS DENIED")
						}
					}
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
			wg.Add(1)
			semaphore <- struct{}{}
			go func(bucketNme, objectkey string) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				downloadInput := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(*object.Key),
				}

				result, err := client.GetObject(downloadInput)

				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						if awsErr.Code() == "AccessDenied" {
							fmt.Printf("Access Denied error for object %s in bucket %s\n", *object.Key, bucketName)
							return
						}
					}
					fmt.Println("Error downloading object", err)
					return
				}
				defer result.Body.Close()

				logContent, err := io.ReadAll(result.Body)

				// logLine := strings.Fields(string(logContent))

				if err != nil {
					fmt.Println("Problem reading the body", err)
				}

				matches := logEntryRegex.FindAllStringSubmatch(string(logContent), -1)
				fmt.Println("Matches:", matches)
				for _, match := range matches {
					method := match[1]

					requestSizeStr := match[2]
					requestSize, err := strconv.ParseInt(requestSizeStr, 10, 64)
					if err != nil {
						fmt.Printf("Error parsing size : %v", err)
					}

					responseSizeStr := match[3]
					responseSize, err := strconv.ParseInt(responseSizeStr, 10, 64)
					if err != nil {
						fmt.Printf("Error parsing size: %v", err)
					}

					metricsMutex.Lock()
					// fmt.Println("Log line", logLine)

					metrics.Methods[method]++
					metrics.RequestSizes[method] += requestSize
					metrics.ResponseSizes[method] += responseSize
					metricsMutex.Unlock()
				}
			}(bucketName, *object.Key)
		}

		if !aws.BoolValue(objectList.IsTruncated) {
			break
		}
		continuationToken = *objectList.NextContinuationToken
	}
	wg.Wait()
	IonosS3Buckets[bucketName] = metrics
}
