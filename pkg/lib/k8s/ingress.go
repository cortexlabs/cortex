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
	kextensions "k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var ingressTypeMeta = kmeta.TypeMeta{
	APIVersion: "extensions/v1beta1",
	Kind:       "Ingress",
}

type IngressSpec struct {
	Name         string
	Namespace    string
	IngressClass string
	ServiceName  string
	ServicePort  int32
	Path         string
	Labels       map[string]string
}

func Ingress(spec *IngressSpec) *kextensions.Ingress {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	ingress := &kextensions.Ingress{
		TypeMeta: ingressTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                                   spec.IngressClass,
				"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "https",
			},
			Labels: spec.Labels,
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

func (c *Client) CreateIngress(spec *IngressSpec) (*kextensions.Ingress, error) {
	ingress, err := c.ingressClient.Create(Ingress(spec))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ingress, nil
}

func (c *Client) UpdateIngress(ingress *kextensions.Ingress) (*kextensions.Ingress, error) {
	ingress, err := c.ingressClient.Update(ingress)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return ingress, nil
}

func (c *Client) GetIngress(name string) (*kextensions.Ingress, error) {
	ingress, err := c.ingressClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ingress.TypeMeta = ingressTypeMeta
	return ingress, nil
}

func (c *Client) DeleteIngress(name string) (bool, error) {
	err := c.ingressClient.Delete(name, deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) IngressExists(name string) (bool, error) {
	ingress, err := c.GetIngress(name)
	if err != nil {
		return false, err
	}
	return ingress != nil, nil
}

func (c *Client) ListIngresses(opts *kmeta.ListOptions) ([]kextensions.Ingress, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	ingressList, err := c.ingressClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range ingressList.Items {
		ingressList.Items[i].TypeMeta = ingressTypeMeta
	}
	return ingressList.Items, nil
}

func (c *Client) ListIngressesByLabels(labels map[string]string) ([]kextensions.Ingress, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListIngresses(opts)
}

func (c *Client) ListIngressesByLabel(labelKey string, labelValue string) ([]kextensions.Ingress, error) {
	return c.ListIngressesByLabels(map[string]string{labelKey: labelValue})
}

func IngressMap(ingresses []kextensions.Ingress) map[string]kextensions.Ingress {
	ingressMap := map[string]kextensions.Ingress{}
	for _, ingress := range ingresses {
		ingressMap[ingress.Name] = ingress
	}
	return ingressMap
}
