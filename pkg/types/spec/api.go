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

package spec

import (
	"bytes"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type API struct {
	*userconfig.API
	ID              string           `json:"id"`
	Key             string           `json:"key"`
	DeploymentID    string           `json:"deployment_id"`
	LastUpdated     int64            `json:"last_updated"`
	MetadataRoot    string           `json:"metadata_root"`
	ProjectID       string           `json:"project_id"`
	ProjectKey      string           `json:"project_key"`
	LocalModelCache *LocalModelCache `json:"local_model_cache"` // local only
	LocalProjectDir string           `json:"local_project_dir"`
}

type LocalModelCache struct {
	ID       string `json:"id"`
	HostPath string `json:"host_path"`
}

func GetAPISpec(apiConfig *userconfig.API, projectID string, deploymentID string) *API {
	var buf bytes.Buffer
	buf.WriteString(apiConfig.Name)
	if apiConfig.Endpoint != nil {
		buf.WriteString(*apiConfig.Endpoint)
	}
	if apiConfig.LocalPort != nil {
		buf.WriteString(s.Obj(*apiConfig.LocalPort))
	}
	buf.WriteString(s.Obj(apiConfig.Predictor))
	buf.WriteString(s.Obj(apiConfig.Monitoring))
	buf.WriteString(deploymentID)
	buf.WriteString(projectID)
	id := hash.Bytes(buf.Bytes())

	return &API{
		API:          apiConfig,
		ID:           id,
		Key:          Key(apiConfig.Name, id),
		DeploymentID: deploymentID,
		LastUpdated:  time.Now().Unix(),
		MetadataRoot: MetadataRoot(apiConfig.Name),
		ProjectID:    projectID,
		ProjectKey:   ProjectKey(projectID),
	}
}

func Key(apiName string, apiID string) string {
	return filepath.Join(
		"apis",
		apiName,
		apiID,
		consts.CortexVersion+"-spec.msgpack",
	)
}

func MetadataRoot(apiName string) string {
	return filepath.Join(
		"apis",
		apiName,
		"metadata",
	)
}

func ProjectKey(projectID string) string {
	return filepath.Join(
		"projects",
		projectID+".zip",
	)
}
