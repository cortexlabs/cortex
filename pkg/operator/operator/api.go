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
		err = config.AWS.UploadMsgpackToS3(api, *config.Cluster.Bucket, api.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("creating %s api", api.Name)

	} else if !areDeploymentsEqual(prevDeployment, deploymentSpec(api, prevDeployment)) {
		isMinReady, err := isDeploymentMinReady(prevDeployment)
		if err != nil {
			return nil, "", err
		}
		if !isMinReady && !force {
			return nil, "", errors.New(fmt.Sprintf("%s api is updating (override with --force)", api.Name))
		}
		err = config.AWS.UploadMsgpackToS3(api, *config.Cluster.Bucket, api.Key)
		if err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err = applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		msg = fmt.Sprintf("updating %s api", api.Name)

	} else { // deployment didn't change
		isMinReady, err := isDeploymentMinReady(prevDeployment)
		if err != nil {
			return nil, "", err
		}
		if !isMinReady {
			msg = fmt.Sprintf("%s api is already updating", api.Name)
		} else {
			msg = fmt.Sprintf("%s api is up to date", api.Name)
		}
	}

	return api, msg, nil
}

func RefreshAPI(apiName string, force bool) (string, error) {
	prevDeployment, err := config.K8s.GetDeployment(k8sName(apiName))
	if err != nil {
		return "", ErrorAPINotDeployed(apiName)
	}

	isMinReady, err := isDeploymentMinReady(prevDeployment)
	if err != nil {
		return "", err
	}
	if !isMinReady && !force {
		return "", errors.New(fmt.Sprintf("%s api is updating (override with --force)", apiName))
	}

	apiID := prevDeployment.Labels["apiID"]
	if apiID == "" {
		return "", errors.New("unable to retreive api ID from deployment") // unexpected
	}

	api, err := DownloadAPISpec(apiName, apiID)
	if err != nil {
		return "", err
	}

	api = getAPISpec(api.API, api.ProjectID, k8s.RandomName())

	err = config.AWS.UploadMsgpackToS3(api, *config.Cluster.Bucket, api.Key)
	if err != nil {
		return "", errors.Wrap(err, "upload api spec")
	}

	if err := applyK8sDeployment(api, prevDeployment); err != nil {
		return "", err
	}

	return fmt.Sprintf("updating %s api", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
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
		return err
	}

	return nil
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
			deployment, err = config.K8s.GetDeployment(k8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			service, err = config.K8s.GetService(k8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(k8sName(apiConfig.Name))
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

	return parallel.RunFirstErr(
		func() error {
			return applyK8sDeployment(api, prevDeployment)
		},
		func() error {
			return applyK8sService(api, prevService)
		},
		func() error {
			return applyK8sVirtualService(api, prevVirtualService)
		},
		func() error {
			// Delete HPA while updating replicas to avoid unwanted autoscaling due to CPU fluctuations
			config.K8s.DeleteHPA(k8sName(api.Name))
			return nil
		},
	)
}

func applyK8sDeployment(api *spec.API, prevDeployment *kapps.Deployment) error {
	newDeployment := deploymentSpec(api, prevDeployment)

	if prevDeployment == nil {
		_, err := config.K8s.CreateDeployment(newDeployment)
		return err
	}

	// Delete deployment if it never became ready
	if prevDeployment.Status.ReadyReplicas == 0 {
		config.K8s.DeleteDeployment(k8sName(api.Name))
		_, err := config.K8s.CreateDeployment(newDeployment)
		return err
	}

	_, err := config.K8s.UpdateDeployment(newDeployment)
	return err
}

func applyK8sService(api *spec.API, prevService *kcore.Service) error {
	newService := serviceSpec(api)

	if prevService == nil {
		_, err := config.K8s.CreateService(newService)
		return err
	}

	_, err := config.K8s.UpdateService(prevService, newService)
	return err
}

func applyK8sVirtualService(api *spec.API, prevVirtualService *kunstructured.Unstructured) error {
	newVirtualService := virtualServiceSpec(api)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			_, err := config.K8s.DeleteDeployment(k8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteService(k8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(k8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteHPA(k8sName(apiName))
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	prefix := s.EnsureSuffix(filepath.Join("apis", apiName), "/")
	return config.AWS.DeleteFromS3ByPrefix(*config.Cluster.Bucket, prefix, true)
}

func isDeploymentMinReady(deployment *kapps.Deployment) (bool, error) {
	pods, err := config.K8s.ListPodsByLabel("apiName", deployment.Labels["apiName"])
	if err != nil {
		return false, err
	}
	updatedReady := numUpdatedReadyReplicas(deployment, pods)

	minReplicas, ok := s.ParseInt32(deployment.Labels["minReplicas"])
	if !ok {
		return false, errors.New("unable to parse minReplicas from deployment") // unexpected
	}

	return updatedReady >= minReplicas, nil
}

func numUpdatedReadyReplicas(deployment *kapps.Deployment, pods []kcore.Pod) int32 {
	var readyReplicas int32
	for _, pod := range pods {
		if pod.Labels["apiName"] != deployment.Labels["apiName"] {
			continue
		}
		if k8s.IsPodReady(&pod) && isPodSpecLatest(deployment, &pod) {
			readyReplicas++
		}
	}
	return readyReplicas
}

func areDeploymentsEqual(d1, d2 *kapps.Deployment) bool {
	return deploymentLabelsEqual(d1.Labels, d2.Labels) &&
		k8s.PodComputesEqual(&d1.Spec.Template.Spec, &d2.Spec.Template.Spec)
}

func isPodSpecLatest(deployment *kapps.Deployment, pod *kcore.Pod) bool {
	return podLabelsEqual(deployment.Spec.Template.Labels, pod.Labels) &&
		k8s.PodComputesEqual(&deployment.Spec.Template.Spec, &pod.Spec)
}

func podLabelsEqual(labels1, labels2 map[string]string) bool {
	return labels1["apiName"] == labels2["apiName"] &&
		labels1["apiID"] == labels2["apiID"] &&
		labels1["deploymentID"] == labels2["deploymentID"]
}

func deploymentLabelsEqual(labels1, labels2 map[string]string) bool {
	return labels1["apiName"] == labels2["apiName"] &&
		labels1["apiID"] == labels2["apiID"] &&
		labels1["deploymentID"] == labels2["deploymentID"] &&
		labels1["minReplicas"] == labels2["minReplicas"] &&
		labels1["maxReplicas"] == labels2["maxReplicas"] &&
		labels1["targetCPUUtilization"] == labels2["targetCPUUtilization"]
}

func IsAPIDeployed(apiName string) (bool, error) {
	return config.K8s.DeploymentExists(k8sName(apiName))
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

	if err := config.AWS.ReadMsgpackFromS3(&api, *config.Cluster.Bucket, s3Key); err != nil {
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
