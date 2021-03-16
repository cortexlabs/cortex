/*
Copyright 2021 Cortex Labs, Inc.

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

package main

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Storage is an interface that abstracts cloud storage uploading
type Storage interface {
	Upload(key string, payload io.Reader, contentType string) error
	Download(key string) ([]byte, error)
	GetLastModified(key string) (time.Time, error)
}

type s3 struct {
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	client     *awss3.S3
	bucket     string
}

// NewS3 creates a new S3 client that satisfies the Storage interface
func NewS3(sess *session.Session, bucket string) Storage {
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)
	client := awss3.New(sess)
	return &s3{
		uploader:   uploader,
		bucket:     bucket,
		downloader: downloader,
		client:     client,
	}
}

// Upload uploads binary data to S3
func (s *s3) Upload(key string, payload io.Reader, contentType string) error {
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Key:         aws.String(key),
		Bucket:      aws.String(s.bucket),
		ContentType: aws.String(contentType),
		Body:        payload,
	})
	return err
}

// Download downloads a file from S3 into memory
func (s *s3) Download(key string) ([]byte, error) {
	buff := &aws.WriteAtBuffer{}
	input := awss3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(s.bucket),
	}

	_, err := s.downloader.Download(buff, &input)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// GetLastModified retrieves the last modified timestamp of an S3 object
func (s *s3) GetLastModified(key string) (time.Time, error) {
	input := awss3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(s.bucket),
	}
	obj, err := s.client.GetObject(&input)
	if err != nil {
		return time.Time{}, err
	}

	return *obj.LastModified, nil
}
