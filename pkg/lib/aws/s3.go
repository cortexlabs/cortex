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

package aws

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
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
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
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
		return "", "", ErrorInvalidS3Path(s3Path)
	}
	fullPath := s3Path[len("s3://"):]
	slashIndex := strings.Index(fullPath, "/")
	if slashIndex == -1 {
		return fullPath, "", nil
	}
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
	if len(parts) == 0 {
		return false
	}
	if parts[0] == "" {
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

// List all S3 objects that are "depth" levels or deeper than the given "s3Path".
// Setting depth to 1 effectively translates to listing the objects one level or deeper than the given prefix (aka listing the directory contents).
//
// 1st returned value is the list of paths found at level <depth> or deeper.
// 2nd returned value is the list of paths found at all levels.
func (c *Client) GetNLevelsDeepFromS3Path(s3Path string, depth int, includeDirObjects bool, maxResults *int64) ([]string, []string, error) {
	paths := strset.New()

	_, key, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, nil, err
	}

	allS3Objects, err := c.ListS3PathDir(s3Path, includeDirObjects, maxResults)
	if err != nil {
		return nil, nil, err
	}
	allPaths := ConvertS3ObjectsToKeys(allS3Objects...)

	keySplit := slices.RemoveEmpties(strings.Split(key, "/"))
	for _, path := range allPaths {
		pathSplit := slices.RemoveEmpties(strings.Split(path, "/"))
		if len(pathSplit)-len(keySplit) >= depth {
			computedPath := strings.Join(pathSplit[:len(keySplit)+depth], "/")
			paths.Add(computedPath)
		}
	}

	return paths.Slice(), allPaths, nil
}

func ConvertS3ObjectsToKeys(s3Objects ...*s3.Object) []string {
	paths := make([]string, 0, len(s3Objects))
	for _, object := range s3Objects {
		paths = append(paths, *object.Key)
	}
	return paths
}

func GetBucketRegionFromS3Path(s3Path string) (string, error) {
	bucket, _, err := SplitS3Path(s3Path)
	if err != nil {
		return "", err
	}

	return GetBucketRegion(bucket)
}

