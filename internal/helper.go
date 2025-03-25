package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Metrics []MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

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

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
