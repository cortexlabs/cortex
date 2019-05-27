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
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/s3blob"

	"github.com/cortexlabs/cortex/pkg/consts"
	awsUtil "github.com/cortexlabs/cortex/pkg/lib/cloud/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cloud/local"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func ProviderTypeFromPath(path string) (ProviderType, error) {

	isLocal, err := local.IsValidPath(path)
	if err != nil {
		return UnknownProviderType, err
	}

	if isLocal {
		return LocalProviderType, nil
	}

	isS3, err := awsUtil.IsValidS3(path)
	if err != nil {
		return UnknownProviderType, err
	}

	isS3a, err := awsUtil.IsValidS3a(path)
	if err != nil {
		return UnknownProviderType, err
	}

	if isS3 || isS3a {
		return AWSProviderType, nil
	}
	return UnknownProviderType, ErrorUnrecognizedFilepath()
}

func ExternalPrefixExists(path string) (bool, error) {
	var b *blob.Bucket
	var prefix string
	var err error
	ctx := context.Background()

	cloudType, err := ProviderTypeFromPath(path)
	if err != nil {
		return false, err
	}

	if cloudType == LocalProviderType {
		if filepath.IsAbs(path) {
			return false, local.ErrorNotAnAbsolutePath()
		}

		if strings.HasSuffix(path, "/") {
			path = path[:1]
		}
		dir := filepath.Dir(path)
		prefix = filepath.Base(path)

		bucketURL := "file://" + dir
		b, err = blob.OpenBucket(ctx, bucketURL)
		if err != nil {
			return false, errors.Wrap(err, "unable to open bucket", bucketURL)
		}
		defer b.Close()
	} else if cloudType == AWSProviderType {
		pathURL, err := url.Parse(path)
		if err != nil {
			return false, err
		}

		sess := session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.AnonymousCredentials,
			Region:      aws.String(consts.DefaultAWSRegion),
		}))

		b, err = s3blob.OpenBucket(ctx, sess, pathURL.Host, nil)
		if err != nil {
			return false, errors.Wrap(err, "unable to open s3 bucket", pathURL.Host)
		}
		defer b.Close()

		prefix = pathURL.Path
		if strings.HasPrefix(prefix, "/") {
			prefix = prefix[1:]
		}
	} else {
		return false, ErrorUnrecognizedFilepath()
	}

	blobIter := b.List(&blob.ListOptions{Prefix: prefix})

	if blobIter == nil {
		return false, ErrorFailedToListBlobs()
	}

	obj, err := blobIter.Next(ctx)

	if obj == nil || err != nil {
		return false, err
	}

	return true, nil
}
