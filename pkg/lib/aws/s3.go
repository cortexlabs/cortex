/*
Copyright 2020 Cortex Labs, Inc.

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

package aws

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const DefaultS3Region string = endpoints.UsWest2RegionID

var S3Regions strset.Set

func init() {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	S3Regions = strset.New()

	for _, p := range partitions {
		if p.ID() == endpoints.AwsPartitionID || p.ID() == endpoints.AwsCnPartitionID {
			for id := range p.Regions() {
				S3Regions.Add(id)
			}
		}
	}
}

func S3Path(bucket string, key string) string {
	return "s3://" + filepath.Join(bucket, key)
}

func JoinS3Path(paths ...string) string {
	if len(paths) == 0 {
		return ""
	}
	paths[0] = paths[0][5:]
	return "s3://" + filepath.Join(paths...)
}

func SplitS3Path(s3Path string) (string, string, error) {
	if !IsValidS3Path(s3Path) {
		return "", "", ErrorInvalidS3aPath(s3Path)
	}
	fullPath := s3Path[len("s3://"):]
	slashIndex := strings.Index(fullPath, "/")
	bucket := fullPath[0:slashIndex]
	key := fullPath[slashIndex+1:]

	return bucket, key, nil
}

func SplitS3aPath(s3aPath string) (string, string, error) {
	if !IsValidS3aPath(s3aPath) {
		return "", "", ErrorInvalidS3aPath(s3aPath)
	}
	fullPath := s3aPath[len("s3a://"):]
	slashIndex := strings.Index(fullPath, "/")
	bucket := fullPath[0:slashIndex]
	key := fullPath[slashIndex+1:]

	return bucket, key, nil
}

func IsValidS3Path(s3Path string) bool {
	if !strings.HasPrefix(s3Path, "s3://") {
		return false
	}
	parts := strings.Split(s3Path[5:], "/")
	if len(parts) < 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	return true
}

func IsValidS3aPath(s3aPath string) bool {
	if !strings.HasPrefix(s3aPath, "s3a://") {
		return false
	}
	parts := strings.Split(s3aPath[6:], "/")
	if len(parts) < 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	return true
}

func (c *Client) IsS3PathFile(s3Path string, s3Paths ...string) (bool, error) {
	allS3Paths := append(s3Paths, s3Path)
	for _, s3Path := range allS3Paths {
		bucket, prefix, err := SplitS3Path(s3Path)
		if err != nil {
			return false, err
		}

		exists, err := c.IsS3File(bucket, prefix)
		if err != nil {
			return false, err
		}

		if !exists {
			return false, nil
		}
	}

	return true, nil
}

func (c *Client) IsS3File(bucket string, fileKey string, fileKeys ...string) (bool, error) {
	allFileKeys := append(fileKeys, fileKey)
	for _, key := range allFileKeys {
		_, err := c.S3().HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})

		if IsNotFoundErr(err) {
			return false, nil
		}
		if err != nil {
			return false, errors.Wrap(err, S3Path(bucket, key))
		}
	}

	return true, nil
}

func (c *Client) IsS3PathPrefix(s3Path string, s3Paths ...string) (bool, error) {
	allS3Paths := append(s3Paths, s3Path)
	for _, s3Path := range allS3Paths {
		bucket, prefix, err := SplitS3Path(s3Path)
		if err != nil {
			return false, err
		}

		exists, err := c.IsS3Prefix(bucket, prefix)
		if err != nil {
			return false, err
		}

		if !exists {
			return false, nil
		}
	}

	return true, nil
}

func (c *Client) IsS3Prefix(bucket string, prefix string, prefixes ...string) (bool, error) {
	allPrefixes := append(prefixes, prefix)
	for _, prefix := range allPrefixes {
		out, err := c.S3().ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(prefix),
		})

		if err != nil {
			return false, errors.Wrap(err, S3Path(bucket, prefix))
		}

		if *out.KeyCount == 0 {
			return false, nil
		}
	}

	return true, nil
}

func (c *Client) IsS3PathDir(s3Path string, s3Paths ...string) (bool, error) {
	allS3Paths := append(s3Paths, s3Path)
	for _, s3Path := range allS3Paths {
		bucket, prefix, err := SplitS3Path(s3Path)
		if err != nil {
			return false, err
		}

		exists, err := c.IsS3Dir(bucket, prefix)
		if err != nil {
			return false, err
		}

		if !exists {
			return false, nil
		}
	}

	return true, nil
}

func (c *Client) IsS3Dir(bucket string, dirPath string, dirPaths ...string) (bool, error) {
	fullDirPaths := make([]string, len(dirPaths)+1)
	allDirPaths := append(dirPaths, dirPath)
	for i, dirPath := range allDirPaths {
		fullDirPaths[i] = s.EnsureSuffix(dirPath, "/")
	}
	return c.IsS3Prefix(bucket, fullDirPaths[0], fullDirPaths[1:]...)
}

func (c *Client) UploadReaderToS3(data io.Reader, bucket string, key string) error {
	_, err := c.S3Uploader().Upload(&s3manager.UploadInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(key),
		Body:                 data,
		ACL:                  aws.String("private"),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	if err != nil {
		return errors.Wrap(err, S3Path(bucket, key))
	}

	return nil
}

func (c *Client) UploadFileToS3(path string, bucket string, key string) error {
	file, err := files.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return c.UploadReaderToS3(file, bucket, key)
}

func (c *Client) UploadBytesToS3(data []byte, bucket string, key string) error {
	return c.UploadReaderToS3(bytes.NewReader(data), bucket, key)
}

func (c *Client) UploadStringToS3(str string, bucket string, key string) error {
	return c.UploadReaderToS3(strings.NewReader(str), bucket, key)
}

func (c *Client) UploadJSONToS3(obj interface{}, bucket string, key string) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.UploadBytesToS3(jsonBytes, bucket, key)
}

func (c *Client) UploadMsgpackToS3(obj interface{}, bucket string, key string) error {
	msgpackBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return err
	}
	return c.UploadBytesToS3(msgpackBytes, bucket, key)
}

// for downloading files, s3manager.Downloader.Download() is faster than s3.S3.GetObject()
// overwrites existing file
func (c *Client) DownloadFileFromS3(path string, bucket string, key string) error {
	file, err := files.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = c.S3Downloader().Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return errors.Wrap(err, S3Path(bucket, key))
	}

	return nil
}

// for reading into memory, s3.S3.GetObject() seems faster than s3manager.Downloader.Download() with aws.NewWriteAtBuffer([]byte{})
// returned io.ReadCloser should be closed by the caller
func (c *Client) ReadReaderFromS3(bucket string, key string) (io.ReadCloser, error) {
	response, err := c.S3().GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return nil, errors.Wrap(err, S3Path(bucket, key))
	}

	return response.Body, nil
}

func (c *Client) ReadBufferFromS3(bucket string, key string) (*bytes.Buffer, error) {
	reader, err := c.ReadReaderFromS3(bucket, key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	return buf, nil
}

func (c *Client) ReadBytesFromS3(bucket string, key string) ([]byte, error) {
	buf, err := c.ReadBufferFromS3(bucket, key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) ReadStringFromS3(bucket string, key string) (string, error) {
	buf, err := c.ReadBufferFromS3(bucket, key)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *Client) ReadJSONFromS3(objPtr interface{}, bucket string, key string) error {
	jsonBytes, err := c.ReadBytesFromS3(bucket, key)
	if err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(jsonBytes, objPtr), S3Path(bucket, key))
}

func (c *Client) ReadMsgpackFromS3(objPtr interface{}, bucket string, key string) error {
	msgpackBytes, err := c.ReadBytesFromS3(bucket, key)
	if err != nil {
		return err
	}
	return errors.Wrap(msgpack.Unmarshal(msgpackBytes, objPtr), S3Path(bucket, key))
}

func (c *Client) ReadStringFromS3Path(s3Path string) (string, error) {
	bucket, key, err := SplitS3Path(s3Path)
	if err != nil {
		return "", err
	}
	return c.ReadStringFromS3(bucket, key)
}

func (c *Client) ReadBytesFromS3Path(s3Path string) ([]byte, error) {
	bucket, key, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}
	return c.ReadBytesFromS3(bucket, key)
}

func (c *Client) ListDir(bucket string, prefix string, maxResults int64) ([]*s3.Object, error) {
	prefix = s.EnsureSuffix(prefix, "/")
	return c.ListPrefix(bucket, prefix, maxResults)
}

func (c *Client) ListPathDir(s3Path string, maxResults int64) ([]*s3.Object, error) {
	s3Path = s.EnsureSuffix(s3Path, "/")
	return c.ListPathPrefix(s3Path, maxResults)
}

func (c *Client) ListPrefix(bucket string, prefix string, maxResults int64) ([]*s3.Object, error) {
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(maxResults),
	}

	output, err := c.S3().ListObjectsV2(listObjectsInput)
	if err != nil {
		return nil, errors.Wrap(err, S3Path(bucket, prefix))
	}

	return output.Contents, nil
}

func (c *Client) ListPathPrefix(s3Path string, maxResults int64) ([]*s3.Object, error) {
	bucket, prefix, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}
	return c.ListPrefix(bucket, prefix, maxResults)
}

func (c *Client) DeleteDir(bucket string, prefix string, continueIfFailure bool) error {
	prefix = s.EnsureSuffix(prefix, "/")
	return c.DeletePrefix(bucket, prefix, continueIfFailure)
}

// Note: this seems to be a lot faster than using s3manager.NewDeleteListIterator()
func (c *Client) DeletePrefix(bucket string, prefix string, continueIfFailure bool) error {
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(1000),
	}

	var subErr error

	err := c.S3().ListObjectsV2Pages(listObjectsInput,
		func(listObjectsOutput *s3.ListObjectsV2Output, lastPage bool) bool {
			deleteObjects := make([]*s3.ObjectIdentifier, len(listObjectsOutput.Contents))
			for i, object := range listObjectsOutput.Contents {
				deleteObjects[i] = &s3.ObjectIdentifier{Key: object.Key}
			}
			deleteObjectsInput := &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &s3.Delete{
					Objects: deleteObjects,
					Quiet:   aws.Bool(true),
				},
			}
			_, newSubErr := c.S3().DeleteObjects(deleteObjectsInput)
			if newSubErr != nil {
				subErr = newSubErr
				if !continueIfFailure {
					return false
				}
			}
			return true
		})

	if subErr != nil {
		return errors.Wrap(subErr, S3Path(bucket, prefix))
	}

	if err != nil {
		return errors.Wrap(err, S3Path(bucket, prefix))
	}

	return nil
}

func (c *Client) CreateBucket(bucket string) error {
	var bucketConfiguration *s3.CreateBucketConfiguration
	if c.Region != "us-east-1" {
		bucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(c.Region),
		}
	}
	_, err := c.S3().CreateBucket(&s3.CreateBucketInput{
		Bucket:                    aws.String(bucket),
		CreateBucketConfiguration: bucketConfiguration,
	})
	if err != nil {
		return errors.Wrap(err, "creating bucket "+bucket)
	}
	return nil
}

// Checks bucket existence and accessibility with credentials
func (c *Client) DoesBucketExist(bucket string) (bool, error) {
	_, err := c.S3().HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return false, nil
			case "Forbidden":
				return false, ErrorBucketInaccessible(bucket)
			}
		}
		return false, errors.Wrap(err, "bucket "+bucket)
	}

	return true, nil
}

func GetBucketRegion(bucket string) (string, error) {
	sess := session.Must(session.NewSession()) // credentials are not necessary for this request, and will not be used
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), sess, bucket, endpoints.UsWest2RegionID)
	if err != nil {
		return "", ErrorBucketNotFound(bucket)
	}
	return region, nil
}
