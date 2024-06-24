package internal

import (
	"fmt"
	"log"
	"os"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func GetEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s not set, returning %s\n", key, fallback)
		return fallback
	} else {
		if value == "" {
			fmt.Printf("%s set but empty, returning %s\n", key, fallback)
			return fallback
		} else {
			fmt.Printf("%s set, returning %s\n", key, value)
			return value
		}
	}
}

func HasLogsFolder(client *s3.S3, bucketName string) bool {
	result, err := client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String("logs/"),
	})

	if err != nil {
		fmt.Println("Error listing objects in bucket: ", err)
		return false
	}

	return len(result.Contents) > 0
}

func GetHeadBucket(client *s3.S3, bucketName string) error {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err := client.HeadBucket(input)
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok && reqErr.StatusCode() == 403 {
			log.Printf("Skipping bucket %s due to Forbidden error: %v\n", bucketName, err)
			return err
		}
		log.Printf("Problem getting the location for bucket %s: %v\n", bucketName, err)
		return err
	}
	log.Printf("Bucket %s exists and is accessible\n", bucketName)
	return nil
}

// func addTagsToBucket(client *s3.S3, bucketName string) {
// 	tags := []*s3.Tag{}

// 	switch bucketName {
// 	case "nbc-bucket01-logs":
// 		tags = []*s3.Tag{
// 			{
// 				Key:   aws.String("Tenant"),
// 				Value: aws.String("Niedersachsen"),
// 			},
// 		}
// 	case "dbp-test-bucketlogs":
// 		tags = []*s3.Tag{
// 			{
// 				Key:   aws.String("Tenant"),
// 				Value: aws.String("Brandenburg"),
// 			},
// 		}
// 	case "dbp-test4logbucket":
// 		tags = []*s3.Tag{
// 			{
// 				Key:   aws.String("Tenant"),
// 				Value: aws.String("Thueringen"),
// 			},
// 		}
// 	case "dbp-test5-logbucket":
// 		tags = []*s3.Tag{
// 			{
// 				Key:   aws.String("Tenant"),
// 				Value: aws.String("HPIBosscloud"),
// 			},
// 		}
// 	default:
// 		tags = []*s3.Tag{

// 			{
// 				Key:   aws.String("Enviroment"),
// 				Value: aws.String("Production"),
// 			},
// 			{
// 				Key:   aws.String("Namespace"),
// 				Value: aws.String("Some Namespace"),
// 			},
// 		}
// 	}
// 	input := &s3.PutBucketTaggingInput{
// 		Bucket: aws.String(bucketName),
// 		Tagging: &s3.Tagging{
// 			TagSet: tags,
// 		},
// 	}
// 	_, err := client.PutBucketTagging(input)
// 	if err != nil {
// 		log.Printf("Error adding tags to bucket %s: %v\n", bucketName, err)
// 	} else {
// 		fmt.Printf("Successfully added tags to bucekt %s\n", bucketName)
// 	}
// }
