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
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _serviceTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "Service",
}

type ServiceSpec struct {
	Name        string
	Port        int32
	TargetPort  int32
	ServiceType kcore.ServiceType
	Selector    map[string]string
	Labels      map[string]string
	Annotations map[string]string
}

func Service(spec *ServiceSpec) *kcore.Service {
	service := &kcore.Service{
		TypeMeta: _serviceTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: kcore.ServiceSpec{
			Selector: spec.Selector,
			Type:     spec.ServiceType,
			Ports: []kcore.ServicePort{
				{
					Protocol: kcore.ProtocolTCP,
					Name:     "http",
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

func (c *Client) CreateService(service *kcore.Service) (*kcore.Service, error) {
	service.TypeMeta = _serviceTypeMeta
	service, err := c.serviceClient.Create(context.Background(), service, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service, nil
}

func (c *Client) UpdateService(existing, updated *kcore.Service) (*kcore.Service, error) {
	updated.TypeMeta = _serviceTypeMeta
	updated.Spec.ClusterIP = existing.Spec.ClusterIP
	updated.ResourceVersion = existing.ResourceVersion

	service, err := c.serviceClient.Update(context.Background(), updated, kmeta.UpdateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service, nil
}

func (c *Client) ApplyService(service *kcore.Service) (*kcore.Service, error) {
	existing, err := c.GetService(service.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateService(service)
	}
	return c.UpdateService(existing, service)
}

func (c *Client) GetService(name string) (*kcore.Service, error) {
	service, err := c.serviceClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	service.TypeMeta = _serviceTypeMeta
	return service, nil
}

func (c *Client) DeleteService(name string) (bool, error) {
	err := c.serviceClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListServices(opts *kmeta.ListOptions) ([]kcore.Service, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	serviceList, err := c.serviceClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range serviceList.Items {
		serviceList.Items[i].TypeMeta = _serviceTypeMeta
	}
	return serviceList.Items, nil
}

func (c *Client) ListServicesByLabels(labels map[string]string) ([]kcore.Service, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListServices(opts)
}

func (c *Client) ListServicesByLabel(labelKey string, labelValue string) ([]kcore.Service, error) {
	return c.ListServicesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListServicesWithLabelKeys(labelKeys ...string) ([]kcore.Service, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListServices(opts)
}

func ServiceMap(services []kcore.Service) map[string]kcore.Service {
	serviceMap := map[string]kcore.Service{}
	for _, service := range services {
		serviceMap[service.Name] = service
	}
	return serviceMap
}
