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

package resources

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

type ProjectFiles struct {
	ProjectByteMap map[string][]byte
	ConfigFilePath string
}

func (projectFiles ProjectFiles) GetAllPaths() []string {
	files := make([]string, len(projectFiles.ProjectByteMap))

	i := 0
	for path := range projectFiles.ProjectByteMap {
		files[i] = path
		i++
	}

	return files
}

func (projectFiles ProjectFiles) GetFile(fileName string) ([]byte, error) {
	bytes, ok := projectFiles.ProjectByteMap[fileName]
	if !ok {
		return nil, files.ErrorFileDoesNotExist(fileName)
	}

	return bytes, nil
}

func (projectFiles ProjectFiles) GetConfigFilePath() string {
	return projectFiles.ConfigFilePath
}

func ValidateClusterAPIs(apis []userconfig.API, projectFiles spec.ProjectFiles) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	virtualServices, maxMem, err := getValidationK8sResources()
	if err != nil {
		return err
	}

	didPrintWarning := false

	withoutAPISplitter := ApisWithoutAPISplitter(apis)
	for i := range apis {
		api := &apis[i]
		fmt.Println(api)
		if api.Kind == userconfig.SyncAPIKind {
			if err := spec.ValidateAPI(api, projectFiles, types.AWSProviderType, config.AWS); err != nil {
				return err
			}
			if err := validateK8s(api, virtualServices, maxMem); err != nil {
				return err
			}

			if !didPrintWarning && api.Networking.LocalPort != nil {
				fmt.Println(fmt.Sprintf("warning: %s will be ignored because it is not supported in an environment using aws provider\n", userconfig.LocalPortKey))
				didPrintWarning = true
			}
		}
		if api.Kind == userconfig.APISplitterKind {
			_, err = isWeight100(api.APIs)
			if err != nil {
				return err
			}
			_, err = checkIfAPIExist(api.APIs, withoutAPISplitter)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("all single api checks done")
	dups := spec.FindDuplicateNames(withoutAPISplitter)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}

	dups = findDuplicateEndpoints(withoutAPISplitter)
	if len(dups) > 0 {
		return spec.ErrorDuplicateEndpointInOneDeploy(dups)
	}

	return nil
}

func validateK8s(api *userconfig.API, virtualServices []istioclientnetworking.VirtualService, maxMem *kresource.Quantity) error {
	if err := validateK8sCompute(api.Compute, maxMem); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.ComputeKey)
	}

	if err := validateEndpointCollisions(api, virtualServices); err != nil {
		return err
	}

	return nil
}

/*
CPU Reservations:

FluentD 200
StatsD 100
KubeProxy 100
AWS cni 10
Reserved (150 + 150) see eks.yaml for details
*/
var _cortexCPUReserve = kresource.MustParse("710m")

/*
Memory Reservations:

FluentD 200
StatsD 100
Reserved (300 + 300 + 200) see eks.yaml for details
*/
var _cortexMemReserve = kresource.MustParse("1100Mi")

var _nvidiaCPUReserve = kresource.MustParse("100m")
var _nvidiaMemReserve = kresource.MustParse("100Mi")

var _inferentiaCPUReserve = kresource.MustParse("100m")
var _inferentiaMemReserve = kresource.MustParse("100Mi")

func validateK8sCompute(compute *userconfig.Compute, maxMem *kresource.Quantity) error {
	maxMem.Sub(_cortexMemReserve)

	maxCPU := config.Cluster.InstanceMetadata.CPU
	maxCPU.Sub(_cortexCPUReserve)

	maxGPU := config.Cluster.InstanceMetadata.GPU
	if maxGPU > 0 {
		// Reserve resources for nvidia device plugin daemonset
		maxCPU.Sub(_nvidiaCPUReserve)
		maxMem.Sub(_nvidiaMemReserve)
	}

	maxInf := config.Cluster.InstanceMetadata.Inf
	if maxInf > 0 {
		// Reserve resources for inferentia device plugin daemonset
		maxCPU.Sub(_inferentiaCPUReserve)
		maxMem.Sub(_inferentiaMemReserve)
	}

	if compute.CPU != nil && maxCPU.Cmp(compute.CPU.Quantity) < 0 {
		return ErrorNoAvailableNodeComputeLimit("CPU", compute.CPU.String(), maxCPU.String())
	}
	if compute.Mem != nil && maxMem.Cmp(compute.Mem.Quantity) < 0 {
		return ErrorNoAvailableNodeComputeLimit("memory", compute.Mem.String(), maxMem.String())
	}
	if compute.GPU > maxGPU {
		return ErrorNoAvailableNodeComputeLimit("GPU", fmt.Sprintf("%d", compute.GPU), fmt.Sprintf("%d", maxGPU))
	}
	if compute.Inf > maxInf {
		return ErrorNoAvailableNodeComputeLimit("Inf", fmt.Sprintf("%d", compute.Inf), fmt.Sprintf("%d", maxInf))
	}
	return nil
}

