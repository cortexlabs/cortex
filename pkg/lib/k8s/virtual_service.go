package k8s

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

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var virtualServiceTypeMeta = metav1.TypeMeta{
	APIVersion: "v1alpha3",
	Kind:       "VirtualService",
}

var virtualServiceGVR = schema.GroupVersionResource{
	Group:    "networking.istio.io",
	Version:  "v1alpha3",
	Resource: "virtualservices",
}

var virtualServiceGVK = schema.GroupVersionKind{
	Group:   "networking.istio.io",
	Version: "v1alpha3",
	Kind:    "VirtualService",
}

type VirtualServiceSpec struct {
	Name        string
	Namespace   string
	Gateways    []string
	ServiceName string
	ServicePort int32
	Path        string
	Labels      map[string]string
}

func VirtualService(spec *VirtualServiceSpec) *unstructured.Unstructured {
	virtualServceConfig := &unstructured.Unstructured{}
	virtualServceConfig.SetGroupVersionKind(virtualServiceGVK)
	virtualServceConfig.SetName(spec.Name)
	virtualServceConfig.SetNamespace(spec.Namespace)
	virtualServceConfig.Object["metadata"] = map[string]interface{}{
		"name":      spec.Name,
		"namespace": spec.Namespace,
	}
	virtualServceConfig.Object["spec"] = map[string]interface{}{
		"hosts":    []string{"*"},
		"gateways": spec.Gateways,
		"http": []map[string]interface{}{
			map[string]interface{}{
				"match": []map[string]interface{}{
					map[string]interface{}{
						"uri": map[string]interface{}{
							"prefix": spec.Path,
						},
					},
				},
				"route": []map[string]interface{}{
					map[string]interface{}{
						"destination": map[string]interface{}{
							"host": spec.ServiceName,
							"port": map[string]interface{}{
								"number": spec.ServicePort,
							},
						},
					},
				},
			},
		},
	}

	return virtualServceConfig
}

func (c *Client) CreateVirtualService(spec *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.
		Resource(virtualServiceGVR).
		Namespace(spec.GetNamespace()).
		Create(spec, metav1.CreateOptions{
			TypeMeta: virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(spec *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.
		Resource(virtualServiceGVR).
		Namespace(spec.GetNamespace()).
		Update(spec, metav1.UpdateOptions{
			TypeMeta: virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(spec *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	existing, err := c.GetVirtualService(spec.GetName(), spec.GetNamespace())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateVirtualService(spec)
	}
	spec.SetResourceVersion(existing.GetResourceVersion())
	return c.UpdateVirtualService(spec)
}

func (c *Client) GetVirtualService(name, namespace string) (*unstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(name, metav1.GetOptions{
		TypeMeta: virtualServiceTypeMeta,
	})

	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) DeleteVirtualService(name, namespace string) (bool, error) {
	err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Delete(name, &metav1.DeleteOptions{
		TypeMeta: virtualServiceTypeMeta,
	})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListVirtualServices(namespace string, opts *kmeta.ListOptions) ([]unstructured.Unstructured, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}

	vsList, err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range vsList.Items {
		vsList.Items[i].SetGroupVersionKind(virtualServiceGVK)
	}
	return vsList.Items, nil
}

func (c *Client) ListVirtualServicesByLabels(namespace string, labels map[string]string) ([]unstructured.Unstructured, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListVirtualServices(namespace, opts)
}

func (c *Client) ListVirtualServicesByLabel(namespace string, labelKey string, labelValue string) ([]unstructured.Unstructured, error) {
	return c.ListVirtualServicesByLabels(namespace, map[string]string{labelKey: labelValue})
}
