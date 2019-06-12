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

package aws

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
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

func (c *Client) S3Path(key string) string {
	return "s3://" + filepath.Join(c.Bucket, key)
}

func (c *Client) IsS3File(key string) (bool, error) {
	_, err := c.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})

	if IsNotFoundErr(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, key)
	}

	return true, nil
}

func (c *Client) IsS3Dir(dirPath string) (bool, error) {
	prefix := s.EnsureSuffix(dirPath, "/")
	return c.IsS3Prefix(prefix)
}

func (c *Client) IsS3Prefix(prefix string) (bool, error) {
	out, err := c.s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(c.Bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return false, errors.Wrap(err, prefix)
	}

	hasPrefix := *out.KeyCount > 0
	return hasPrefix, nil
}

func (c *Client) UploadBytesToS3(data []byte, key string) error {
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Body:                 bytes.NewReader(data),
		Key:                  aws.String(key),
		Bucket:               aws.String(c.Bucket),
		ACL:                  aws.String("private"),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return errors.Wrap(err, key)
}

func (c *Client) UploadBytesesToS3(data []byte, keys ...string) error {
	fns := make([]func() error, len(keys))
	for i, key := range keys {
		key := key
		fns[i] = func() error {
			return c.UploadBytesToS3(data, key)
		}
	}
	return parallel.RunFirstErr(fns...)
}

func (c *Client) UploadFileToS3(filePath string, key string) error {
	data, err := files.ReadFileBytes(filePath)
	if err != nil {
		return err
	}
	return c.UploadBytesToS3(data, key)
}

func (c *Client) UploadBufferToS3(buffer *bytes.Buffer, key string) error {
	return c.UploadBytesToS3(buffer.Bytes(), key)
}

func (c *Client) UploadStringToS3(str string, key string) error {
	str = strings.TrimSpace(str)
	return c.UploadBytesToS3([]byte(str), key)
}

func (c *Client) UploadJSONToS3(obj interface{}, key string) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.UploadBytesToS3(jsonBytes, key)
}

func (c *Client) ReadJSONFromS3(objPtr interface{}, key string) error {
	jsonBytes, err := c.ReadBytesFromS3(key)
	if err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(jsonBytes, objPtr), key)
}

func (c *Client) UploadMsgpackToS3(obj interface{}, key string) error {
	msgpackBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return err
	}
	return c.UploadBytesToS3(msgpackBytes, key)
}

func (c *Client) ReadMsgpackFromS3(objPtr interface{}, key string) error {
	msgpackBytes, err := c.ReadBytesFromS3(key)
	if err != nil {
		return err
	}
	return errors.Wrap(msgpack.Unmarshal(msgpackBytes, objPtr), key)
}

func (c *Client) ReadStringFromS3(key string) (string, error) {
	response, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(c.Bucket),
	})

	if err != nil {
		return "", errors.Wrap(err, key)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.String(), nil
}

func (c *Client) ReadBytesFromS3(key string) ([]byte, error) {
	response, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(c.Bucket),
	})

	if err != nil {
		return nil, errors.Wrap(err, key)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.Bytes(), nil
}

func (c *Client) DeleteFromS3ByPrefix(prefix string, continueIfFailure bool) error {
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.Bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(1000),
	}

	var subErr error

	err := c.s3Client.ListObjectsV2Pages(listObjectsInput,
		func(listObjectsOutput *s3.ListObjectsV2Output, lastPage bool) bool {
			deleteObjects := make([]*s3.ObjectIdentifier, len(listObjectsOutput.Contents))
			for i, object := range listObjectsOutput.Contents {
				deleteObjects[i] = &s3.ObjectIdentifier{Key: object.Key}
			}
			deleteObjectsInput := &s3.DeleteObjectsInput{
				Bucket: aws.String(c.Bucket),
				Delete: &s3.Delete{
					Objects: deleteObjects,
					Quiet:   aws.Bool(true),
				},
			}
			_, newSubErr := c.s3Client.DeleteObjects(deleteObjectsInput)
			if newSubErr != nil {
				subErr = newSubErr
				if !continueIfFailure {
					return false
				}
			}
			return true
		})

	if subErr != nil {
		return errors.Wrap(subErr, prefix)
	}
	return errors.Wrap(err, prefix)
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

func SplitS3aPath(s3aPath string) (string, string, error) {
	if !IsValidS3aPath(s3aPath) {
		return "", "", ErrorInvalidS3aPath(s3aPath)
	}
	fullPath := s3aPath[6:]
	slashIndex := strings.Index(fullPath, "/")
	bucket := fullPath[0:slashIndex]
	key := fullPath[slashIndex+1:]

	return bucket, key, nil
}

func IsS3PrefixExternal(bucket string, prefix string, region string) (bool, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	out, err := s3.New(sess).ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return false, errors.Wrap(err, prefix)
	}

	hasPrefix := *out.KeyCount > 0
	return hasPrefix, nil
}

func IsS3aPrefixExternal(s3aPath string, region string) (bool, error) {
	bucket, prefix, err := SplitS3aPath(s3aPath)
	if err != nil {
		return false, err
	}
	return IsS3PrefixExternal(bucket, prefix, region)
}
