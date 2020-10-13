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
	"fmt"
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
	ID                    string                 `json:"id"`
	SpecID                string                 `json:"spec_id"`
	PredictorID           string                 `json:"predictor_id"`
	DeploymentID          string                 `json:"deployment_id"`
	Key                   string                 `json:"key"`
	PredictorKey          string                 `json:"predictor_key"`
	LastUpdated           int64                  `json:"last_updated"`
	MetadataRoot          string                 `json:"metadata_root"`
	ProjectID             string                 `json:"project_id"`
	ProjectKey            string                 `json:"project_key"`
	CuratedModelResources []CuratedModelResource `json:"curated_model_resources"`
	LocalModelCaches      []*LocalModelCache     `json:"local_model_cache"` // local only
	LocalProjectDir       string                 `json:"local_project_dir"`
}

type LocalModelCache struct {
	ID         string `json:"id"`
	HostPath   string `json:"host_path"`
	TargetPath string `json:"target_path"`
}

type CuratedModelResource struct {
	*userconfig.ModelResource
	S3Path   bool    `json:"s3_path"`
	Versions []int64 `json:"versions"`
}

func GetAPISpec(apiConfig *userconfig.API, models []CuratedModelResource, projectID string, deploymentID string, clusterName string) *API {
	/*
		APIID (uniquely identifies an api configuration for a given deployment)
			* SpecID (uniquely identifies api configuration specified by user)
				* PredictorID (used to determine when rolling updates need to happen)
					* Resource
					* Predictor
					* Monitoring
					* Compute
					* ProjectID
				* Deployment Strategy
				* Autoscaling
				* Networking
				* APIs
			* DeploymentID (used for refreshing a deployment)
	*/
	var buf bytes.Buffer

	buf.WriteString(s.Obj(apiConfig.Resource))
	buf.WriteString(s.Obj(apiConfig.Predictor))
	buf.WriteString(s.Obj(apiConfig.Monitoring))
	buf.WriteString(projectID)
	if apiConfig.Compute != nil {
		buf.WriteString(s.Obj(apiConfig.Compute.Normalized()))
	}
	predictorID := hash.Bytes(buf.Bytes())

	buf.Reset()
	buf.WriteString(predictorID)
	buf.WriteString(s.Obj(apiConfig.APIs))
	buf.WriteString(s.Obj(apiConfig.Networking))
	buf.WriteString(s.Obj(apiConfig.Autoscaling))
	buf.WriteString(s.Obj(apiConfig.UpdateStrategy))
	specID := hash.Bytes(buf.Bytes())[:32]

	apiID := fmt.Sprintf("%s-%s-%s", MonotonicallyDecreasingID(), deploymentID, specID) // should be up to 60 characters long

	return &API{
		API:                   apiConfig,
		CuratedModelResources: models,
		ID:                    apiID,
		SpecID:                specID,
		PredictorID:           predictorID,
		Key:                   Key(apiConfig.Name, apiID, clusterName),
		PredictorKey:          PredictorKey(apiConfig.Name, predictorID, clusterName),
		DeploymentID:          deploymentID,
		LastUpdated:           time.Now().Unix(),
		MetadataRoot:          MetadataRoot(apiConfig.Name, clusterName),
		ProjectID:             projectID,
		ProjectKey:            ProjectKey(projectID, clusterName),
	}
}

func NumLocalModels(models []CuratedModelResource) int {
	numLocalModels := 0
	if len(models) > 0 {
		for _, model := range models {
			if model.S3Path {
				continue
			}
			if len(model.Versions) > 0 {
				numLocalModels += len(model.Versions)
			} else {
				numLocalModels++
			}
		}
	}

	return numLocalModels
}

func NumModels(models []CuratedModelResource) int {
	numModels := 0
	if len(models) > 0 {
		for _, model := range models {
			if len(model.Versions) > 0 {
				numModels += len(model.Versions)
			} else {
				numModels++
			}
		}
	}

	return numModels
}

func (api *API) NumLocalModels() int {
	numLocalModels := 0
	if api != nil {
		numLocalModels = NumLocalModels(api.CuratedModelResources)
	}
	return numLocalModels
}

func (api *API) NumModels() int {
	numModels := 0
	if api != nil {
		numModels = NumModels(api.CuratedModelResources)
	}
	return numModels
}

// Keep track of models in the model cache used by this API (local only)
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
	if api != nil && len(api.CuratedModelResources) > 0 {
		for _, model := range api.CuratedModelResources {
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

func PredictorKey(apiName string, predictorID string, clusterName string) string {
	return filepath.Join(
		clusterName,
		"apis",
		apiName,
		"predictor",
		predictorID,
		consts.CortexVersion+"-spec.json",
	)
}

func Key(apiName string, apiID string, clusterName string) string {
	return filepath.Join(
		clusterName,
		"apis",
		apiName,
		"api",
		apiID,
		consts.CortexVersion+"-spec.json",
	)
}

func (api API) RawAPIKey(clusterName string) string {
	return filepath.Join(
		clusterName,
		"apis",
		api.Name,
		"raw_api",
		api.ID,
		consts.CortexVersion+"-cortex.yaml",
	)
}

func MetadataRoot(apiName string, clusterName string) string {
	return filepath.Join(
		clusterName,
		"apis",
		apiName,
		"metadata",
	)
}

func ProjectKey(projectID string, clusterName string) string {
	return filepath.Join(
		clusterName,
		"projects",
		projectID+".zip",
	)
}
