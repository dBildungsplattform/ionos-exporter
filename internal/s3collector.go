package internal

import (
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
	IonosS3Buckets = make(map[string]Metrics)
	//map of maps for bucket tags stores tags for every bucket
	//one bucket can have more tags.
	TagsForPrometheus = make(map[string]map[string]string)
)

// object for Metrics
type Metrics struct {
	Methods       map[string]int32
	RequestSizes  map[string]int64
	ResponseSizes map[string]int64
	Regions       string
	Owner         string
}

const (
	MethodGET  = "GET"
	MethodPUT  = "PUT"
	MethodPOST = "POST"
	MethodHEAD = "HEAD"
)

// how many objects to scan per page
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
		log.Printf("Error establishing session with AWS S3 Endpoint: %v", err)
		return nil, fmt.Errorf("error establishing session with AWS S3 Endpoint: %s", err)
	}
	return s3.New(sess), nil
}

func S3CollectResources(m *sync.RWMutex, cycletime int32) {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	// file, _ := os.Create("S3ioutput.txt")
	// defer file.Close()

	// oldStdout := os.Stdout
	// defer func() { os.Stdout = oldStdout }()
	// os.Stdout = file
	if accessKey == "" || secretKey == "" {
		log.Println("AWS credentials are nto set in the enviroment variables.")
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
		for endpoint, config := range endpoints {

			if _, exists := IonosS3Buckets[endpoint]; exists {
				continue
			}
			client, err := createS3ServiceClient(config.Region, config.AccessKey, config.SecretKey, config.Endpoint)

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
					//check if exists if not initialise
					metrics := Metrics{
						Methods:       make(map[string]int32),
						RequestSizes:  make(map[string]int64),
						ResponseSizes: make(map[string]int64),
						Regions:       config.Region,
					}
					IonosS3Buckets[bucketName] = metrics

				}
				wg.Add(1)
				go func(client *s3.S3, bucketName string) {
					defer wg.Done()
					getBucketTags(client, bucketName)
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

/*
function for processing buckets getting the Traffic of all the operations
and their sizes.
*/
func processBucket(client *s3.S3, bucketName string) {

	var wg sync.WaitGroup
	var logEntryRegex = regexp.MustCompile(`(GET|PUT|HEAD|POST) \/[^"]*" \d+ \S+ (\d+|-) (\d+|-) \d+ (\d+|-)`)
	semaphore := make(chan struct{}, maxConcurrent)
	continuationToken := ""

	metrics := Metrics{
		Methods:       make(map[string]int32),
		RequestSizes:  make(map[string]int64),
		ResponseSizes: make(map[string]int64),
		Regions:       "",
		Owner:         "",
	}
	metrics.Regions = *client.Config.Region

	//getting owner
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
	//main loop
	for {

		//get all objects in a bucket use max keys defined in global scope and go through
		//the pages of a bucket
		objectList, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(bucketName),
			Prefix:            aws.String("logs/"),
			ContinuationToken: aws.String(continuationToken),
			MaxKeys:           aws.Int64(objectPerPage),
		})
		//error handling
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
		//check if the bucket has any objects in logs folder
		if len(objectList.Contents) == 0 {
			log.Printf("bucket %s does not contain any objects with the 'logs/' prefix\n", bucketName)
			return
		}
		//iterate through those objects and check the input of logs
		//here we are using concurrency
		for _, object := range objectList.Contents {

			objectKey := *object.Key
			wg.Add(1)
			semaphore <- struct{}{}
			go func(bucketNme, objectkey string) {
				defer func() {
					<-semaphore
					wg.Done()
				}()
				downloadInput := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(objectKey),
				}
				result, err := client.GetObject(downloadInput)
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						if awsErr.Code() == "AccessDenied" {
							log.Printf("Access Denied error for object %s in bucket %s\n", objectKey, bucketName)
							return
						}
					}
					log.Println("Error downloading object", err)
					return
				}
				defer result.Body.Close()
				logContent, err := io.ReadAll(result.Body)
				if err != nil {
					log.Println("Problem reading the body", err)
				}
				//check for matches using regex we are checkign for GET, PUT, POST, HEAD
				//and their response/request size
				matches := logEntryRegex.FindAllStringSubmatch(string(logContent), -1)

				for _, match := range matches {
					metricsMutex.Lock()

					method := match[1]
					requestSizeStr := match[3]
					responseSizeStr := match[2]

					if requestSizeStr != "-" {
						requestSize, err := strconv.ParseInt(requestSizeStr, 10, 64)
						if err != nil {
							log.Printf("Error parsing size: %v", err)
						}
						metrics.RequestSizes[method] += requestSize
					}
					if responseSizeStr != "-" {
						responseSize, err := strconv.ParseInt(responseSizeStr, 10, 64)
						if err != nil {
							log.Printf("Error parsing size: %v", err)
						}
						metrics.ResponseSizes[method] += responseSize
					}

					metrics.Methods[method]++
					metricsMutex.Unlock()
				}
			}(bucketName, *object.Key)
		}
		//if there is no more pages break the loop
		if !aws.BoolValue(objectList.IsTruncated) {
			break
		}
		//go to next page
		continuationToken = *objectList.NextContinuationToken
	}
	wg.Wait()
	//make it thread safe with a mutex
	metricsMutex.Lock()
	IonosS3Buckets[bucketName] = metrics
	metricsMutex.Unlock()
}

/*
function for getting bucket Tags, takes two parameters, the service client
and the bucket name, then it checks for tags using the aws sdk GetBucketTagging
no return value it saves everything to map of maps for Tags which is sent
to prometheus
*/
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
