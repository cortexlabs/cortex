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

package cloud

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
)

func (c *Client) BucketPath() string {
	return c.OperatorBucketURL().String()
}

func (c *Client) InternalPath(key string) string {
	url := c.WorkloadBucketURL()
	url.Path = filepath.Join(url.Path, key)
	return url.String()
}

func (c *Client) FileExists(key string) (bool, error) {
	ctx := context.Background()
	return c.bucket.Exists(ctx, key)
}

func (c *Client) PutBytes(data []byte, key string) error {
	ctx := context.Background()
	w, err := c.bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return errors.Wrap(err, "unable to open writer")
	}
	defer w.Close()

	_, err = w.Write(data)
	return err
}

func (c *Client) PutFile(filePath string, key string) error {
	data, err := files.ReadFileBytes(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	return c.PutBytes(data, key)
}

func (c *Client) PutBuffer(buffer *bytes.Buffer, key string) error {
	return c.PutBytes(buffer.Bytes(), key)
}

func (c *Client) PutString(str string, key string) error {
	str = strings.TrimSpace(str)
	return c.PutBytes([]byte(str), key)
}

func (c *Client) PutJSON(obj interface{}, key string) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "failed to json marshalling")
	}
	return c.PutBytes(jsonBytes, key)
}

func (c *Client) GetJSON(objPtr interface{}, key string) error {
	jsonBytes, err := c.GetBytes(key)
	if err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(jsonBytes, objPtr), key)
}

func (c *Client) PutMsgpack(obj interface{}, key string) error {
	msgpackBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return err
	}
	return c.PutBytes(msgpackBytes, key)
}

func (c *Client) GetBytes(key string) ([]byte, error) {
	ctx := context.Background()
	return c.bucket.ReadAll(ctx, key)
}

func (c *Client) GetMsgpack(objPtr interface{}, key string) error {
	msgpackBytes, err := c.GetBytes(key)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(msgpackBytes, objPtr)
}

func (c *Client) GetString(key string) (string, error) {
	bytes, err := c.GetBytes(key)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Client) DeleteByPrefix(prefix string, continueIfFailure bool) error {
	ctx := context.Background()

	blobIter := c.bucket.List(&blob.ListOptions{Prefix: prefix})
	if blobIter == nil {
		return ErrorFailedToListBlobs()
	}

	for {
		blob, err := blobIter.Next(ctx)

		if blob == nil {
			return nil
		}

		if err != nil {
			return errors.Wrap(err, "next failed", blob.Key)
		}

		err = c.bucket.Delete(ctx, blob.Key)
		if err != nil {
			return errors.Wrap(err, "delete failed", blob.Key)
		}
	}
}
