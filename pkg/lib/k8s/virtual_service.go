/*
Copyright 2020 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	_virtualServiceTypeMeta = kmeta.TypeMeta{
		APIVersion: "v1alpha3",
		Kind:       "VirtualService",
	}

	_virtualServiceGVR = kschema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	}

	_virtualServiceGVK = kschema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}
)

type VirtualServiceSpec struct {
	Name        string
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
	virtualServiceConfig.SetGroupVersionKind(_virtualServiceGVK)
	virtualServiceConfig.SetName(spec.Name)
	virtualServiceConfig.Object["metadata"] = map[string]interface{}{
		"name":        spec.Name,
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
	spec.Object["metadata"].(map[string]interface{})["namespace"] = c.Namespace

	virtualService, err := c.dynamicClient.
		Resource(_virtualServiceGVR).
		Namespace(spec.GetNamespace()).
		Create(spec, kmeta.CreateOptions{
			TypeMeta: _virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(existing, updated *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	updated.Object["metadata"].(map[string]interface{})["namespace"] = c.Namespace
	updated.SetResourceVersion(existing.GetResourceVersion())

	virtualService, err := c.dynamicClient.
		Resource(_virtualServiceGVR).
		Namespace(updated.GetNamespace()).
		Update(updated, kmeta.UpdateOptions{
			TypeMeta: _virtualServiceTypeMeta,
		})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(spec *kunstructured.Unstructured) (*kunstructured.Unstructured, error) {
	existing, err := c.GetVirtualService(spec.GetName())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateVirtualService(spec)
	}
	return c.UpdateVirtualService(existing, spec)
}

func (c *Client) GetVirtualService(name string) (*kunstructured.Unstructured, error) {
	virtualService, err := c.dynamicClient.Resource(_virtualServiceGVR).Namespace(c.Namespace).Get(name, kmeta.GetOptions{
		TypeMeta: _virtualServiceTypeMeta,
	})

	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) DeleteVirtualService(name string) (bool, error) {
	err := c.dynamicClient.Resource(_virtualServiceGVR).Namespace(c.Namespace).Delete(name, &kmeta.DeleteOptions{
		TypeMeta: _virtualServiceTypeMeta,
	})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListVirtualServices(opts *kmeta.ListOptions) ([]kunstructured.Unstructured, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}

	vsList, err := c.dynamicClient.Resource(_virtualServiceGVR).Namespace(c.Namespace).List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range vsList.Items {
		vsList.Items[i].SetGroupVersionKind(_virtualServiceGVK)
	}
	return vsList.Items, nil
}

func (c *Client) ListVirtualServicesByLabels(labels map[string]string) ([]kunstructured.Unstructured, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListVirtualServices(opts)
}

func (c *Client) ListVirtualServicesByLabel(labelKey string, labelValue string) ([]kunstructured.Unstructured, error) {
	return c.ListVirtualServicesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListVirtualServicesWithLabelKeys(labelKeys ...string) ([]kunstructured.Unstructured, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListVirtualServices(opts)
}

func ExtractVirtualServiceGateways(virtualService *kunstructured.Unstructured) (strset.Set, error) {
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

func ExtractVirtualServiceEndpoints(virtualService *kunstructured.Unstructured) (strset.Set, error) {
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
