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
	"net/url"
)

func IsValidLocalPath(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return len(pathURL.Scheme) == 0 && len(pathURL.Host) == 0 && len(pathURL.Path) > 0, nil
}

func IsValidS3aPath(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return pathURL.Scheme == "s3a" && len(pathURL.Host) > 0 && len(pathURL.Path) > 0, nil
}

func IsValidS3Path(path string) (bool, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return false, err
	}

	return pathURL.Scheme == "s3" && len(pathURL.Host) > 0 && len(pathURL.Path) > 0, nil
}

func IsValidAWSPath(path string) (bool, error) {
	isS3, err := IsValidS3Path(path)
	if err != nil {
		return false, err
	}

	isS3a, err := IsValidS3aPath(path)
	if err != nil {
		return false, err
	}

	return isS3 || isS3a, nil
}

func ProviderTypeFromPath(path string) (ProviderType, error) {
	isLocal, err := IsValidLocalPath(path)
	if err != nil {
		return UnknownProviderType, err
	}

	if isLocal {
		return LocalProviderType, nil
	}

	isAWS, err := IsValidAWSPath(path)

	if isAWS {
		return AWSProviderType, nil
	}

	return UnknownProviderType, ErrorUnsupportedFilePath(path, "s3", "s3a", "local")
}