func validateEndpointCollisions(api *userconfig.API, virtualServices []istioclientnetworking.VirtualService) error {
	for _, virtualService := range virtualServices {
		gateways := k8s.ExtractVirtualServiceGateways(&virtualService)
		if !gateways.Has("apis-gateway") {
			continue
		}

		endpoints := k8s.ExtractVirtualServiceEndpoints(&virtualService)
		for endpoint := range endpoints {
			if s.EnsureSuffix(endpoint, "/") == s.EnsureSuffix(*api.Networking.Endpoint, "/") && virtualService.Labels["apiName"] != api.Name {
				return errors.Wrap(spec.ErrorDuplicateEndpoint(virtualService.Labels["apiName"]), api.Identify(), userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}

func findDuplicateEndpoints(apis []userconfig.API) []userconfig.API {
	endpoints := make(map[string][]userconfig.API)

	for _, api := range apis {
		if api.Kind == userconfig.SyncAPIKind {
			endpoints[*api.Networking.Endpoint] = append(endpoints[*api.Networking.Endpoint], api)
		}
		if api.Kind == userconfig.APISplitterKind {
			endpoints[api.Endpoint] = append(endpoints[api.Endpoint], api)
		}

	}

	for endpoint := range endpoints {
		if len(endpoints[endpoint]) > 1 {
			return endpoints[endpoint]
		}
	}

	return nil
}

func getValidationK8sResources() ([]istioclientnetworking.VirtualService, *kresource.Quantity, error) {
	var virtualServices []istioclientnetworking.VirtualService
	var maxMem *kresource.Quantity

	err := parallel.RunFirstErr(
		func() error {
			var err error
			virtualServices, err = config.K8s.ListVirtualServices(nil)
			return err
		},
		func() error {
			var err error
			maxMem, err = operator.UpdateMemoryCapacityConfigMap()
			return err
		},
	)

	return virtualServices, maxMem, err
}

func ApisWithoutAPISplitter(apis []userconfig.API) []userconfig.API {
	withoutAPISplitter := []userconfig.API{}
	for _, api := range apis {
		if api.Kind != userconfig.APISplitterKind {
			withoutAPISplitter = append(withoutAPISplitter, api)
		}

	}
	return withoutAPISplitter
}

func ApisWithoutSyncAPI(apis []userconfig.API) []userconfig.API {
	withoutSyncAPI := []userconfig.API{}
	for _, api := range apis {
		if api.Kind != userconfig.SyncAPIKind {
			withoutSyncAPI = append(withoutSyncAPI, api)
		}

	}
	return withoutSyncAPI
}

func checkIfAPIExist(trafficSplitterAPIs []*userconfig.TrafficSplitter, apis []userconfig.API) (bool, error) {

	// TO DOOOOOO
	deployedAPIs, err := GetAPIs()
	if err != nil {
		return false, err
	}

	// check if apis named in trafficsplitter are either defined in same yaml or already deployed
	// OHHHH NO DOUBLE FOR LOOP
	for _, trafficSplitAPI := range trafficSplitterAPIs {
		deployed := false
		//check if already deployed
		for _, deployedAPI := range deployedAPIs.SyncAPIs {
			if trafficSplitAPI.Name == deployedAPI.Spec.Name {
				deployed = true
			}
		}
		// check defined apis
		for _, definedAPI := range apis {
			if trafficSplitAPI.Name == definedAPI.Name {
				deployed = true
			}
		}
		if deployed == false {
			return false, fmt.Errorf("unable to find apis specified in apisplitter")
		}
	}
	return true, nil

}

func isWeight100(apis []*userconfig.TrafficSplitter) (bool, error) {

	totalWeight := 0
	for _, api := range apis {
		totalWeight += api.Weight
	}
	if totalWeight == 100 {
		return true, nil
	}
	fmt.Println(totalWeight)
	return false, fmt.Errorf("api splitter weights added up to %d instead of 100", totalWeight)
}
