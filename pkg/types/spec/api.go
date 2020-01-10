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

package spec

import (
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type API struct {
	*userconfig.API
	ID           string `json:"id"`
	Key          string `json:"key"`
	DeploymentID string `json:"deployment_id"`
	LastUpdated  int64  `json:"last_updated"`
	MetadataRoot string `json:"metadata_root"`
	ProjectID    string `json:"project_id"`
	ProjectKey   string `json:"project_key"`
}
