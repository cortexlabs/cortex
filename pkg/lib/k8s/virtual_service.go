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
	istionetworking "istio.io/api/networking/v1alpha3"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioversionedclient "istio.io/client-go/pkg/clientset/versioned"
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

func VirtualService(spec *VirtualServiceSpec) *istioclientnetworking.VirtualService {
	virtualService := &istioclientnetworking.VirtualService{
		TypeMeta: _virtualServiceTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: istionetworking.VirtualService{
			Hosts:    []string{"*"},
			Gateways: spec.Gateways,
			Http: []*istionetworking.HTTPRoute{{
				Match: []*istionetworking.HTTPMatchRequest{{
					Uri: &istionetworking.StringMatch{
						MatchType: &istionetworking.StringMatch_Exact{
							Exact: urls.CanonicalizeEndpoint(spec.Path),
						},
					},
				},
				},
				Route: []*istionetworking.HTTPRouteDestination{{
					Destination: &istionetworking.Destination{
						Host: spec.ServiceName,
						Port: &istionetworking.PortSelector{
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

func (c *Client) CreateVirtualService(spec *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	spec.SetNamespace(c.Namespace)

	istio, err := istioversionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(spec.GetNamespace()).Create(spec)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(existing, updated *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	updated.SetNamespace(c.Namespace)
	updated.SetResourceVersion(existing.GetResourceVersion())

	istio, err := istioversionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(updated.GetNamespace()).Update(updated)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(spec *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	existing, err := c.GetVirtualService(spec.GetName())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateVirtualService(spec)
	}
	return c.UpdateVirtualService(existing, spec)
}

func (c *Client) GetVirtualService(name string) (*istioclientnetworking.VirtualService, error) {
	istio, err := istioversionedclient.NewForConfig(c.RestConfig)
	virtualService, err := istio.NetworkingV1alpha3().VirtualServices(c.Namespace).Get(name, v1.GetOptions{
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
	istio, err := istioversionedclient.NewForConfig(c.RestConfig)
	err = istio.NetworkingV1alpha3().VirtualServices(c.Namespace).Delete(name, &v1.DeleteOptions{
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

func (c *Client) ListVirtualServices(opts *v1.ListOptions) ([]istioclientnetworking.VirtualService, error) {
	if opts == nil {
		opts = &v1.ListOptions{}
	}
	istio, err := istioversionedclient.NewForConfig(c.RestConfig)
	vsList, err := istio.NetworkingV1alpha3().VirtualServices(c.Namespace).List(v1.ListOptions{
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

func (c *Client) ListVirtualServicesByLabels(labels map[string]string) ([]istioclientnetworking.VirtualService, error) {
	opts := &v1.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListVirtualServices(opts)
}

func (c *Client) ListVirtualServicesByLabel(labelKey string, labelValue string) ([]istioclientnetworking.VirtualService, error) {
	return c.ListVirtualServicesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListVirtualServicesWithLabelKeys(labelKeys ...string) ([]istioclientnetworking.VirtualService, error) {
	opts := &v1.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListVirtualServices(opts)
}

func ExtractVirtualServiceGateways(virtualService *istioclientnetworking.VirtualService) (strset.Set, error) {
	return strset.FromSlice(virtualService.Spec.Gateways), nil
}

func ExtractVirtualServiceEndpoints(virtualService *istioclientnetworking.VirtualService) (strset.Set, error) {
	endpoints := strset.New()
	httpRoutes := virtualService.Spec.Http
	for _, http := range httpRoutes {
		for _, match := range http.Match {
			endpoints.Add(urls.CanonicalizeEndpoint(match.Uri.GetExact()))
		}
	}
	return endpoints, nil
}
