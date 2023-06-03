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
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
)

type API struct {
	*userconfig.API
	ID           string `json:"id" yaml:"id"`
	SpecID       string `json:"spec_id" yaml:"spec_id"`
	PodID        string `json:"pod_id" yaml:"pod_id"`
	DeploymentID string `json:"deployment_id" yaml:"deployment_id"`

	Key string `json:"key" yaml:"key"`

	InitialDeploymentTime int64  `json:"initial_deployment_time" yaml:"initial_deployment_time"`
	LastUpdated           int64  `json:"last_updated" yaml:"last_updated"`
	MetadataRoot          string `json:"metadata_root" yaml:"metadata_root"`
}

type Metadata struct {
	*userconfig.Resource
	APIID        string `json:"id" yaml:"id"`
	DeploymentID string `json:"deployment_id,omitempty" yaml:"deployment_id,omitempty"`
	LastUpdated  int64  `json:"last_updated" yaml:"last_updated"`
}

func MetadataFromDeployment(deployment *kapps.Deployment) (*Metadata, error) {
	lastUpdated, err := TimeFromAPIID(deployment.Labels["apiID"])
	if err != nil {
		return nil, err
	}
	return &Metadata{
		Resource: &userconfig.Resource{
			Name: deployment.Labels["apiName"],
			Kind: userconfig.KindFromString(deployment.Labels["apiKind"]),
		},
		APIID:        deployment.Labels["apiID"],
		DeploymentID: deployment.Labels["deploymentID"],
		LastUpdated:  lastUpdated.Unix(),
	}, nil
}

func MetadataFromVirtualService(vs *istioclientnetworking.VirtualService) (*Metadata, error) {
	lastUpdated, err := TimeFromAPIID(vs.Labels["apiID"])
	if err != nil {
		return nil, err
	}
	return &Metadata{
		Resource: &userconfig.Resource{
			Name: vs.Labels["apiName"],
			Kind: userconfig.KindFromString(vs.Labels["apiKind"]),
		},
		APIID:        vs.Labels["apiID"],
		DeploymentID: vs.Labels["deploymentID"],
		LastUpdated:  lastUpdated.Unix(),
	}, nil
}

/*
* ID (uniquely identifies an api configuration for a given deployment)
  - DeploymentID (used for refreshing a deployment)
  - SpecID (uniquely identifies api configuration specified by user)
  - PodID (an ID representing the pod spec)
  - Resource
  - Containers
  - Compute
  - Pod
  - Deployment Strategy
  - Autoscaling
  - Networking
  - APIs

initialDeploymentTime is Time.UnixNano()
*/
func GetAPISpec(apiConfig *userconfig.API, initialDeploymentTime int64, deploymentID string, clusterUID string) *API {
	var buf bytes.Buffer

	buf.WriteString(s.Obj(apiConfig.Resource))
	buf.WriteString(s.Obj(apiConfig.Pod))
	podID := hash.Bytes(buf.Bytes())

	buf.Reset()
	buf.WriteString(podID)
	buf.WriteString(s.Obj(apiConfig.APIs))
	buf.WriteString(s.Obj(apiConfig.Networking))
	buf.WriteString(s.Obj(apiConfig.Autoscaling))
	buf.WriteString(s.Obj(apiConfig.UpdateStrategy))
	buf.WriteString(s.Obj(apiConfig.NodeGroups))
	specID := hash.Bytes(buf.Bytes())[:32]

	apiID := fmt.Sprintf("%s-%s-%s", MonotonicallyDecreasingID(), deploymentID, specID) // should be up to 60 characters long

	return &API{
		API:                   apiConfig,
		ID:                    apiID,
		SpecID:                specID,
		PodID:                 podID,
		Key:                   Key(apiConfig.Name, apiID, clusterUID),
		InitialDeploymentTime: initialDeploymentTime,
		DeploymentID:          deploymentID,
		LastUpdated:           time.Now().Unix(),
		MetadataRoot:          MetadataRoot(apiConfig.Name, clusterUID),
	}
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
