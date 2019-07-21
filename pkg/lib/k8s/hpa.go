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
	kautoscaling "k8s.io/api/autoscaling/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var hpaTypeMeta = kmeta.TypeMeta{
	APIVersion: "autoscaling/v1",
	Kind:       "HorizontalPodAutoscaler",
}

type HPASpec struct {
	DeploymentName       string
	Namespace            string
	MinReplicas          int32
	MaxReplicas          int32
	TargetCPUUtilization int32
	Labels               map[string]string
}

func HPA(spec *HPASpec) *kautoscaling.HorizontalPodAutoscaler {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	hpa := &kautoscaling.HorizontalPodAutoscaler{
		TypeMeta: hpaTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:      spec.DeploymentName,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: kautoscaling.HorizontalPodAutoscalerSpec{
			MinReplicas:                    &spec.MinReplicas,
			MaxReplicas:                    spec.MaxReplicas,
			TargetCPUUtilizationPercentage: &spec.TargetCPUUtilization,
			ScaleTargetRef: kautoscaling.CrossVersionObjectReference{
				Kind:       deploymentTypeMeta.Kind,
				Name:       spec.DeploymentName,
				APIVersion: deploymentTypeMeta.APIVersion,
			},
		},
	}
	return hpa
}

func (c *Client) CreateHPA(hpa *kautoscaling.HorizontalPodAutoscaler) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa.TypeMeta = hpaTypeMeta
	hpa, err := c.hpaClient.Create(hpa)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hpa, nil
}

func (c *Client) updateHPA(hpa *kautoscaling.HorizontalPodAutoscaler) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa.TypeMeta = hpaTypeMeta
	hpa, err := c.hpaClient.Update(hpa)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hpa, nil
}

func (c *Client) ApplyHPA(hpa *kautoscaling.HorizontalPodAutoscaler) (*kautoscaling.HorizontalPodAutoscaler, error) {
	existing, err := c.GetHPA(hpa.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateHPA(hpa)
	}
	return c.updateHPA(hpa)
}

func (c *Client) GetHPA(name string) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa, err := c.hpaClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	hpa.TypeMeta = hpaTypeMeta
	return hpa, nil
}

func (c *Client) DeleteHPA(name string) (bool, error) {
	err := c.hpaClient.Delete(name, deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) HPAExists(name string) (bool, error) {
	hpa, err := c.GetHPA(name)
	if err != nil {
		return false, err
	}
	return hpa != nil, nil
}

func (c *Client) ListHPAs(opts *kmeta.ListOptions) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	hpaList, err := c.hpaClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range hpaList.Items {
		hpaList.Items[i].TypeMeta = hpaTypeMeta
	}
	return hpaList.Items, nil
}

func (c *Client) ListHPAsByLabels(labels map[string]string) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListHPAs(opts)
}

func (c *Client) ListHPAsByLabel(labelKey string, labelValue string) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	return c.ListHPAsByLabels(map[string]string{labelKey: labelValue})
}

func HPAMap(hpas []kautoscaling.HorizontalPodAutoscaler) map[string]kautoscaling.HorizontalPodAutoscaler {
	hpaMap := map[string]kautoscaling.HorizontalPodAutoscaler{}
	for _, hpa := range hpas {
		hpaMap[hpa.Name] = hpa
	}
	return hpaMap
}
