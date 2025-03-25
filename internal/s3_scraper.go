package internal

import (
	"bufio"
	"fmt"
	"io"
	"log"
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
	IonosS3Buckets    = make(map[string]Metrics)
	TagsForPrometheus = make(map[string]map[string]string)
	metricsMutex      sync.Mutex
)

type Metrics struct {
	Methods       map[string]int32
	RequestSizes  map[string]int64
	ResponseSizes map[string]int64
	Regions       string
	Owner         string
}

const (
	MethodGET     = "GET"
	MethodPUT     = "PUT"
	MethodPOST    = "POST"
	MethodHEAD    = "HEAD"
	objectPerPage = 1000
	maxConcurrent = 10
)

func createS3ServiceClient(region, accessKey, secretKey, endpoint string) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:    aws.String(endpoint),
	})
	if err != nil {
		log.Printf("Error establishing session with AWS S3 Endpoint: %v", err)
		return nil, fmt.Errorf("error establishing session with AWS S3 Endpoint: %s", err)
	}
	return s3.New(sess), nil
}

func S3CollectResources(m *sync.RWMutex, cycletime int32) {
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")

	if accessKey == "" || secretKey == "" {
		log.Println("AWS credentials are not set in the environment variables.")
		return
	}
	endpoints := map[string]EndpointConfig{
		"eu-central-2": {
			Region:    "eu-central-2",
			AccessKey: accessKey,
			SecretKey: secretKey,
			Endpoint:  "https://s3-eu-central-2.ionoscloud.com",
		},
		"de": {
			Region:    "de",
			AccessKey: accessKey,
			SecretKey: secretKey,
			Endpoint:  "https://s3-eu-central-1.ionoscloud.com",
		},
	}
	semaphore := make(chan struct{}, maxConcurrent)
	for {
		var wg sync.WaitGroup
		for _, endpoint := range endpoints {

			if _, exists := IonosS3Buckets[endpoint.Endpoint]; exists {
				continue
			}
			client, err := createS3ServiceClient(endpoint.Region, accessKey, secretKey, endpoint.Endpoint)

			if err != nil {
				fmt.Printf("Error creating service client for endpoint %s: %v\n", endpoint, err)
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
				if _, exists := IonosS3Buckets[bucketName]; !exists {
					metrics := Metrics{
						Methods:       make(map[string]int32),
						RequestSizes:  make(map[string]int64),
						ResponseSizes: make(map[string]int64),
						Regions:       "",
					}
					IonosS3Buckets[bucketName] = metrics
				}
				wg.Add(1)
				fmt.Println("Processing Bucket: ", bucketName)
				go func(client *s3.S3, bucketName string) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Recovered in goroutine: %v", r)
						}
					}()
					if err := GetHeadBucket(client, bucketName); err != nil {
						if reqErr, ok := err.(awserr.RequestFailure); ok && reqErr.StatusCode() == 403 {
							return
						}
						log.Println("Error checking the bucket head:", err)
						return
					}
					semaphore <- struct{}{}
					defer func() {
						<-semaphore
					}()
					processBucket(client, bucketName)
				}(client, bucketName)
			}

		}
		wg.Wait()
		time.Sleep(time.Duration(cycletime) * time.Second)
	}

}

