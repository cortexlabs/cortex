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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Storage is an interface that abstracts cloud storage uploading
type Storage interface {
	Upload(key string, payload io.Reader, contentType string) error
}

type s3 struct {
	uploader *s3manager.Uploader
	bucket   string
}

// NewS3 creates a new S3 client that satisfies the Storage interface
func NewS3(sess *session.Session, bucket string) Storage {
	uploader := s3manager.NewUploader(sess)
	return &s3{uploader: uploader, bucket: bucket}
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
