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

package workloads

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
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

	apiSpec, err := getAPISpec(apiConfig, projectID, deploymentID)
	if err != nil {
		return nil, "", err
	}

	var msg string

	if prevDeployment == nil {
		err = config.AWS.UploadMsgpackToS3(apiSpec, apiSpec.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(apiSpec, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("creating %s api", apiSpec.Name)

	} else if !k8s.AreDeploymentsEqual(prevDeployment, deploymentSpec(apiSpec)) {
		if isDeploymentUpdating(prevDeployment) && !force {
			return nil, "", errors.New(fmt.Sprintf("%s api is updating (override with --force)", apiSpec.Name))
		}
		err = config.AWS.UploadMsgpackToS3(apiSpec, apiSpec.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(apiSpec, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("updating %s api", apiSpec.Name)

	} else { // deployment didn't change
		if isDeploymentUpdating(prevDeployment) {
			msg = fmt.Sprintf("%s api is already updating", apiSpec.Name)
		} else {
			msg = fmt.Sprintf("%s api is up to date", apiSpec.Name)
		}
	}

	return apiSpec, msg, nil
}

func DeleteAPI(apiName string, keepCache bool) bool {
	wasDeployed := config.Kubernetes.DeploymentExists(apiName)

	parallel.Run(
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

	return wasDeployed
}

func getAPISpec(
	apiConfig *userconfig.API,
	projectID string,
	deploymentID string,
) (*context.Context, error) {

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
		ProjectKey:   projectKey(projectID),
	}

	ctx.ID = calculateID(ctx)
	ctx.Key = ctxKey(ctx.ID, ctx.App.Name)
	return ctx, nil
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
			deployment, err = config.Kubernetes.GetDeployment(apiConfig.Name)
			return err
		},
		func() error {
			var err error
			service, err = config.Kubernetes.GetService(apiConfig.Name)
			return err
		},
		func() error {
			var err error
			virtualService, err = config.Kubernetes.GetVirtualService(apiConfig.Name)
			return err
		},
	)

	return deployment, service, virtualService, err
}

func applyK8sResources(
	apiSpec *spec.API,
	prevDeployment *kapps.Deployment,
	prevService *kcore.Service,
	prevVirtualService *kunstructured.Unstructured,
) error {

	newDeployment := deploymentSpec(apiSpec, prevDeployment)
	newService := serviceSpec(apiSpec)
	newVirtualService := virtualServiceSpec(apiSpec)

	return parallel.RunFirstErr(
		func() error {
			if prevDeployment == nil {
				return config.Kubernetes.CreateDeployment(newDeployment)
			} else {
				// Delete deployment if it never became ready
				if prevDeployment.Status.ReadyReplicas == 0 {
					config.Kubernetes.DeleteDeployment(apiSpec.Name)
					return config.Kubernetes.CreateDeployment(newDeployment)
				}
				return config.Kubernetes.UpdateDeployment(newDeployment)
			}
		},
		func() error {
			if prevService == nil {
				return config.Kubernetes.CreateService(newService)
			} else {
				return config.Kubernetes.UpdateService(newService)
			}
		},
		func() error {
			if prevVirtualService == nil {
				return config.Kubernetes.CreateVirtualService(newVirtualService)
			} else {
				return config.Kubernetes.UpdateVirtualService(newVirtualService)
			}
		},
		func() error {
			// Delete HPA while updating replicas to avoid unwanted autoscaling due to CPU fluctuations
			config.Kubernetes.DeleteHPA(apiSpec.Name)
			return nil
		},
	)
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			return config.Kubernetes.DeleteDeployment(apiName)
		},
		func() error {
			return config.Kubernetes.DeleteService(apiName)
		},
		func() error {
			return config.Kubernetes.DeleteVirtualService(apiName, "default")
		},
		func() error {
			return config.Kubernetes.DeleteHPA(apiName)
		},
	)
}

func deleteS3Resources(apiName string) error {
	prefix := s.EnsureSuffix(filepath.Join("apis", apiName), "/")
	return config.AWS.DeleteFromS3ByPrefix(prefix, true)
}

func APIsBaseURL() (string, error) {
	service, err := config.IstioKubernetes.GetService("apis-ingressgateway")
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
			apis[localIdx] = api
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
