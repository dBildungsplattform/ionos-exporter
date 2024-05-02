package internal

import (
	"fmt"
	"os"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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

func NewS3ServiceClient() (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("eu-central-2"),
		Credentials: credentials.NewStaticCredentials("00e556b6437d8a8d1776", "LbypY0AmotQCDDckTz+cAPFI7l0eQvSFeQ1WxKtw", ""),
		Endpoint:    aws.String("https://s3-eu-central-2.ionoscloud.com"),
	})

	if err != nil {
		return nil, err
	}
	return s3.New(sess), nil
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