func GetBucketRegion(bucket string) (string, error) {
	sess := session.Must(session.NewSession()) // credentials are not necessary for this request, and will not be used
	region, err := s3manager.GetBucketRegion(aws.BackgroundContext(), sess, bucket, endpoints.UsWest2RegionID)
	if err != nil {
		return "", ErrorBucketNotFound(bucket)
	}
	return region, nil
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
			Bucket:  aws.String(bucket),
			Prefix:  aws.String(prefix),
			MaxKeys: aws.Int64(1),
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

func (c *Client) UploadDirToS3(localDirPath string, bucket string, s3Dir string, ignoreFns ...files.IgnoreFn) error {
	localRelPaths, err := files.ListDirRecursive(localDirPath, true, ignoreFns...)
	if err != nil {
		return err
	}

	for _, localRelPath := range localRelPaths {
		localPath := filepath.Join(localDirPath, localRelPath)
		key := filepath.Join(s3Dir, localRelPath)
		if err := c.UploadFileToS3(localPath, bucket, key); err != nil {
			return err
		}
	}

	return nil
}

// returned io.ReadCloser should be closed by the caller
func (c *Client) ReadReaderFromS3(bucket string, key string) (io.ReadCloser, error) {
	// for reading into memory, s3.S3.GetObject() seems faster than s3manager.Downloader.Download() with aws.NewWriteAtBuffer([]byte{})
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

func (c *Client) ReadMsgpackFromS3Path(objPtr interface{}, s3Path string) error {
	bucket, key, err := SplitS3Path(s3Path)
	if err != nil {
		return err
	}

	return c.ReadMsgpackFromS3(objPtr, bucket, key)
}

// overwrites existing file
func (c *Client) DownloadFileFromS3(bucket string, key string, localPath string) error {
	file, err := files.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// for downloading files, s3manager.Downloader.Download() is faster than s3.S3.GetObject()
	_, err = c.S3Downloader().Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return errors.Wrap(err, S3Path(bucket, key))
	}

	return nil
}

func (c *Client) DownloadDirFromS3(bucket string, s3Dir string, localDirPath string, shouldTrimDirPrefix bool, maxFiles *int64) error {
	prefix := s.EnsureSuffix(s3Dir, "/")
	return c.DownloadPrefixFromS3(bucket, prefix, localDirPath, shouldTrimDirPrefix, maxFiles)
}

// if shouldTrimDirPrefix is true, the directory path of prefix will be trimmed when downloading files
//
// example:
// s3: [test/dir/1.txt, test/dir2/2.txt, test/directions.txt]
// localDirPath: ~/downloads
//
// shouldTrimDirPrefix = true
//   prefix: "test/dir"
//   result: [~/downloads/dir/1.txt, ~/downloads/dir2/1.txt, ~/downloads/directions.txt]
//
//   prefix: "test/dir/"
//   result: [~/downloads/1.txt]
//
// shouldTrimDirPrefix = false
//   prefix: "test/dir"
//   result: [~/downloads/test/dir/1.txt, ~/downloads/test/dir2/1.txt, ~/downloads/test/directions.txt]
//
//   prefix: "test/dir/"
//   result: [~/downloads/test/dir/1.txt]
func (c *Client) DownloadPrefixFromS3(bucket string, prefix string, localDirPath string, shouldTrimDirPrefix bool, maxFiles *int64) error {
	if _, err := files.CreateDirIfMissing(localDirPath); err != nil {
		return err
	}
	createdDirs := strset.New(localDirPath)

	var trimPrefix string
	if shouldTrimDirPrefix {
		lastIndex := strings.LastIndex(prefix, "/")
		if lastIndex == -1 {
			trimPrefix = ""
		} else {
			trimPrefix = prefix[:lastIndex+1]
		}
	}

	err := c.S3Iterator(bucket, prefix, true, maxFiles, func(object *s3.Object) (bool, error) {
		localRelPath := *object.Key
		if shouldTrimDirPrefix {
			localRelPath = strings.TrimPrefix(localRelPath, trimPrefix)
		}

		localPath := filepath.Join(localDirPath, localRelPath)

		// check for directory objects
		if strings.HasSuffix(*object.Key, "/") {
			if !createdDirs.Has(localPath) {
				if _, err := files.CreateDirIfMissing(localPath); err != nil {
					return false, err
				}
				createdDirs.Add(localPath)
			}
			return true, nil
		}

		localDir := filepath.Dir(localPath)
		if !createdDirs.Has(localDir) {
			if _, err := files.CreateDirIfMissing(localDir); err != nil {
				return false, err
			}
			createdDirs.Add(localDir)
		}

		if err := c.DownloadFileFromS3(bucket, *object.Key, localPath); err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) S3FileIterator(bucket string, s3Obj *s3.Object, partSize int, fn func(buffer io.ReadCloser, isLastPart bool) (bool, error)) error {
	size := int(*s3Obj.Size)

	iters := size / partSize
	if size%partSize != 0 {
		iters++
	}

	for i := 0; i < iters; i++ {
		min := i * (partSize)
		max := (i + 1) * (partSize)
		if max > size {
			max = size
		}
		max--

		byteRange := fmt.Sprintf("bytes=%d-%d", min, max)
		obj, err := c.S3().GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    s3Obj.Key,
			Range:  aws.String(byteRange), // use range instead of part numbers because only files uploaded using multipart have parts
		})
		if err != nil {
			return errors.Wrap(err, S3Path(bucket, *s3Obj.Key), "range "+byteRange)
		}

		isLastChunk := i+1 == iters
		shouldContinue, err := fn(obj.Body, isLastChunk)
		if err != nil {
			return errors.Wrap(err, S3Path(bucket, *s3Obj.Key))
		}

		if !shouldContinue {
			break
		}
	}

	return nil
}

func (c *Client) ListS3Dir(bucket string, s3Dir string, includeDirObjects bool, maxResults *int64) ([]*s3.Object, error) {
	prefix := s.EnsureSuffix(s3Dir, "/")
	return c.ListS3Prefix(bucket, prefix, includeDirObjects, maxResults)
}

func (c *Client) ListS3PathDir(s3DirPath string, includeDirObjects bool, maxResults *int64) ([]*s3.Object, error) {
	s3Path := s.EnsureSuffix(s3DirPath, "/")
	return c.ListS3PathPrefix(s3Path, includeDirObjects, maxResults)
}

// This behaves like you'd expect `ls` to behave on a local file system
// "directory" names will be returned even if S3 directory objects don't exist
func (c *Client) ListS3DirOneLevel(bucket string, s3Dir string, maxResults *int64) ([]string, error) {
	s3Dir = s.EnsureSuffix(s3Dir, "/")

	allNames := strset.New()

	err := c.S3Iterator(bucket, s3Dir, true, nil, func(object *s3.Object) (bool, error) {
		relativePath := strings.TrimPrefix(*object.Key, s3Dir)
		oneLevelPath := strings.Split(relativePath, "/")[0]
		allNames.Add(oneLevelPath)

		if maxResults != nil && int64(len(allNames)) >= *maxResults {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, S3Path(bucket, s3Dir))
	}

	return allNames.SliceSorted(), nil
}

func (c *Client) ListS3Prefix(bucket string, prefix string, includeDirObjects bool, maxResults *int64) ([]*s3.Object, error) {
	var allObjects []*s3.Object

	err := c.S3BatchIterator(bucket, prefix, includeDirObjects, maxResults, func(objects []*s3.Object) (bool, error) {
		allObjects = append(allObjects, objects...)
		return true, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, S3Path(bucket, prefix))
	}

	return allObjects, nil
}

func (c *Client) ListS3PathPrefix(s3Path string, includeDirObjects bool, maxResults *int64) ([]*s3.Object, error) {
	bucket, prefix, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}
	return c.ListS3Prefix(bucket, prefix, includeDirObjects, maxResults)
}

func (c *Client) DeleteS3File(bucket string, key string) error {
	_, err := c.S3().DeleteObject(
		&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)

	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) DeleteS3Dir(bucket string, s3Dir string, continueIfFailure bool) error {
	prefix := s.EnsureSuffix(s3Dir, "/")
	return c.DeleteS3Prefix(bucket, prefix, continueIfFailure)
}

func (c *Client) DeleteS3Prefix(bucket string, prefix string, continueIfFailure bool) error {
	err := c.S3BatchIterator(bucket, prefix, true, nil, func(objects []*s3.Object) (bool, error) {
		deleteObjects := make([]*s3.ObjectIdentifier, len(objects))
		for i, object := range objects {
			deleteObjects[i] = &s3.ObjectIdentifier{Key: object.Key}
		}

		_, err := c.S3().DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{
				Objects: deleteObjects,
				Quiet:   aws.Bool(true),
			},
		})

		if err != nil {
			err := errors.Wrap(err, S3Path(bucket, prefix))
			if !continueIfFailure {
				return false, err
			}
			return true, err
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) HashS3Dir(bucket string, prefix string, maxResults *int64) (string, error) {
	md5Hash := md5.New()

	err := c.S3BatchIterator(bucket, prefix, true, maxResults, func(objects []*s3.Object) (bool, error) {
		var subErr error
		for _, object := range objects {
			io.WriteString(md5Hash, *object.ETag)
		}
		return true, subErr
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString((md5Hash.Sum(nil))), nil
}

// Directory objects are empty objects ending in "/". They are not guaranteed to exists, and there may or may not be files "in" the directory
func (c *Client) S3Iterator(bucket string, prefix string, includeDirObjects bool, maxResults *int64, fn func(*s3.Object) (bool, error)) error {
	err := c.S3BatchIterator(bucket, prefix, includeDirObjects, maxResults, func(objects []*s3.Object) (bool, error) {
		var subErr error
		for _, object := range objects {
			shouldContinue, newSubErr := fn(object)
			if newSubErr != nil {
				subErr = newSubErr
			}
			if !shouldContinue {
				return false, subErr
			}
		}
		return true, subErr
	})

	if err != nil {
		return err
	}

	return nil
}

// The return value of fn([]*s3.Object) (bool, error) should be whether to continue iterating, and an error (if any occurred)
// Directory objects are empty objects ending in "/". They are not guaranteed to exists, and there may or may not be files "in" the directory
func (c *Client) S3BatchIterator(bucket string, prefix string, includeDirObjects bool, maxResults *int64, fn func([]*s3.Object) (bool, error)) error {
	var maxResultsRemaining *int64
	if maxResults != nil {
		maxResultsRemaining = pointer.Int64(*maxResults)
	}

	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: maxResultsRemaining,
	}

	var numSeen int64
	var subErr error

	err := c.S3().ListObjectsV2Pages(listObjectsInput,
		func(listObjectsOutput *s3.ListObjectsV2Output, lastPage bool) bool {
			objects := listObjectsOutput.Contents

			// filter directory objects (https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating)
			if !includeDirObjects {
				filtered := objects[:0]
				for _, object := range objects {
					if !strings.HasSuffix(*object.Key, "/") {
						filtered = append(filtered, object)
					}
				}
				objects = filtered
			}

			if len(objects) == 0 {
				return true
			}

			shouldContinue, newSubErr := fn(objects)
			if newSubErr != nil {
				subErr = newSubErr
			}
			if !shouldContinue {
				return false
			}

			numSeen += int64(len(objects))

			if maxResults != nil {
				if numSeen >= *maxResults {
					return false
				}
				*maxResultsRemaining = *maxResults - numSeen
			}

			return true
		})

	if subErr != nil {
		return subErr
	}
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) TagBucket(bucket string, tagMap map[string]string) error {
	var tagSet []*s3.Tag
	for key, value := range tagMap {
		tagSet = append(tagSet, &s3.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err := c.S3().PutBucketTagging(
		&s3.PutBucketTaggingInput{
			Bucket: aws.String(bucket),
			Tagging: &s3.Tagging{
				TagSet: tagSet,
			},
		},
	)

	if err != nil {
		return errors.Wrap(err, "failed to add tags to bucket", bucket)
	}

	return nil
}
