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
	"reflect"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	istionetworking "istio.io/api/networking/v1beta1"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _virtualServiceTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1beta1",
	Kind:       "VirtualService",
}

type VirtualServiceSpec struct {
	Name         string
	Gateways     []string
	ExactPath    *string // either this or PrefixPath
	PrefixPath   *string // either this or ExactPath
	Destinations []Destination
	Rewrite      *string
	Labels       map[string]string
	Annotations  map[string]string
}

type Destination struct {
	ServiceName string
	Weight      int32
	Port        uint32
	Shadow      bool
}

func VirtualService(spec *VirtualServiceSpec) *istioclientnetworking.VirtualService {
	destinations := []*istionetworking.HTTPRouteDestination{}
	var mirror *istionetworking.Destination
	var mirrorWeight *istionetworking.Percent

	for _, destination := range spec.Destinations {
		if destination.Shadow {
			mirror = &istionetworking.Destination{
				Host: destination.ServiceName,
				Port: &istionetworking.PortSelector{
					Number: destination.Port,
				},
			}
			mirrorWeight = &istionetworking.Percent{Value: float64(destination.Weight)}
		} else {
			destinations = append(destinations, &istionetworking.HTTPRouteDestination{
				Destination: &istionetworking.Destination{
					Host: destination.ServiceName,
					Port: &istionetworking.PortSelector{
						Number: destination.Port,
					},
				},
				Weight: destination.Weight,
			})
		}
	}

	var httpRoutes []*istionetworking.HTTPRoute

	if spec.ExactPath != nil {
		httpRoutes = append(httpRoutes, &istionetworking.HTTPRoute{
			Match: []*istionetworking.HTTPMatchRequest{
				{
					Uri: &istionetworking.StringMatch{
						MatchType: &istionetworking.StringMatch_Exact{
							Exact: urls.CanonicalizeEndpoint(*spec.ExactPath),
						},
					},
				},
			},
			Route:            destinations,
			Mirror:           mirror,
			MirrorPercentage: mirrorWeight,
		})

		if spec.Rewrite != nil {
			httpRoutes[0].Rewrite = &istionetworking.HTTPRewrite{
				Uri: urls.CanonicalizeEndpoint(*spec.Rewrite),
			}
		}
	} else {
		exactMatch := &istionetworking.HTTPRoute{
			Match: []*istionetworking.HTTPMatchRequest{
				{
					Uri: &istionetworking.StringMatch{
						MatchType: &istionetworking.StringMatch_Exact{
							Exact: urls.CanonicalizeEndpoint(*spec.PrefixPath),
						},
					},
				},
			},
			Route:            destinations,
			Mirror:           mirror,
			MirrorPercentage: mirrorWeight,
		}

		prefixMatch := &istionetworking.HTTPRoute{
			Match: []*istionetworking.HTTPMatchRequest{
				{
					Uri: &istionetworking.StringMatch{
						MatchType: &istionetworking.StringMatch_Prefix{
							Prefix: urls.CanonicalizeEndpointWithTrailingSlash(*spec.PrefixPath),
						},
					},
				},
			},
			Route:            destinations,
			Mirror:           mirror,
			MirrorPercentage: mirrorWeight,
		}

		if spec.Rewrite != nil {
			exactMatch.Rewrite = &istionetworking.HTTPRewrite{
				Uri: urls.CanonicalizeEndpoint(*spec.Rewrite),
			}

			prefixMatch.Rewrite = &istionetworking.HTTPRewrite{
				Uri: urls.CanonicalizeEndpointWithTrailingSlash(*spec.Rewrite),
			}
		}

		httpRoutes = append(httpRoutes, exactMatch, prefixMatch)
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
			Http:     httpRoutes,
		},
	}

	return virtualService
}

func (c *Client) CreateVirtualService(virtualService *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	virtualService.TypeMeta = _virtualServiceTypeMeta
	virtualService, err := c.virtualServiceClient.Create(context.Background(), virtualService, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return virtualService, nil
}

func (c *Client) UpdateVirtualService(existing, updated *istioclientnetworking.VirtualService) (*istioclientnetworking.VirtualService, error) {
	updated.TypeMeta = _virtualServiceTypeMeta
	updated.ResourceVersion = existing.ResourceVersion

	virtualService, err := c.virtualServiceClient.Update(context.Background(), updated, kmeta.UpdateOptions{})
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
	virtualService, err := c.virtualServiceClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	virtualService.TypeMeta = _virtualServiceTypeMeta
	return virtualService, nil
}

func (c *Client) DeleteVirtualService(name string) (bool, error) {
	err := c.virtualServiceClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListVirtualServices(opts *kmeta.ListOptions) ([]istioclientnetworking.VirtualService, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	vsList, err := c.virtualServiceClient.List(context.Background(), *opts)
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
			if match.Uri.GetExact() != "" {
				endpoints.Add(urls.CanonicalizeEndpoint(match.Uri.GetExact()))
			}

			if match.Uri.GetPrefix() != "" {
				endpoints.Add(urls.CanonicalizeEndpoint(match.Uri.GetPrefix()))
			}
		}
	}
	return endpoints
}

func VirtualServicesMatch(vs1, vs2 istionetworking.VirtualService) bool {
	if !strset.New(vs1.Hosts...).IsEqual(strset.New(vs2.Hosts...)) {
		return false
	}

	if !strset.New(vs1.Gateways...).IsEqual(strset.New(vs2.Gateways...)) {
		return false
	}

	if !strset.New(vs1.ExportTo...).IsEqual(strset.New(vs2.ExportTo...)) {
		return false
	}

	if !reflect.DeepEqual(vs1.Http, vs2.Http) {
		return false
	}

	return true
}
