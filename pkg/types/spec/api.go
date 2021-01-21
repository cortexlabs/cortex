/*
Copyright 2021 Cortex Labs, Inc.

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
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type API struct {
	*userconfig.API
	ID           string `json:"id"`
	SpecID       string `json:"spec_id"`
	PredictorID  string `json:"predictor_id"`
	DeploymentID string `json:"deployment_id"`
	Key          string `json:"key"`
	PredictorKey string `json:"predictor_key"`
	LastUpdated  int64  `json:"last_updated"`
	MetadataRoot string `json:"metadata_root"`
	ProjectID    string `json:"project_id"`
	ProjectKey   string `json:"project_key"`
}

type CuratedModelResource struct {
	*userconfig.ModelResource
	S3Path  bool `json:"s3_path"`
	GCSPath bool `json:"gcs_path"`

	// has no utility in the go stack, but in the python stack, this is required for
	// single model paths (ONNX) because models are made available locally to the api pod
	LocalPath bool `json:"local_path"`

	IsFilePath bool    `json:"file_path"`
	Versions   []int64 `json:"versions"`
}

/*
APIID (uniquely identifies an api configuration for a given deployment)
	* SpecID (uniquely identifies api configuration specified by user)
		* PredictorID (used to determine when rolling updates need to happen)
			* Resource
			* Predictor
			* TaskDefinition
			* Compute
			* ProjectID
		* Deployment Strategy
		* Autoscaling
		* Networking
		* APIs
	* DeploymentID (used for refreshing a deployment)
*/
func GetAPISpec(apiConfig *userconfig.API, projectID string, deploymentID string, clusterName string) *API {
	var buf bytes.Buffer

	buf.WriteString(s.Obj(apiConfig.Resource))
	buf.WriteString(s.Obj(apiConfig.Predictor))
	buf.WriteString(s.Obj(apiConfig.TaskDefinition))
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
		API:          apiConfig,
		ID:           apiID,
		SpecID:       specID,
		PredictorID:  predictorID,
		Key:          Key(apiConfig.Name, apiID, clusterName),
		PredictorKey: PredictorKey(apiConfig.Name, predictorID, clusterName),
		DeploymentID: deploymentID,
		LastUpdated:  time.Now().Unix(),
		MetadataRoot: MetadataRoot(apiConfig.Name, clusterName),
		ProjectID:    projectID,
		ProjectKey:   ProjectKey(projectID, clusterName),
	}
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

// The path to the directory which contains one subdirectory for each API ID (for its API spec)
func KeysPrefix(apiName string, clusterName string) string {
	return filepath.Join(
		clusterName,
		"apis",
		apiName,
		"api",
	) + "/"
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

// Extract the timestamp from an API ID
func TimeFromAPIID(apiID string) (time.Time, error) {
	timeIDStr := strings.Split(apiID, "-")[0]
	timeID, err := strconv.ParseInt(timeIDStr, 16, 64)
	if err != nil {
		return time.Time{}, errors.Wrap(err, fmt.Sprintf("unable to parse API timestamp (%s)", timeIDStr))
	}
	timeNanos := math.MaxInt64 - timeID
	return time.Unix(0, timeNanos), nil
}
