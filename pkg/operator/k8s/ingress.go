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
	v1beta1 "k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

var ingressTypeMeta = metav1.TypeMeta{
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

func Ingress(spec *IngressSpec) *v1beta1.Ingress {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	ingress := &v1beta1.Ingress{
		TypeMeta: ingressTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                                   spec.IngressClass,
				"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "https",
			},
			Labels: spec.Labels,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: spec.Path,
									Backend: v1beta1.IngressBackend{
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

func CreateIngress(spec *IngressSpec) (*v1beta1.Ingress, error) {
	ingress, err := ingressClient.Create(Ingress(spec))
	if err != nil {
		return nil, err
	}
	return ingress, nil
}

func GetIngress(name string) (*v1beta1.Ingress, error) {
	ingress, err := ingressClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ingress.TypeMeta = ingressTypeMeta
	return ingress, nil
}

func DeleteIngress(name string) (bool, error) {
	err := ingressClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func IngressExists(name string) (bool, error) {
	ingress, err := GetIngress(name)
	if err != nil {
		return false, err
	}
	return ingress != nil, nil
}

func ListIngresses(opts *metav1.ListOptions) ([]v1beta1.Ingress, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	ingressList, err := ingressClient.List(*opts)
	if err != nil {
		return nil, err
	}
	for i := range ingressList.Items {
		ingressList.Items[i].TypeMeta = ingressTypeMeta
	}
	return ingressList.Items, nil
}

func ListIngressesByLabels(labels map[string]string) ([]v1beta1.Ingress, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return ListIngresses(opts)
}

func ListIngressesByLabel(labelKey string, labelValue string) ([]v1beta1.Ingress, error) {
	return ListIngressesByLabels(map[string]string{labelKey: labelValue})
}

func IngressMap(ingresses []v1beta1.Ingress) map[string]v1beta1.Ingress {
	ingressMap := map[string]v1beta1.Ingress{}
	for _, ingress := range ingresses {
		ingressMap[ingress.Name] = ingress
	}
	return ingressMap
}
