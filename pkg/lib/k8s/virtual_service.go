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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _virtualServiceTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1alpha3",
	Kind:       "VirtualService",
}

type VirtualServiceSpec struct {
	Name        string
	Gateways    []string
	ServiceName string
	ServicePort int32
	ExactPath   *string
	PrefixPath  *string
	Rewrite     *string
	Labels      map[string]string
	Annotations map[string]string
}

func VirtualService(spec *VirtualServiceSpec) *istioclientnetworking.VirtualService {
	var stringMatch *istionetworking.StringMatch
	// 		uriMap["prefix"] = urls.CanonicalizeEndpoint(*spec.PrefixPath)

	if spec.ExactPath != nil {
		stringMatch = &istionetworking.StringMatch{
			MatchType: &istionetworking.StringMatch_Exact{
				Exact: urls.CanonicalizeEndpoint(*spec.ExactPath),
			},
		}
	} else {
		stringMatch = &istionetworking.StringMatch{
			MatchType: &istionetworking.StringMatch_Prefix{
				Prefix: urls.CanonicalizeEndpoint(*spec.PrefixPath),
			},
		}
	}

	virtualService := &istioclientnetworking.VirtualService{
		TypeMeta: _virtualServiceTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: istionetworking.VirtualService{
			Hosts:    []string{"*"},
			Gateways: spec.Gateways,
			Http: []*istionetworking.HTTPRoute{
				{
					Match: []*istionetworking.HTTPMatchRequest{
						{
							Uri: stringMatch,
						},
					},
					Route: []*istionetworking.HTTPRouteDestination{
						{
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

	var path string

	if spec.ExactPath != nil {
		path = *spec.ExactPath
	} else {
		path = *spec.PrefixPath
	}

	// TODO what happens if rewrite path == PrefixPath or ExactPath?
	if spec.Rewrite != nil && urls.CanonicalizeEndpoint(*spec.Rewrite) != urls.CanonicalizeEndpoint(path) {
		virtualService.Spec.Http[0].Rewrite = &istionetworking.HTTPRewrite{
			Uri: urls.CanonicalizeEndpoint(*spec.Rewrite),
		}
	}

	return virtualService
}

func (c *Client) CreateVirtualService(virtualService *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	virtualService.TypeMeta = _virtualServiceTypeMeta
	virtualService, err := c.virtualServiceClient.Create(virtualService)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(existing, updated *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	updated.TypeMeta = _virtualServiceTypeMeta
	updated.ResourceVersion = existing.ResourceVersion

	virtualService, err := c.virtualServiceClient.Update(updated)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) ApplyVirtualService(virtualService *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	existing, err := c.GetVirtualService(virtualService.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreateVirtualService(virtualService)
	}
	return c.UpdateVirtualService(existing, virtualService)
}

func (c *Client) GetVirtualService(name string) (*istioclientnetworking.VirtualService, error) {
	virtualService, err := c.virtualServiceClient.Get(name, kmeta.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	virtualService.TypeMeta = _virtualServiceTypeMeta
	return virtualService, nil
}

func (c *Client) DeleteVirtualService(name string) (bool, error) {
	err := c.virtualServiceClient.Delete(name, _deleteOpts)
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListVirtualServices(opts *kmeta.ListOptions) ([]istioclientnetworking.VirtualService, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	vsList, err := c.virtualServiceClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range vsList.Items {
		vsList.Items[i].TypeMeta = _virtualServiceTypeMeta
	}
	return vsList.Items, nil
}

func (c *Client) ListVirtualServicesByLabels(labels map[string]string) ([]istioclientnetworking.VirtualService, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListVirtualServices(opts)
}

func (c *Client) ListVirtualServicesByLabel(labelKey string, labelValue string) ([]istioclientnetworking.VirtualService, error) {
	return c.ListVirtualServicesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListVirtualServicesWithLabelKeys(labelKeys ...string) ([]istioclientnetworking.VirtualService, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListVirtualServices(opts)
}

func ExtractVirtualServiceGateways(virtualService *istioclientnetworking.VirtualService) strset.Set {
	return strset.FromSlice(virtualService.Spec.Gateways)
}

func ExtractVirtualServiceEndpoints(virtualService *istioclientnetworking.VirtualService) strset.Set {
	endpoints := strset.New()
	for _, http := range virtualService.Spec.Http {
		for _, match := range http.Match {
			endpoints.Add(urls.CanonicalizeEndpoint(match.Uri.GetExact()))
		}
	}
	return endpoints
}
