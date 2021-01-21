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

package k8s

import (
	"context"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _deploymentTypeMeta = kmeta.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

type DeploymentSpec struct {
	Name           string
	Replicas       int32
	PodSpec        PodSpec
	MaxSurge       *string // Can be a percentage (e.g. 10%) or an absolute number (e.g. 2)
	MaxUnavailable *string // Can be a percentage (e.g. 10%) or an absolute number (e.g. 2)
	Selector       map[string]string
	Labels         map[string]string
	Annotations    map[string]string
}

func Deployment(spec *DeploymentSpec) *kapps.Deployment {
	if spec.PodSpec.Name == "" {
		spec.PodSpec.Name = spec.Name
	}
	if spec.Selector == nil {
		spec.Selector = spec.PodSpec.Labels
	}

	var maxSurge *intstr.IntOrString
	if spec.MaxSurge != nil {
		intStr := intstr.Parse(*spec.MaxSurge)
		maxSurge = &intStr
	}

	var maxUnavailable *intstr.IntOrString
	if spec.MaxUnavailable != nil {
		intStr := intstr.Parse(*spec.MaxUnavailable)
		maxUnavailable = &intStr
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
			Strategy: kapps.DeploymentStrategy{
				Type: kapps.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &kapps.RollingUpdateDeployment{
					MaxSurge:       maxSurge,
					MaxUnavailable: maxUnavailable,
				},
			},
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
	deployment, err := c.deploymentClient.Create(context.Background(), deployment, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return deployment, nil
}

func (c *Client) UpdateDeployment(deployment *kapps.Deployment) (*kapps.Deployment, error) {
	deployment.TypeMeta = _deploymentTypeMeta
	deployment, err := c.deploymentClient.Update(context.Background(), deployment, kmeta.UpdateOptions{})
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
	deployment, err := c.deploymentClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	deployment.TypeMeta = _deploymentTypeMeta
	return deployment, nil
}

func (c *Client) DeleteDeployment(name string) (bool, error) {
	err := c.deploymentClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListDeployments(opts *kmeta.ListOptions) ([]kapps.Deployment, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	deploymentList, err := c.deploymentClient.List(context.Background(), *opts)
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
		LabelSelector: klabels.SelectorFromSet(labels).String(),
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

func DeploymentStrategiesMatch(s1, s2 kapps.DeploymentStrategy) bool {
	if s1.Type != s2.Type {
		return false
	}
	if s1.RollingUpdate == nil && s2.RollingUpdate == nil {
		return true
	}
	if s1.RollingUpdate == nil || s2.RollingUpdate == nil {
		return false
	}
	if !intOrStrPtrsMatch(s1.RollingUpdate.MaxUnavailable, s2.RollingUpdate.MaxUnavailable) {
		return false
	}
	if !intOrStrPtrsMatch(s1.RollingUpdate.MaxSurge, s2.RollingUpdate.MaxSurge) {
		return false
	}
	return true
}

func intOrStrPtrsMatch(intStr1, intStr2 *intstr.IntOrString) bool {
	if intStr1 == nil && intStr2 == nil {
		return true
	}
	if intStr1 == nil || intStr2 == nil {
		return false
	}
	return (*intStr1).String() == (*intStr2).String()
}
