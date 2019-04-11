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
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/msgpack"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	libs3 "github.com/cortexlabs/cortex/pkg/lib/aws/s3"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
)

func S3Path(key string) string {
	return "s3://" + filepath.Join(cc.Bucket, key)
}

func IsS3File(key string) (bool, error) {
	return IsS3FileExternal(key, cc.Bucket)
}

func IsS3Dir(dirPath string) (bool, error) {
	prefix := s.EnsureSuffix(dirPath, "/")
	return IsS3Prefix(prefix)
}

func IsS3Prefix(prefix string) (bool, error) {
	return IsS3PrefixExternal(prefix, cc.Bucket)
}

func IsS3FileExternal(key string, bucket string) (bool, error) {
	_, err := s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
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

func IsS3PrefixExternal(prefix string, bucket string) (bool, error) {
	out, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return false, errors.Wrap(err, prefix)
	}

	hasPrefix := *out.KeyCount > 0
	return hasPrefix, nil
}

func IsS3aPrefixExternal(s3aPath string) (bool, error) {
	bucket, key, err := libs3.SplitS3aPath(s3aPath)
	if err != nil {
		return false, err
	}
	return IsS3PrefixExternal(key, bucket)
}

func UploadBytesToS3(data []byte, key string) error {
	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:                 bytes.NewReader(data),
		Key:                  aws.String(key),
		Bucket:               aws.String(cc.Bucket),
		ACL:                  aws.String("private"),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return errors.Wrap(err, key)
}

func UploadBytesesToS3(data []byte, keys ...string) error {
	fns := make([]func() error, len(keys))
	for i, key := range keys {
		key := key
		fns[i] = func() error {
			return UploadBytesToS3(data, key)
		}
	}
	return parallel.RunFirstErr(fns...)
}

func UploadFileToS3(filePath string, key string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, ErrorReadFile(filePath).Error())
	}
	return UploadBytesToS3(data, key)
}

func UploadBufferToS3(buffer *bytes.Buffer, key string) error {
	return UploadBytesToS3(buffer.Bytes(), key)
}

func UploadStringToS3(str string, key string) error {
	str = strings.TrimSpace(str)
	return UploadBytesToS3([]byte(str), key)
}

func UploadJSONToS3(obj interface{}, key string) error {
	jsonBytes, err := libjson.MarshalJSON(obj)
	if err != nil {
		return err
	}
	return UploadBytesToS3(jsonBytes, key)
}

func ReadJSONFromS3(objPtr interface{}, key string) error {
	jsonBytes, err := ReadBytesFromS3(key)
	if err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(jsonBytes, objPtr), key)
}

func UploadMsgpackToS3(obj interface{}, key string) error {
	msgpackBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return err
	}
	return UploadBytesToS3(msgpackBytes, key)
}

func ReadMsgpackFromS3(objPtr interface{}, key string) error {
	msgpackBytes, err := ReadBytesFromS3(key)
	if err != nil {
		return err
	}
	return errors.Wrap(msgpack.Unmarshal(msgpackBytes, objPtr), key)
}

func ReadStringFromS3(key string) (string, error) {
	response, err := s3Client.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(cc.Bucket),
	})

	if err != nil {
		return "", errors.Wrap(err, key)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.String(), nil
}

func ReadBytesFromS3(key string) ([]byte, error) {
	response, err := s3Client.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(cc.Bucket),
	})

	if err != nil {
		return nil, errors.Wrap(err, key)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.Bytes(), nil
}

func DeleteFromS3ByPrefix(prefix string, continueIfFailure bool) error {
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(cc.Bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(1000),
	}

	var subErr error

	err := s3Client.ListObjectsV2Pages(listObjectsInput,
		func(listObjectsOutput *s3.ListObjectsV2Output, lastPage bool) bool {
			deleteObjects := make([]*s3.ObjectIdentifier, len(listObjectsOutput.Contents))
			for i, object := range listObjectsOutput.Contents {
				deleteObjects[i] = &s3.ObjectIdentifier{Key: object.Key}
			}
			deleteObjectsInput := &s3.DeleteObjectsInput{
				Bucket: aws.String(cc.Bucket),
				Delete: &s3.Delete{
					Objects: deleteObjects,
					Quiet:   aws.Bool(true),
				},
			}
			_, newSubErr := s3Client.DeleteObjects(deleteObjectsInput)
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
