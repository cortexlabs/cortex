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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

var (
	virtualServiceTypeMeta = kmeta.TypeMeta{
		APIVersion: "v1alpha3",
		Kind:       "VirtualService",
	}

	virtualServiceGVR = kschema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	}

	virtualServiceGVK = kschema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}
)

type VirtualServiceSpec struct {
	Name        string
	Namespace   string
	Gateways    []string
	ServiceName string
	ServicePort int32
	Path        string
	Rewrite     *string
	Labels      map[string]string
	Annotations map[string]string
}

func VirtualService(spec *VirtualServiceSpec) *kunstructured.Unstructured {
	virtualServiceConfig := &kunstructured.Unstructured{}
	virtualServiceConfig.SetGroupVersionKind(virtualServiceGVK)
	virtualServiceConfig.SetName(spec.Name)
	virtualServiceConfig.SetNamespace(spec.Namespace)
	virtualServiceConfig.Object["metadata"] = map[string]interface{}{
		"name":        spec.Name,
		"namespace":   spec.Namespace,
		"labels":      spec.Labels,
		"annotations": spec.Annotations,
	}

	httpSpec := map[string]interface{}{
		"match": []map[string]interface{}{
			{
				"uri": map[string]interface{}{
					"exact": urls.CanonicalizeEndpoint(spec.Path),
				},
			},
		},
		"route": []map[string]interface{}{
			{
				"destination": map[string]interface{}{
					"host": spec.ServiceName,
					"port": map[string]interface{}{
						"number": spec.ServicePort,
					},
				},
			},
		},
	}

	if spec.Rewrite != nil && urls.CanonicalizeEndpoint(*spec.Rewrite) != urls.CanonicalizeEndpoint(spec.Path) {
		httpSpec["rewrite"] = map[string]interface{}{
			"uri": urls.CanonicalizeEndpoint(*spec.Rewrite),
		}
	}

	virtualServiceConfig.Object["spec"] = map[string]interface{}{
		"hosts":    []string{"*"},
		"gateways": spec.Gateways,
		"http":     []map[string]interface{}{httpSpec},
	}

	return virtualServiceConfig
}

func (c *Client) CreateVirtualService(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.
		Resource(virtualServiceGVR).
		Namespace(spec.GetNamespace()).
		Create(spec, kmeta.CreateOptions{
			TypeMeta: virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.
		Resource(virtualServiceGVR).
		Namespace(spec.GetNamespace()).
		Update(spec, kmeta.UpdateOptions{
			TypeMeta: virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
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

func (c *Client) GetVirtualService(name, namespace string) (*kunstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(name, kmeta.GetOptions{
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

func (c *Client) VirtualServiceExists(name, namespace string) (bool, error) {
	service, err := c.GetVirtualService(name, namespace)
	if err != nil {
		return false, err
	}
	return service != nil, nil
}

func (c *Client) DeleteVirtualService(name, namespace string) (bool, error) {
	err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Delete(name, &kmeta.DeleteOptions{
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

func (c *Client) ListVirtualServices(namespace string, opts *kmeta.ListOptions) ([]kunstructured.Unstructured, error) {
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

func (c *Client) ListVirtualServicesByLabels(namespace string, labels map[string]string) ([]kunstructured.Unstructured, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListVirtualServices(namespace, opts)
}

func (c *Client) ListVirtualServicesByLabel(namespace string, labelKey string, labelValue string) ([]kunstructured.Unstructured, error) {
	return c.ListVirtualServicesByLabels(namespace, map[string]string{labelKey: labelValue})
}

func GetVirtualServiceGateways(virtualService *kunstructured.Unstructured) (strset.Set, error) {
	spec, ok := virtualService.UnstructuredContent()["spec"].(map[string]interface{})
	if !ok {
		return nil, errors.New("virtual service spec is not a map[string]interface{}") // unexpected
	}

	gatewaysInterface, ok := spec["gateways"]
	if !ok {
		return strset.New(), nil
	}
	gateways, ok := gatewaysInterface.([]interface{})
	if !ok {
		return nil, errors.New("gateways is not a []interface{}") // unexpected
	}

	gatewayStrs := strset.New()

	for _, gatewayInterface := range gateways {
		gateway, ok := gatewayInterface.(string)
		if !ok {
			return nil, errors.New("gateway is not a string") // unexpected
		}
		gatewayStrs.Add(gateway)
	}

	return gatewayStrs, nil
}

func GetVirtualServiceEndpoints(virtualService *kunstructured.Unstructured) (strset.Set, error) {
	spec, ok := virtualService.UnstructuredContent()["spec"].(map[string]interface{})
	if !ok {
		return nil, errors.New("virtual service spec is not a map[string]interface{}") // unexpected
	}

	httpConfigsInterface, ok := spec["http"]
	if !ok {
		return strset.New("/"), nil
	}
	httpConfigs, ok := httpConfigsInterface.([]interface{})
	if !ok {
		return nil, errors.New("http is not a []interface{}") // unexpected
	}

	endpoints := strset.New()

	for _, httpConfigInterface := range httpConfigs {
		httpConfig, ok := httpConfigInterface.(map[string]interface{})
		if !ok {
			return nil, errors.New("http item is not a map[string]interface{}") // unexpected
		}

		matchesInterface, ok := httpConfig["match"]
		if !ok {
			return strset.New("/"), nil
		}
		matches, ok := matchesInterface.([]interface{})
		if !ok {
			return nil, errors.New("match is not a []interface{}") // unexpected
		}

		for _, matchInterface := range matches {
			match, ok := matchInterface.(map[string]interface{})
			if !ok {
				return nil, errors.New("match item is not a map[string]interface{}") // unexpected
			}

			uriInterface, ok := match["uri"]
			if !ok {
				return strset.New("/"), nil
			}
			uri, ok := uriInterface.(map[string]interface{})
			if !ok {
				return nil, errors.New("uri is not a map[string]interface{}") // unexpected
			}

			exactInferface, ok := uri["exact"]
			if !ok {
				return strset.New("/"), nil
			}
			exact, ok := exactInferface.(string)
			if !ok {
				return nil, errors.New("url is not a string") // unexpected
			}

			endpoints.Add(urls.CanonicalizeEndpoint(exact))
		}
	}

	return endpoints, nil
}
