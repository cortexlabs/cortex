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

package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
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

func (c *Client) CreateBucket(bucket, projectID string, ignoreErrorIfBucketExists bool) error {
	gcsClient, err := c.GCS()
	if err != nil {
		return err
	}
	err = gcsClient.Bucket(bucket).Create(context.Background(), projectID, nil)
	if err != nil {
		if DoesBucketAlreadyExistError(err) {
			if !ignoreErrorIfBucketExists {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (c *Client) IsGCSFile(bucket string, key string) (bool, error) {
	gcsClient, err := c.GCS()
	if err != nil {
		return false, err
	}
	_, err = gcsClient.Bucket(bucket).Object(key).Attrs(context.Background())
	if err != nil && err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) ReadJSONFromGCS(objPtr interface{}, bucket string, key string) error {
	jsonBytes, err := c.ReadBytesFromGCS(bucket, key)
	if err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(jsonBytes, objPtr), GCSPath(bucket, key))
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
		return err
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

	var attrs *storage.ObjectAttrs
	for {
		attrs, err = objectIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil && !continueIfFailure {
			break
		}
		err = bkt.Object(attrs.Name).Delete(context.Background())
		if err != nil && !continueIfFailure {
			break
		}
	}

	if err != nil && err != iterator.Done && !continueIfFailure {
		return err
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
		return err
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
		return nil, err
	}
	defer objectReader.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(objectReader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) ListGCSDir(bucket string, gcsDir string, maxResults *int64) ([]string, error) {
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
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		allNames.Add(attrs.Name)
		if maxResults != nil && int64(len(allNames)) >= *maxResults {
			break
		}
	}

	return allNames.SliceSorted(), nil
}

func (c *Client) ListGCSPathDir(gcsDirPath string, maxResults *int64) ([]string, error) {
	bucket, gcsDir, err := SplitGCSPath(gcsDirPath)
	if err != nil {
		return nil, err
	}
	return c.ListGCSDir(bucket, gcsDir, maxResults)
}

func (c *Client) ListGCSDirOneLevel(bucket string, gcsDir string, maxResults *int64) ([]string, error) {
	objects, err := c.ListGCSDir(bucket, gcsDir, maxResults)
	if err != nil {
		return nil, err
	}

	filteredObjects := strset.New()
	for _, object := range objects {
		relativePath := strings.TrimPrefix(object, gcsDir)
		oneLevelPath := slices.UniqueStrings(strings.Split(relativePath, "/"))[0]
		filteredObjects.Add(oneLevelPath)
	}

	return filteredObjects.SliceSorted(), nil
}
