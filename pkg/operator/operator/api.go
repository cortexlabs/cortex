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

package operator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func UpdateAPI(
	apiConfig *userconfig.API,
	projectID string,
	refresh bool,
	force bool,
) (*spec.API, string, error) {

	prevDeployment, prevService, prevVirtualService, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}

	deploymentID := k8s.RandomName()
	if prevDeployment != nil && prevDeployment.Labels["deploymentID"] != "" && !refresh {
		deploymentID = prevDeployment.Labels["deploymentID"]
	}

	api := getAPISpec(apiConfig, projectID, deploymentID)

	var msg string

	if prevDeployment == nil {
		err = config.AWS.UploadMsgpackToS3(api, api.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("creating %s api", api.Name)

	} else if !k8s.AreDeploymentsEqual(prevDeployment, deploymentSpec(api, prevDeployment)) {
		if prevDeployment.Status.UpdatedReplicas < api.Compute.MinReplicas && !force { // TODO: UpdatedReplicas may include replicas that are not ready
			return nil, "", errors.New(fmt.Sprintf("%s api is updating (override with --force)", api.Name))
		}
		err = config.AWS.UploadMsgpackToS3(api, api.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("updating %s api", api.Name)

	} else { // deployment didn't change
		if prevDeployment.Status.UpdatedReplicas < api.Compute.MinReplicas { // TODO: UpdatedReplicas may include replicas that are not ready
			msg = fmt.Sprintf("%s api is already updating", api.Name)
		} else {
			msg = fmt.Sprintf("%s api is up to date", api.Name)
		}
	}

	return api, msg, nil
}

func DeleteAPI(apiName string, keepCache bool) (bool, error) {
	wasDeployed, err := config.K8s.DeploymentExists(apiName)
	if err != nil {
		return false, err
	}

	err = parallel.RunFirstErr(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			return deleteS3Resources(apiName)
		},
	)

	if err != nil {
		return false, err
	}

	return wasDeployed, nil
}

func getAPISpec(
	apiConfig *userconfig.API,
	projectID string,
	deploymentID string,
) *spec.API {
	var buf bytes.Buffer
	buf.WriteString(apiConfig.Name)
	buf.WriteString(*apiConfig.Endpoint)
	buf.WriteString(s.Obj(apiConfig.Predictor))
	buf.WriteString(s.Obj(apiConfig.Tracker))
	buf.WriteString(deploymentID)
	buf.WriteString(projectID)
	id := hash.Bytes(buf.Bytes())

	return &spec.API{
		API:          apiConfig,
		ID:           id,
		Key:          specKey(apiConfig.Name, id),
		DeploymentID: deploymentID,
		LastUpdated:  time.Now().Unix(),
		MetadataRoot: metadataRoot(apiConfig.Name, id),
		ProjectID:    projectID,
		ProjectKey:   ProjectKey(projectID),
	}
}

func getK8sResources(apiConfig *userconfig.API) (
	*kapps.Deployment,
	*kcore.Service,
	*kunstructured.Unstructured,
	error,
) {

	var deployment *kapps.Deployment
	var service *kcore.Service
	var virtualService *kunstructured.Unstructured

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(apiConfig.Name)
			return err
		},
		func() error {
			var err error
			service, err = config.K8s.GetService(apiConfig.Name)
			return err
		},
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(apiConfig.Name)
			return err
		},
	)

	return deployment, service, virtualService, err
}

func applyK8sResources(
	api *spec.API,
	prevDeployment *kapps.Deployment,
	prevService *kcore.Service,
	prevVirtualService *kunstructured.Unstructured,
) error {

	newDeployment := deploymentSpec(api, prevDeployment)
	newService := serviceSpec(api)
	newVirtualService := virtualServiceSpec(api)

	return parallel.RunFirstErr(
		func() error {
			if prevDeployment == nil {
				_, err := config.K8s.CreateDeployment(newDeployment)
				return err
			} else {
				// Delete deployment if it never became ready
				if prevDeployment.Status.ReadyReplicas == 0 {
					config.K8s.DeleteDeployment(api.Name)
					_, err := config.K8s.CreateDeployment(newDeployment)
					return err
				}
				_, err := config.K8s.UpdateDeployment(newDeployment)
				return err
			}
		},
		func() error {
			if prevService == nil {
				_, err := config.K8s.CreateService(newService)
				return err
			} else {
				_, err := config.K8s.UpdateService(newService)
				return err
			}
		},
		func() error {
			if prevVirtualService == nil {
				_, err := config.K8s.CreateVirtualService(newVirtualService)
				return err
			} else {
				_, err := config.K8s.UpdateVirtualService(newVirtualService)
				return err
			}
		},
		func() error {
			// Delete HPA while updating replicas to avoid unwanted autoscaling due to CPU fluctuations
			config.K8s.DeleteHPA(api.Name)
			return nil
		},
	)
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			_, err := config.K8s.DeleteDeployment(apiName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteService(apiName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(apiName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteHPA(apiName)
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	prefix := s.EnsureSuffix(filepath.Join("apis", apiName), "/")
	return config.AWS.DeleteFromS3ByPrefix(prefix, true)
}

func APIsBaseURL() (string, error) {
	service, err := config.K8sIstio.GetService("ingressgateway-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func DownloadAPISpec(apiName string, apiID string) (*spec.API, error) {
	s3Key := specKey(apiName, apiID)
	var api spec.API

	if err := config.AWS.ReadMsgpackFromS3(&api, s3Key); err != nil {
		return nil, err
	}

	return &api, nil
}

func DownloadAPISpecs(apiNames []string, apiIDs []string) ([]spec.API, error) {
	apis := make([]spec.API, len(apiNames))
	fns := make([]func() error, len(apiNames))

	for i := range apiNames {
		localIdx := i
		fns[i] = func() error {
			api, err := DownloadAPISpec(apiNames[localIdx], apiIDs[localIdx])
			if err != nil {
				return err
			}
			apis[localIdx] = *api
			return nil
		}
	}

	err := parallel.RunFirstErr(fns...)
	if err != nil {
		return nil, err
	}

	return apis, nil
}

func specKey(apiName string, apiID string) string {
	return filepath.Join(
		"apis",
		apiName,
		apiID,
		"spec.msgpack",
	)
}

func metadataRoot(apiName string, apiID string) string {
	return filepath.Join(
		"apis",
		apiName,
		apiID,
		"metadata",
	)
}

func ProjectKey(projectID string) string {
	return filepath.Join(
		"projects",
		projectID+".zip",
	)
}
