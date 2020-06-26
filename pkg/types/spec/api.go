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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type API struct {
	*userconfig.API
	ID               string             `json:"id"`
	Key              string             `json:"key"`
	DeploymentID     string             `json:"deployment_id"`
	LastUpdated      int64              `json:"last_updated"`
	MetadataRoot     string             `json:"metadata_root"`
	ProjectID        string             `json:"project_id"`
	ProjectKey       string             `json:"project_key"`
	LocalModelCaches []*LocalModelCache `json:"local_model_cache"` // local only
	LocalProjectDir  string             `json:"local_project_dir"`
}

type LocalModelCache struct {
	ID         string `json:"id"`
	HostPath   string `json:"host_path"`
	TargetPath string `json:"target_path"`
}

func GetAPISpec(apiConfig *userconfig.API, projectID string, deploymentID string) *API {
	var buf bytes.Buffer
	buf.WriteString(apiConfig.Name)
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

func (api *API) ModelIDs() []string {
	models := []string{}
	if api != nil && len(api.LocalModelCaches) > 0 {
		for _, localModelCache := range api.LocalModelCaches {
			models = append(models, localModelCache.ID)
		}
	}

	return models
}

func (api *API) ModelNames() []string {
	names := []string{}
	if api != nil && len(api.Predictor.Models) > 0 {
		for _, model := range api.Predictor.Models {
			names = append(names, model.Name)
		}
	}

	return names
}

func (api *API) SubtractModelIDs(apis ...*API) []string {
	modelIDs := strset.FromSlice(api.ModelIDs())
	for _, a := range apis {
		modelIDs.Remove(a.ModelIDs()...)
	}
	return modelIDs.Slice()
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
