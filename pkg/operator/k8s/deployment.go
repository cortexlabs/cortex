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

package k8s

import (
	"time"

	appsv1b1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

const DeploymentSuccessConditionAll = "!status.unavailableReplicas"

type DeploymentSpec struct {
	Name      string
	Namespace string
	Replicas  int32
	PodSpec   PodSpec
	Labels    map[string]string
	Selector  map[string]string
}

func Deployment(spec *DeploymentSpec) *appsv1b1.Deployment {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	if spec.PodSpec.Namespace == "" {
		spec.PodSpec.Namespace = spec.Namespace
	}
	if spec.PodSpec.Name == "" {
		spec.PodSpec.Name = spec.Name
	}
	if spec.Selector == nil {
		spec.Selector = spec.PodSpec.Labels
	}

	deployment := &appsv1b1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: appsv1b1.DeploymentSpec{
			Replicas: &spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spec.PodSpec.Name,
					Namespace: spec.PodSpec.Namespace,
					Labels:    spec.PodSpec.Labels,
				},
				Spec: spec.PodSpec.K8sPodSpec,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: spec.Selector,
			},
		},
	}
	return deployment
}

func CreateDeployment(spec *DeploymentSpec) (*appsv1b1.Deployment, error) {
	deployment, err := deploymentClient.Create(Deployment(spec))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return deployment, nil
}

func UpdateDeployment(deployment *appsv1b1.Deployment) (*appsv1b1.Deployment, error) {
	deployment, err := deploymentClient.Update(deployment)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return deployment, nil
}

func GetDeployment(name string) (*appsv1b1.Deployment, error) {
	deployment, err := deploymentClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}
	deployment.TypeMeta = deploymentTypeMeta
	return deployment, nil
}

func DeleteDeployment(name string) (bool, error) {
	err := deploymentClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err)
	}
	return true, nil
}

func DeploymentExists(name string) (bool, error) {
	deployment, err := GetDeployment(name)
	if err != nil {
		return false, err
	}
	return deployment != nil, nil
}

func ListDeployments(opts *metav1.ListOptions) ([]appsv1b1.Deployment, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	deploymentList, err := deploymentClient.List(*opts)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for i := range deploymentList.Items {
		deploymentList.Items[i].TypeMeta = deploymentTypeMeta
	}
	return deploymentList.Items, nil
}

func ListDeploymentsByLabels(labels map[string]string) ([]appsv1b1.Deployment, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return ListDeployments(opts)
}

func ListDeploymentsByLabel(labelKey string, labelValue string) ([]appsv1b1.Deployment, error) {
	return ListDeploymentsByLabels(map[string]string{labelKey: labelValue})
}

func DeploymentMap(deployments []appsv1b1.Deployment) map[string]appsv1b1.Deployment {
	deploymentMap := map[string]appsv1b1.Deployment{}
	for _, deployment := range deployments {
		deploymentMap[deployment.Name] = deployment
	}
	return deploymentMap
}

func DeploymentStartTime(deployment *appsv1b1.Deployment) *time.Time {
	if deployment == nil {
		return nil
	}
	t := deployment.CreationTimestamp.Time
	if t.IsZero() {
		return nil
	}
	return &deployment.CreationTimestamp.Time
}
