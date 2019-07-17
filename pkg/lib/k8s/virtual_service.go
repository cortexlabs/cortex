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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var virtualServiceTypeMeta = metav1.TypeMeta{
	APIVersion: "v1alpha3",
	Kind:       "VirtualService",
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
	virtualServceConfig.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	})
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

func (c *Client) CreateVirtualService(spec *VirtualServiceSpec) (*unstructured.Unstructured, error) {
	virtualServiceGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	}

	service, err := c.dynamicClient.Resource(virtualServiceGVR).Namespace(spec.Namespace).Create(VirtualService(spec), metav1.CreateOptions{
		TypeMeta: virtualServiceTypeMeta,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service, nil
}
