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

package gcp

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"google.golang.org/api/iterator"
)

func GCSPath(bucket string, key string) string {
	return "gs://" + filepath.Join(bucket, key)
}

func SplitGCSPath(gcsPath string) (string, string, error) {
	if !IsValidGCSPath(gcsPath) {
		return "", "", ErrorInvalidGCSPath(gcsPath)
	}
	fullPath := gcsPath[len("s3://"):]
	slashIndex := strings.Index(fullPath, "/")
	if slashIndex == -1 {
		return fullPath, "", nil
	}
	bucket := fullPath[0:slashIndex]
	key := fullPath[slashIndex+1:]

	return bucket, key, nil
}

func IsValidGCSPath(gcsPath string) bool {
	if !strings.HasPrefix(gcsPath, "gs://") {
		return false
	}
	parts := strings.Split(gcsPath[5:], "/")
	if len(parts) == 0 {
		return false
	}
	if parts[0] == "" {
		return false
	}
	return true
}

func (c *Client) DoesBucketExist(bucket string) (bool, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return false, err
	}

	_, err = gcsClient.Bucket(bucket).Attrs(context.Background())
	if err != nil {
		if IsBucketDoesNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "bucket", bucket)
	}
	return true, nil
}

func (c *Client) CreateBucket(bucket string, location string, ignoreErrorIfBucketExists bool) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	err = gcsClient.Bucket(bucket).Create(context.Background(), c.ProjectID, &storage.BucketAttrs{
		Location: location,
	})
	if err != nil {
		if IsBucketAlreadyExistsError(err) {
			if !ignoreErrorIfBucketExists {
				return errors.WithStack(err)
			}
		} else {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (c *Client) DeleteBucket(bucket string) error {
	err := c.DeleteGCSPrefix(bucket, "", false)
	if err != nil {
		return err
	}
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	err = gcsClient.Bucket(bucket).Delete(context.Background())
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) IsGCSFile(bucket string, key string) (bool, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return false, err
	}
	_, err = gcsClient.Bucket(bucket).Object(key).Attrs(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) UploadStringToGCS(str string, bucket string, key string) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	objectWriter := gcsClient.Bucket(bucket).Object(key).NewWriter(context.Background())
	objectWriter.ContentType = "text/plain"
	defer objectWriter.Close()
	if _, err := objectWriter.Write([]byte(str)); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) ReadJSONFromGCS(objPtr interface{}, bucket string, key string) error {
	jsonBytes, err := c.ReadBytesFromGCS(bucket, key)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, objPtr)
	if err != nil {
		return errors.Wrap(err, GCSPath(bucket, key))
	}
	return nil
}

func (c *Client) UploadJSONToGCS(obj interface{}, bucket string, key string) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.UploadBytesToGCS(jsonBytes, bucket, key)
}

func (c *Client) DeleteGCSFile(bucket string, key string) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	if err := gcsClient.Bucket(bucket).Object(key).Delete(context.Background()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) DeleteGCSDir(bucket string, gcsDir string, continueIfFailure bool) error {
	prefix := s.EnsureSuffix(gcsDir, "/")
	return c.DeleteGCSPrefix(bucket, prefix, continueIfFailure)
}

func (c *Client) DeleteGCSPrefix(bucket string, gcsDir string, continueIfFailure bool) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}

	bkt := gcsClient.Bucket(bucket)
	objectIterator := bkt.Objects(context.Background(), &storage.Query{
		Prefix: gcsDir,
	})

	var subErr error
	for {
		attrs, err := objectIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return errors.WithStack(err)
		}
		err = bkt.Object(attrs.Name).Delete(context.Background())
		if err != nil {
			subErr = errors.WithStack(err)
			if !continueIfFailure {
				break
			}
		}
	}

	if subErr != nil {
		return subErr
	}
	return nil
}

func (c *Client) UploadBytesToGCS(data []byte, bucket string, key string) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	objectWriter := gcsClient.Bucket(bucket).Object(key).NewWriter(context.Background())
	defer objectWriter.Close()
	if _, err := objectWriter.Write(data); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) ReadBytesFromGCS(bucket string, key string) ([]byte, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return nil, err
	}
	objectReader, err := gcsClient.Bucket(bucket).Object(key).NewReader(context.Background())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer objectReader.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(objectReader); err != nil {
		return nil, errors.WithStack(err)
	}
	return buf.Bytes(), nil
}

func ConvertGCSObjectsToKeys(gcsObjects ...*storage.ObjectAttrs) []string {
	paths := make([]string, 0, len(gcsObjects))
	for _, object := range gcsObjects {
		if object != nil {
			paths = append(paths, object.Name)
		}
	}
	return paths
}

func (c *Client) ListGCSDir(bucket string, gcsDir string, includeDirObjects bool, maxResults *int64) ([]*storage.ObjectAttrs, error) {
	gcsDir = s.EnsureSuffix(gcsDir, "/")
	return c.ListGCSPrefix(bucket, gcsDir, includeDirObjects, maxResults)
}

func (c *Client) ListGCSPrefix(bucket string, prefix string, includeDirObjects bool, maxResults *int64) ([]*storage.ObjectAttrs, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return nil, err
	}

	objectIterator := gcsClient.Bucket(bucket).Objects(context.Background(), &storage.Query{
		Prefix: prefix,
	})

	var gcsObjects []*storage.ObjectAttrs
	for {
		attrs, err := objectIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, errors.WithStack(err)
		}
		if attrs == nil {
			continue
		}

		if includeDirObjects || !strings.HasSuffix(attrs.Name, "/") {
			gcsObjects = append(gcsObjects, attrs)
		}

		if maxResults != nil && int64(len(gcsObjects)) >= *maxResults {
			break
		}
	}

	return gcsObjects, nil
}

func (c *Client) ListGCSPathDir(gcsDirPath string, includeDirObjects bool, maxResults *int64) ([]*storage.ObjectAttrs, error) {
	bucket, gcsDir, err := SplitGCSPath(gcsDirPath)
	if err != nil {
		return nil, err
	}
	return c.ListGCSDir(bucket, gcsDir, includeDirObjects, maxResults)
}

// This behaves like you'd expect `ls` to behave on a local file system
// "directory" names will be returned even if S3 directory objects don't exist
func (c *Client) ListGCSDirOneLevel(bucket string, gcsDir string, maxResults *int64) ([]string, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return nil, err
	}
	gcsDir = s.EnsureSuffix(gcsDir, "/")

	objectIterator := gcsClient.Bucket(bucket).Objects(context.Background(), &storage.Query{
		Prefix: gcsDir,
	})

	allNames := strset.New()

	for {
		attrs, err := objectIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, errors.WithStack(err)
		}

		relativePath := strings.TrimPrefix(attrs.Name, gcsDir)
		oneLevelPath := strings.Split(relativePath, "/")[0]
		allNames.Add(oneLevelPath)

		if maxResults != nil && int64(len(allNames)) >= *maxResults {
			break
		}
	}

	return allNames.SliceSorted(), nil
}
