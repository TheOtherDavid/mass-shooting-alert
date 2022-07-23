package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func queryS3Bucket() (lastModified time.Time, err error) {
	//So they have an S3 bucket, and we should get the file
	bucket := "mass-shooting-tracker-data"
	// TODO: Dynamically construct this
	year := "2022"
	//Target filename: 2022-data.json
	filename := year + "-data.json"

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")

	if accessKey == "" {
		errors.New("Env variable access_key not found")
		return time.Time{}, err
	}

	if secretKey == "" {
		errors.New("Env variable secret_key not found")
		return time.Time{}, err
	}

	client := s3.New(s3.Options{
		Region:      "us-east-2",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	})

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(filename),
	}

	p := s3.NewListObjectsV2Paginator(client, params)
	// Iterate through the Amazon S3 object pages.
	var s3File S3File

	for p.HasMorePages() {
		// next page takes a context
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return time.Time{}, err
		}
		//Take first (probably only) record
		file := page.Contents[0]
		s3File = S3File{
			LastModified: *file.LastModified,
			Key:          *file.Key,
		}
	}

	println(s3File.Key)
	return s3File.LastModified, nil

}

type S3File struct {
	Key          string
	LastModified time.Time
}

func getIncidents() (incidents []Incident, err error) {
	//So they have an S3 bucket, and we should get the file
	bucket := "mass-shooting-tracker-data"
	// TODO: Dynamically construct this
	year := "2022"
	//Target filename: 2022-data.json
	filename := year + "-data.json"

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")

	client := s3.New(s3.Options{
		Region:      "us-east-2",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	})

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}

	result, err := client.GetObject(context.TODO(), params)
	if err != nil {
		return
	}

	defer result.Body.Close()
	body1, err := io.ReadAll(result.Body)
	if err != nil {
		return
	}

	_ = json.Unmarshal([]byte(string(body1)), &incidents)

	incidents, err = convertDateStringToDate(incidents)
	if err != nil {
		return
	}

	return
}

type Incident struct {
	Date       time.Time
	DateString string   `json:"date"`
	Killed     string   `json:"killed"`
	Wounded    string   `json:"wounded"`
	City       string   `json:"city"`
	Names      []string `json:"names"`
	Sources    []string `json:"sources"`
}
