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
	"bytes"

	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type API struct {
	*userconfig.API
	ID                string                        `json:"id"`
	Key               string                        `json:"key"`
	WorkloadID        string                        `json:"workload_id"`
	CreatedEpoch      int64                         `json:"created_epoch"`
	ClusterConfig     *clusterconfig.InternalConfig `json:"cluster_config"`
	DeploymentVersion string                        `json:"deployment_version"`
	Root              string                        `json:"root"`
	MetadataRoot      string                        `json:"metadata_root"`
	StatusPrefix      string                        `json:"status_prefix"`
	ProjectID         string                        `json:"project_id"`
	ProjectKey        string                        `json:"project_key"`
}

func (api *API) LogGroupName() string {
	return api.ClusterConfig.LogGroup + "." + api.Name
}

func (api *API) Validate() error {
	return nil
}

func apiSpec(apiConfig *userconfig.API, deploymentVersion string, projectID string) *API {
	var buf bytes.Buffer
	buf.WriteString(apiConfig.Name)
	buf.WriteString(*apiConfig.Endpoint)
	buf.WriteString(s.Obj(apiConfig.Tracker))
	buf.WriteString(deploymentVersion)
	buf.WriteString(s.Obj(apiConfig.Predictor))
	buf.WriteString(projectID)
	id := hash.Bytes(buf.Bytes())

	return &spec.API{
		API: apiConfig,
		ID:           id,
		ResourceType: resource.APIType,
	}, nil
}
