/*
Copyright 2022 Cortex Labs, Inc.

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

package trafficsplitter

import (
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
)

func virtualServiceSpec(trafficSplitter *spec.API) *istioclientnetworking.VirtualService {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:         workloads.K8sName(trafficSplitter.Name),
		Gateways:     []string{"apis-gateway"},
		Destinations: getTrafficSplitterDestinations(trafficSplitter),
		ExactPath:    trafficSplitter.Networking.Endpoint,
		Rewrite:      pointer.String("/"),
		Retries:      pointer.Int32(0),
		Annotations:  trafficSplitter.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":               trafficSplitter.Name,
			"apiKind":               trafficSplitter.Kind.String(),
			"apiID":                 trafficSplitter.ID,
			"specID":                trafficSplitter.SpecID,
			"initialDeploymentTime": s.Int64(trafficSplitter.InitialDeploymentTime),
			"cortex.dev/api":        "true",
		},
	})
}
