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

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
)

type Storage struct {
	Bucket *blob.Bucket
}

func (c *Storage) FileExists(key string) (bool, error) {
	ctx := context.Background()
	exists, err := c.Bucket.Exists(ctx, key)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return exists, nil
}

func (c *Storage) PutBytes(data []byte, key string) error {
	ctx := context.Background()
	w, err := c.Bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	defer w.Close()

	if _, err = w.Write(data); err != nil {
		return errors.Wrap(err, key)
	}
	return nil
}

func (c *Storage) PutFile(filePath string, key string) error {
	data, err := files.ReadFileBytes(filePath)
	if err != nil {
		return err
	}
	return c.PutBytes(data, key)
}

func (c *Storage) PutBuffer(buffer *bytes.Buffer, key string) error {
	return c.PutBytes(buffer.Bytes(), key)
}

func (c *Storage) PutString(str string, key string) error {
	return c.PutBytes([]byte(str), key)
}

func (c *Storage) PutJSON(obj interface{}, key string) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, key)
	}
	return c.PutBytes(jsonBytes, key)
}

func (c *Storage) GetJSON(objPtr interface{}, key string) error {
	jsonBytes, err := c.GetBytes(key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonBytes, objPtr); err != nil {
		return errors.Wrap(err, key)
	}
	return nil
}

func (c *Storage) PutMsgpack(obj interface{}, key string) error {
	msgpackBytes, err := msgpack.Marshal(obj)
	if err != nil {
		return err
	}
	return c.PutBytes(msgpackBytes, key)
}

func (c *Storage) GetBytes(key string) ([]byte, error) {
	ctx := context.Background()
	bytes, err := c.Bucket.ReadAll(ctx, key)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (c *Storage) GetMsgpack(objPtr interface{}, key string) error {
	msgpackBytes, err := c.GetBytes(key)
	if err != nil {
		return err
	}
	if err := msgpack.Unmarshal(msgpackBytes, objPtr); err != nil {
		return errors.Wrap(err, key)
	}
	return nil
}

func (c *Storage) GetString(key string) (string, error) {
	bytes, err := c.GetBytes(key)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Storage) DeleteByPrefix(prefix string, continueIfFailure bool) error {
	ctx := context.Background()

	blobIter := c.Bucket.List(&blob.ListOptions{Prefix: prefix})
	if blobIter == nil {
		return ErrorFailedToListBlobs()
	}

	for {
		blob, err := blobIter.Next(ctx)

		if blob == nil {
			return nil
		}

		if err != nil {
			return errors.Wrap(err, "blob iteration failed", blob.Key)
		}

		err = c.Bucket.Delete(ctx, blob.Key)
		if err != nil {
			return errors.Wrap(err, "delete failed", blob.Key)
		}
	}
}
