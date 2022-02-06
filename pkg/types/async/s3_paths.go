/*
Copyright 2022 Cortex Labs, Inc.

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

package async

import (
	"fmt"
)

func StoragePath(clusterUID, apiName string) string {
	return fmt.Sprintf("%s/workloads/%s", clusterUID, apiName)
}

func PayloadPath(storagePath string, requestID string) string {
	return fmt.Sprintf("%s/%s/payload", storagePath, requestID)
}

func HeadersPath(storagePath string, requestID string) string {
	return fmt.Sprintf("%s/%s/headers.json", storagePath, requestID)
}

func ResultPath(storagePath string, requestID string) string {
	return fmt.Sprintf("%s/%s/result.json", storagePath, requestID)
}

func StatusPrefixPath(storagePath string, requestID string) string {
	return fmt.Sprintf("%s/%s/status", storagePath, requestID)
}

func StatusPath(storagePath string, requestID string, status Status) string {
	return fmt.Sprintf("%s/%s", StatusPrefixPath(storagePath, requestID), status)
}
