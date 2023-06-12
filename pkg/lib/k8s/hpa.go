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

package k8s

import (
	"context"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kautoscaling "k8s.io/api/autoscaling/v2"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _hpaTypeMeta = kmeta.TypeMeta{
	APIVersion: "autoscaling/v1",
	Kind:       "HorizontalPodAutoscaler",
}

type HPASpec struct {
	DeploymentName       string
	MinReplicas          int32
	MaxReplicas          int32
	TargetCPUUtilization int32
	TargetMemUtilization int32
	Labels               map[string]string
	Annotations          map[string]string
}

func HPA(spec *HPASpec) (*kautoscaling.HorizontalPodAutoscaler, error) {
	metrics := []kautoscaling.MetricSpec{}
	if spec.TargetCPUUtilization > 0 {
		metrics = append(metrics, kautoscaling.MetricSpec{
			Type: kautoscaling.ResourceMetricSourceType,
			Resource: &kautoscaling.ResourceMetricSource{
				Name: kcore.ResourceCPU,
				Target: kautoscaling.MetricTarget{
					Type:               kautoscaling.UtilizationMetricType,
					AverageUtilization: &spec.TargetCPUUtilization,
				},
			},
		})
	}
	if spec.TargetMemUtilization > 0 {
		metrics = append(metrics, kautoscaling.MetricSpec{
			Type: kautoscaling.ResourceMetricSourceType,
			Resource: &kautoscaling.ResourceMetricSource{
				Name: kcore.ResourceMemory,
				Target: kautoscaling.MetricTarget{
					Type:               kautoscaling.UtilizationMetricType,
					AverageUtilization: &spec.TargetMemUtilization,
				},
			},
		})
	}

	if len(metrics) == 0 {
		return nil, ErrorMissingMetrics()
	}

	hpa := &kautoscaling.HorizontalPodAutoscaler{
		TypeMeta: _hpaTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.DeploymentName,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: kautoscaling.HorizontalPodAutoscalerSpec{
			MinReplicas: &spec.MinReplicas,
			MaxReplicas: spec.MaxReplicas,
			Metrics:     metrics,
			ScaleTargetRef: kautoscaling.CrossVersionObjectReference{
				Kind:       _deploymentTypeMeta.Kind,
				Name:       spec.DeploymentName,
				APIVersion: _deploymentTypeMeta.APIVersion,
			},
		},
	}
	return hpa, nil
}

func (c *Client) CreateHPA(hpa *kautoscaling.HorizontalPodAutoscaler) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa.TypeMeta = _hpaTypeMeta
	hpa, err := c.hpaClient.Create(context.Background(), hpa, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hpa, nil
}

func (c *Client) UpdateHPA(hpa *kautoscaling.HorizontalPodAutoscaler) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa.TypeMeta = _hpaTypeMeta
	hpa, err := c.hpaClient.Update(context.Background(), hpa, kmeta.UpdateOptions{})
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
	return c.UpdateHPA(hpa)
}

func (c *Client) GetHPA(name string) (*kautoscaling.HorizontalPodAutoscaler, error) {
	hpa, err := c.hpaClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	hpa.TypeMeta = _hpaTypeMeta
	return hpa, nil
}

func (c *Client) DeleteHPA(name string) (bool, error) {
	err := c.hpaClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListHPAs(opts *kmeta.ListOptions) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	hpaList, err := c.hpaClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range hpaList.Items {
		hpaList.Items[i].TypeMeta = _hpaTypeMeta
	}
	return hpaList.Items, nil
}

func (c *Client) ListHPAsByLabels(labels map[string]string) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListHPAs(opts)
}

func (c *Client) ListHPAsByLabel(labelKey string, labelValue string) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	return c.ListHPAsByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListHPAsWithLabelKeys(labelKeys ...string) ([]kautoscaling.HorizontalPodAutoscaler, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListHPAs(opts)
}

func HPAMap(hpas []kautoscaling.HorizontalPodAutoscaler) map[string]kautoscaling.HorizontalPodAutoscaler {
	hpaMap := map[string]kautoscaling.HorizontalPodAutoscaler{}
	for _, hpa := range hpas {
		hpaMap[hpa.Name] = hpa
	}
	return hpaMap
}

func IsHPAUpToDate(hpa *kautoscaling.HorizontalPodAutoscaler, minReplicas, maxReplicas, targetCPUUtilization, targetMemUtilization int32) bool {
	if hpa == nil {
		return false
	}

	if hpa.Spec.MinReplicas == nil || *hpa.Spec.MinReplicas != minReplicas {
		return false
	}

	if hpa.Spec.MaxReplicas != maxReplicas {
		return false
	}

	if len(hpa.Spec.Metrics) != 2 {
		return false
	}

	for _, metric := range hpa.Spec.Metrics {
		if metric.Type != kautoscaling.ResourceMetricSourceType || metric.Resource == nil {
			return false
		}
		if metric.Resource.Target.Type != kautoscaling.UtilizationMetricType || metric.Resource.Target.AverageUtilization == nil {
			return false
		}
		if metric.Resource.Name != kcore.ResourceCPU && metric.Resource.Name != kcore.ResourceMemory {
			return false
		}
		if metric.Resource.Name == kcore.ResourceCPU && *metric.Resource.Target.AverageUtilization != targetCPUUtilization {
			return false
		}
		if metric.Resource.Name == kcore.ResourceMemory && *metric.Resource.Target.AverageUtilization != targetMemUtilization {
			return false
		}
	}

	return true
}
