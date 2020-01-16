/*
Copyright 2019 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//go:generate python3 gen_resource_metadata.py
//go:generate gofmt -s -w resource_metadata.go

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Client struct {
	Region          string
	sess            *session.Session
	clients         clients
	accountID       *string
	hashedAccountID *string
}

func NewFromEnv(region string) (*Client, error) {
	return New(region, nil)
}

func NewFromCreds(region string, accessKeyID string, secretAccessKey string) (*Client, error) {
	creds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	return New(region, creds)
}

func NewFromEnvS3Path(s3Path string) (*Client, error) {
	bucket, _, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}
	return NewFromEnvS3Bucket(bucket)
}

func NewFromEnvS3Bucket(bucket string) (*Client, error) {
	region, err := GetBucketRegion(bucket)
	if err != nil {
		return nil, err
	}
	return NewFromEnv(region)
}

func NewFromCredsS3Path(s3Path string, accessKeyID string, secretAccessKey string) (*Client, error) {
	bucket, _, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}
	return NewFromCredsS3Bucket(bucket, accessKeyID, secretAccessKey)
}

func NewFromCredsS3Bucket(bucket string, accessKeyID string, secretAccessKey string) (*Client, error) {
	region, err := GetBucketRegion(bucket)
	if err != nil {
		return nil, err
	}
	return NewFromCreds(region, accessKeyID, secretAccessKey)
}

func New(region string, creds *credentials.Credentials) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		sess:   sess,
		Region: *sess.Config.Region,
	}, nil
}
