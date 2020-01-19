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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _deploymentTypeMeta = kmeta.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

type DeploymentSpec struct {
	Name        string
	Replicas    int32
	PodSpec     PodSpec
	Selector    map[string]string
	Labels      map[string]string
	Annotations map[string]string
}

func Deployment(spec *DeploymentSpec) *kapps.Deployment {
	if spec.PodSpec.Name == "" {
		spec.PodSpec.Name = spec.Name
	}
	if spec.Selector == nil {
		spec.Selector = spec.PodSpec.Labels
	}

	deployment := &kapps.Deployment{
		TypeMeta: _deploymentTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: kapps.DeploymentSpec{
			Replicas: &spec.Replicas,
			Template: kcore.PodTemplateSpec{
				ObjectMeta: kmeta.ObjectMeta{
					Name:        spec.PodSpec.Name,
					Labels:      spec.PodSpec.Labels,
					Annotations: spec.PodSpec.Annotations,
				},
				Spec: spec.PodSpec.K8sPodSpec,
			},
			Selector: &kmeta.LabelSelector{
				MatchLabels: spec.Selector,
			},
		},
	}
	return deployment
}

func (c *Client) CreateDeployment(deployment *kapps.Deployment) (*kapps.Deployment, error) {
	deployment.TypeMeta = _deploymentTypeMeta
	deployment, err := c.deploymentClient.Create(deployment)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return deployment, nil
}

func (c *Client) UpdateDeployment(deployment *kapps.Deployment) (*kapps.Deployment, error) {
	deployment.TypeMeta = _deploymentTypeMeta
	deployment, err := c.deploymentClient.Update(deployment)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return deployment, nil
}

func (c *Client) ApplyDeployment(deployment *kapps.Deployment) (*kapps.Deployment, error) {
	existing, err := c.GetDeployment(deployment.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateDeployment(deployment)
	}
	return c.UpdateDeployment(deployment)
}

func (c *Client) GetDeployment(name string) (*kapps.Deployment, error) {
	deployment, err := c.deploymentClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	deployment.TypeMeta = _deploymentTypeMeta
	return deployment, nil
}

func (c *Client) DeleteDeployment(name string) (bool, error) {
	err := c.deploymentClient.Delete(name, _deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListDeployments(opts *kmeta.ListOptions) ([]kapps.Deployment, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	deploymentList, err := c.deploymentClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range deploymentList.Items {
		deploymentList.Items[i].TypeMeta = _deploymentTypeMeta
	}
	return deploymentList.Items, nil
}

func (c *Client) ListDeploymentsByLabels(labels map[string]string) ([]kapps.Deployment, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListDeployments(opts)
}

func (c *Client) ListDeploymentsByLabel(labelKey string, labelValue string) ([]kapps.Deployment, error) {
	return c.ListDeploymentsByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListDeploymentsWithLabelKeys(labelKeys ...string) ([]kapps.Deployment, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListDeployments(opts)
}

func DeploymentMap(deployments []kapps.Deployment) map[string]kapps.Deployment {
	deploymentMap := map[string]kapps.Deployment{}
	for _, deployment := range deployments {
		deploymentMap[deployment.Name] = deployment
	}
	return deploymentMap
}

func DeploymentStartTime(deployment *kapps.Deployment) *time.Time {
	if deployment == nil {
		return nil
	}
	t := deployment.CreationTimestamp.Time
	if t.IsZero() {
		return nil
	}
	return &deployment.CreationTimestamp.Time
}