func processBucket(client *s3.S3, bucketName string) {

	var wg sync.WaitGroup
	logEntryRegex := regexp.MustCompile(`(GET|PUT|HEAD|POST) \/[^"]*" \d+ \S+ (\d+|-) (\d+|-) \d+ (\d+|-)`)
	semaphore := make(chan struct{}, maxConcurrent)

	getBucketTags(client, bucketName)
	metrics := Metrics{
		Methods:       make(map[string]int32),
		RequestSizes:  make(map[string]int64),
		ResponseSizes: make(map[string]int64),
		Regions:       "",
		Owner:         "",
	}
	metrics.Regions = *client.Config.Region

	continuationToken := ""

	getAclInput := &s3.GetBucketAclInput{
		Bucket: aws.String(bucketName),
	}
	getAclOutput, err := client.GetBucketAcl(getAclInput)
	if err != nil {
		log.Printf("Error retrieving ACL for bucket %s: %v\n", bucketName, err)
		return
	}
	if len(*getAclOutput.Owner.DisplayName) > 0 {
		metrics.Owner = *getAclOutput.Owner.DisplayName
	} else {
		metrics.Owner = "Unknown"
	}

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
					log.Printf("bucket %s does not exist\n", bucketName)
				default:
					if awserr, ok := err.(awserr.Error); ok {
						if awserr.Code() == "AccessDenied" {
							log.Println("Bucket not in current endpoint skipping")
						}
					}
					fmt.Printf("error listing objects in bucket %s: %s\n", bucketName, aerr.Message())
				}
			}
			return
		}
		if len(objectList.Contents) == 0 {
			log.Printf("bucket %s does not contain any objects with the 'logs/' prefix\n", bucketName)
			return
		}
		for _, object := range objectList.Contents {
			wg.Add(1)
			semaphore <- struct{}{}
			go func(object *s3.Object) {
				defer wg.Done()
				defer func() { <-semaphore }()
				processObject(client, bucketName, object, logEntryRegex, &metrics)
			}(object)
		}
		if !aws.BoolValue(objectList.IsTruncated) {
			break
		}
		continuationToken = *objectList.NextContinuationToken
	}
	wg.Wait()
	metricsMutex.Lock()
	IonosS3Buckets[bucketName] = metrics
	metricsMutex.Unlock()
}

func getBucketTags(client *s3.S3, bucketName string) {
	tagsOutput, err := client.GetBucketTagging(&s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				log.Printf("Bucket %s does not exist\n", bucketName)
				return
			case "NoSuchTagSet":
				log.Printf("No tags set for Bucket %s\n", bucketName)
				return
			default:
				log.Printf("Error retrieving tags in false endpoint for bucket %s: %s\n", bucketName, aerr.Message())
				return
			}
		} else {
			log.Printf("Error retrieving tags for bucket %s: %s\n", bucketName, err.Error())
			return
		}
	}
	tags := make(map[string]string)
	for _, tag := range tagsOutput.TagSet {
		tags[*tag.Key] = *tag.Value
	}

	metricsMutex.Lock()
	TagsForPrometheus[bucketName] = tags
	metricsMutex.Unlock()
}

func processObject(client *s3.S3, bucketName string, object *s3.Object, logEntryRegex *regexp.Regexp, metrics *Metrics) {
	downloadInput := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(*object.Key),
	}
	result, err := client.GetObject(downloadInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "AccessDenied" {
			log.Printf("Access Denied error for object %s in bucket %s\n", *object.Key, bucketName)
			return
		}
		log.Println("Error downloading object", err)
		return
	}
	defer result.Body.Close()

	reader := bufio.NewReader(result.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Println("Problem reading the body", err)
			}
			break
		}
		processLine(line, logEntryRegex, metrics)
	}
}

func processLine(line []byte, logEntryRegex *regexp.Regexp, metrics *Metrics) {
	matches := logEntryRegex.FindAllStringSubmatch(string(line), -1)
	for _, match := range matches {
		metricsMutex.Lock()
		method := match[1]
		requestSizeStr := match[3]
		responseSizeStr := match[2]

		if requestSizeStr != "-" {
			requestSize, err := strconv.ParseInt(requestSizeStr, 10, 64)
			if err == nil {
				metrics.RequestSizes[method] += requestSize
			}
		}
		if responseSizeStr != "-" {
			responseSize, err := strconv.ParseInt(responseSizeStr, 10, 64)
			if err == nil {
				metrics.ResponseSizes[method] += responseSize
			}
		}
		metrics.Methods[method]++
		metricsMutex.Unlock()
	}
}
