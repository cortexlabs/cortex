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
	HandlerID    string `json:"handler_id"`
	DeploymentID string `json:"deployment_id"`
	Key          string `json:"key"`
	HandlerKey   string `json:"handler_key"`
	LastUpdated  int64  `json:"last_updated"`
	MetadataRoot string `json:"metadata_root"`
	ProjectID    string `json:"project_id"`
}

/*
APIID (uniquely identifies an api configuration for a given deployment)
	* SpecID (uniquely identifies api configuration specified by user)
		* HandlerID (used to determine when rolling updates need to happen)
			* Resource
			* Handler
			* TaskDefinition
			* Compute
			* ProjectID
		* Deployment Strategy
		* Autoscaling
		* Networking
		* APIs
	* DeploymentID (used for refreshing a deployment)
*/
func GetAPISpec(apiConfig *userconfig.API, projectID string, deploymentID string, clusterUID string) *API {
	var buf bytes.Buffer

	buf.WriteString(s.Obj(apiConfig.Resource))
	buf.WriteString(s.Obj(apiConfig.Pod))
	buf.WriteString(projectID)
	handlerID := hash.Bytes(buf.Bytes())

	buf.Reset()
	buf.WriteString(handlerID)
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
		HandlerID:    handlerID,
		Key:          Key(apiConfig.Name, apiID, clusterUID),
		HandlerKey:   HandlerKey(apiConfig.Name, handlerID, clusterUID),
		DeploymentID: deploymentID,
		LastUpdated:  time.Now().Unix(),
		MetadataRoot: MetadataRoot(apiConfig.Name, clusterUID),
		ProjectID:    projectID,
	}
}

func HandlerKey(apiName string, handlerID string, clusterUID string) string {
	return filepath.Join(
		clusterUID,
		"apis",
		apiName,
		"handler",
		handlerID,
		consts.CortexVersion+"-spec.json",
	)
}

func Key(apiName string, apiID string, clusterUID string) string {
	return filepath.Join(
		clusterUID,
		"apis",
		apiName,
		"api",
		apiID,
		consts.CortexVersion+"-spec.json",
	)
}

// The path to the directory which contains one subdirectory for each API ID (for its API spec)
func KeysPrefix(apiName string, clusterUID string) string {
	return filepath.Join(
		clusterUID,
		"apis",
		apiName,
		"api",
	) + "/"
}

func MetadataRoot(apiName string, clusterUID string) string {
	return filepath.Join(
		clusterUID,
		"apis",
		apiName,
		"metadata",
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
