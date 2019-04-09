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
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

var serviceTypeMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "Service",
}

type ServiceSpec struct {
	Name       string
	Namespace  string
	Port       int32
	TargetPort int32
	Labels     map[string]string
	Selector   map[string]string
}

func Service(spec *ServiceSpec) *corev1.Service {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	service := &corev1.Service{
		TypeMeta: serviceTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: spec.Selector,
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     spec.Port,
					TargetPort: intstr.IntOrString{
						IntVal: spec.TargetPort,
					},
				},
			},
		},
	}
	return service
}

func CreateService(spec *ServiceSpec) (*corev1.Service, error) {
	service, err := serviceClient.Create(Service(spec))
	if err != nil {
		return nil, err
	}
	return service, nil
}

func GetService(name string) (*corev1.Service, error) {
	service, err := serviceClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	service.TypeMeta = serviceTypeMeta
	return service, nil
}

func DeleteService(name string) (bool, error) {
	err := serviceClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ServiceExists(name string) (bool, error) {
	service, err := GetService(name)
	if err != nil {
		return false, err
	}
	return service != nil, nil
}

func ListServices(opts *metav1.ListOptions) ([]corev1.Service, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	serviceList, err := serviceClient.List(*opts)
	if err != nil {
		return nil, err
	}
	for i := range serviceList.Items {
		serviceList.Items[i].TypeMeta = serviceTypeMeta
	}
	return serviceList.Items, nil
}

func ListServicesByLabels(labels map[string]string) ([]corev1.Service, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return ListServices(opts)
}

func ListServicesByLabel(labelKey string, labelValue string) ([]corev1.Service, error) {
	return ListServicesByLabels(map[string]string{labelKey: labelValue})
}

func ServiceMap(services []corev1.Service) map[string]corev1.Service {
	serviceMap := map[string]corev1.Service{}
	for _, service := range services {
		serviceMap[service.Name] = service
	}
	return serviceMap
}
