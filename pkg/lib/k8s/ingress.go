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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kextensions "k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ingressTypeMeta = kmeta.TypeMeta{
	APIVersion: "extensions/v1beta1",
	Kind:       "Ingress",
}

type IngressSpec struct {
	Name         string
	IngressClass string
	ServiceName  string
	ServicePort  int32
	Path         string
	Labels       map[string]string
	Annotations  map[string]string
}

func Ingress(spec *IngressSpec) *kextensions.Ingress {
	if spec.Annotations == nil {
		spec.Annotations = make(map[string]string)
	}
	spec.Annotations["kubernetes.io/ingress.class"] = spec.IngressClass

	ingress := &kextensions.Ingress{
		TypeMeta: _ingressTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Annotations: spec.Annotations,
			Labels:      spec.Labels,
		},
		Spec: kextensions.IngressSpec{
			Rules: []kextensions.IngressRule{
				{
					IngressRuleValue: kextensions.IngressRuleValue{
						HTTP: &kextensions.HTTPIngressRuleValue{
							Paths: []kextensions.HTTPIngressPath{
								{
									Path: spec.Path,
									Backend: kextensions.IngressBackend{
										ServiceName: spec.ServiceName,
										ServicePort: intstr.IntOrString{
											IntVal: spec.ServicePort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ingress
}

func (c *Client) CreateIngress(ingress *kextensions.Ingress) (*kextensions.Ingress, error) {
	ingress.TypeMeta = _ingressTypeMeta
	ingress, err := c.ingressClient.Create(context.Background(), ingress, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ingress, nil
}

func (c *Client) UpdateIngress(ingress *kextensions.Ingress) (*kextensions.Ingress, error) {
	ingress.TypeMeta = _ingressTypeMeta
	ingress, err := c.ingressClient.Update(context.Background(), ingress, kmeta.UpdateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ingress, nil
}

func (c *Client) ApplyIngress(ingress *kextensions.Ingress) (*kextensions.Ingress, error) {
	existing, err := c.GetIngress(ingress.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateIngress(ingress)
	}
	return c.UpdateIngress(ingress)
}

func (c *Client) GetIngress(name string) (*kextensions.Ingress, error) {
	ingress, err := c.ingressClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	ingress.TypeMeta = _ingressTypeMeta
	return ingress, nil
}

func (c *Client) DeleteIngress(name string) (bool, error) {
	err := c.ingressClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListIngresses(opts *kmeta.ListOptions) ([]kextensions.Ingress, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	ingressList, err := c.ingressClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range ingressList.Items {
		ingressList.Items[i].TypeMeta = _ingressTypeMeta
	}
	return ingressList.Items, nil
}

func (c *Client) ListIngressesByLabels(labels map[string]string) ([]kextensions.Ingress, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListIngresses(opts)
}

func (c *Client) ListIngressesByLabel(labelKey string, labelValue string) ([]kextensions.Ingress, error) {
	return c.ListIngressesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListIngressesWithLabelKeys(labelKeys ...string) ([]kextensions.Ingress, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListIngresses(opts)
}

func IngressMap(ingresses []kextensions.Ingress) map[string]kextensions.Ingress {
	ingressMap := map[string]kextensions.Ingress{}
	for _, ingress := range ingresses {
		ingressMap[ingress.Name] = ingress
	}
	return ingressMap
}
