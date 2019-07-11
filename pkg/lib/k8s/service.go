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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
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

func (c *Client) CreateService(spec *ServiceSpec) (*corev1.Service, error) {
	service, err := c.serviceClient.Create(Service(spec))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service, nil
}

func (c *Client) UpdateService(service *corev1.Service) (*corev1.Service, error) {
	service, err := c.serviceClient.Update(service)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service, nil
}

func (c *Client) GetService(name string) (*corev1.Service, error) {
	service, err := c.serviceClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	service.TypeMeta = serviceTypeMeta
	return service, nil
}

func (c *Client) GetIstioService(name string) (*corev1.Service, error) {
	service, err := c.istioServiceClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	service.TypeMeta = serviceTypeMeta
	return service, nil
}

func (c *Client) DeleteService(name string) (bool, error) {
	err := c.serviceClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ServiceExists(name string) (bool, error) {
	service, err := c.GetService(name)
	if err != nil {
		return false, err
	}
	return service != nil, nil
}

func (c *Client) ListServices(opts *metav1.ListOptions) ([]corev1.Service, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	serviceList, err := c.serviceClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range serviceList.Items {
		serviceList.Items[i].TypeMeta = serviceTypeMeta
	}
	return serviceList.Items, nil
}

func (c *Client) ListServicesByLabels(labels map[string]string) ([]corev1.Service, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListServices(opts)
}

func (c *Client) ListServicesByLabel(labelKey string, labelValue string) ([]corev1.Service, error) {
	return c.ListServicesByLabels(map[string]string{labelKey: labelValue})
}

func ServiceMap(services []corev1.Service) map[string]corev1.Service {
	serviceMap := map[string]corev1.Service{}
	for _, service := range services {
		serviceMap[service.Name] = service
	}
	return serviceMap
}
