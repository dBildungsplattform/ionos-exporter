package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

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

func GetBoolEnv(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s not set, returning %t\n", key, fallback)
		return fallback, nil
	} else {
		if value == "" {
			fmt.Printf("%s set but empty, returning %t\n", key, fallback)
			return fallback, nil
		} else {
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return fallback, fmt.Errorf("Invalid value for %s=%q (expected true/false): %v", key, value, err)
			}
			return boolValue, nil
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

var toSnakeRe = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnake(s string) string {
	return strings.ToLower(toSnakeRe.ReplaceAllString(s, "${1}_${2}"))
}

// Can be used if any error of a function should be fatal
func Must[T any](t T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return t
}
