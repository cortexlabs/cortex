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
	"context"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	istioapi "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	_virtualServiceTypeMeta = v1.TypeMeta{
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

func VirtualService(spec *VirtualServiceSpec) *v1alpha3.VirtualService {
	virtualService := &v1alpha3.VirtualService{
		TypeMeta: _virtualServiceTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: istioapi.VirtualService{
			Hosts:    []string{"*"},
			Gateways: spec.Gateways,
			Http: []*istioapi.HTTPRoute{{
				Match: []*istioapi.HTTPMatchRequest{{
					Uri: &istioapi.StringMatch{
						MatchType: &istioapi.StringMatch_Exact{
							Exact: urls.CanonicalizeEndpoint(spec.Path),
						},
					},
				},
				},
				Route: []*istioapi.HTTPRouteDestination{{
					Destination: &istioapi.Destination{
						Host: spec.ServiceName,
						Port: &istioapi.PortSelector{
							Number: uint32(spec.ServicePort),
						},
					},
				},
				},
			},
			},
		},
	}

	return virtualService
}

func (c *Client) CreateVirtualService(spec *v1alpha3.VirtualService) (*v1alpha3.VirtualService, error) {
	spec.SetNamespace(c.Namespace)

	istio, err := versionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(spec.GetNamespace()).Create(context.TODO(), spec, v1.CreateOptions{
		TypeMeta: _virtualServiceTypeMeta,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(existing, updated *v1alpha3.VirtualService) (*v1alpha3.VirtualService, error) {
	updated.SetNamespace(c.Namespace)
	updated.SetResourceVersion(existing.GetResourceVersion())

	istio, err := versionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(updated.GetNamespace()).Update(context.TODO(), updated, v1.UpdateOptions{
		TypeMeta: _virtualServiceTypeMeta,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(spec *v1alpha3.VirtualService) (*v1alpha3.VirtualService, error) {
	existing, err := c.GetVirtualService(spec.GetName())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateVirtualService(spec)
	}
	return c.UpdateVirtualService(existing, spec)
}

func (c *Client) GetVirtualService(name string) (*v1alpha3.VirtualService, error) {

	istio, err := versionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(c.Namespace).Get(context.TODO(), name, v1.GetOptions{
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
	istio, err := versionedclient.NewForConfig(c.RestConfig)
	err = istio.NetworkingV1alpha3().VirtualServices(c.Namespace).Delete(context.TODO(), name, v1.DeleteOptions{
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

func (c *Client) ListVirtualServices(opts *v1.ListOptions) ([]v1alpha3.VirtualService, error) {
	if opts == nil {
		opts = &v1.ListOptions{}
	}

	istio, err := versionedclient.NewForConfig(c.RestConfig)
	vsList, err := istio.NetworkingV1alpha3().VirtualServices(c.Namespace).List(context.TODO(), v1.ListOptions{
		TypeMeta: _virtualServiceTypeMeta,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range vsList.Items {
		vsList.Items[i].SetGroupVersionKind(_virtualServiceGVK)
	}
	return vsList.Items, nil
}

func (c *Client) ListVirtualServicesByLabels(labels map[string]string) ([]v1alpha3.VirtualService, error) {
	opts := &v1.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListVirtualServices(opts)
}

func (c *Client) ListVirtualServicesByLabel(labelKey string, labelValue string) ([]v1alpha3.VirtualService, error) {
	return c.ListVirtualServicesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListVirtualServicesWithLabelKeys(labelKeys ...string) ([]v1alpha3.VirtualService, error) {
	opts := &v1.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListVirtualServices(opts)
}

func ExtractVirtualServiceGateways(virtualService *v1alpha3.VirtualService) (strset.Set, error) {
	return strset.FromSlice(virtualService.Spec.Gateways), nil
}

func ExtractVirtualServiceEndpoints(virtualService *v1alpha3.VirtualService) (strset.Set, error) {
	endpoints := strset.New()

	httpRoutes := virtualService.Spec.Http
	for _, http := range httpRoutes {
		for _, match := range http.Match {
			endpoints.Add(urls.CanonicalizeEndpoint(match.Uri.GetExact()))
		}
	}
	return endpoints, nil
}
