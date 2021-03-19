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

package resources

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

type ProjectFiles struct {
	ProjectByteMap map[string][]byte
}

func (projectFiles ProjectFiles) AllPaths() []string {
	pFiles := make([]string, 0, len(projectFiles.ProjectByteMap))
	for path := range projectFiles.ProjectByteMap {
		pFiles = append(pFiles, path)
	}
	return pFiles
}

func (projectFiles ProjectFiles) GetFile(path string) ([]byte, error) {
	bytes, ok := projectFiles.ProjectByteMap[path]
	if !ok {
		return nil, files.ErrorFileDoesNotExist(path)
	}
	return bytes, nil
}

func (projectFiles ProjectFiles) HasFile(path string) bool {
	_, ok := projectFiles.ProjectByteMap[path]
	return ok
}

func (projectFiles ProjectFiles) HasDir(path string) bool {
	path = s.EnsureSuffix(path, "/")
	for projectFilePath := range projectFiles.ProjectByteMap {
		if strings.HasPrefix(projectFilePath, path) {
			return true
		}
	}
	return false
}

func ValidateClusterAPIs(apis []userconfig.API, projectFiles spec.ProjectFiles) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	virtualServices, err := config.K8s.ListVirtualServices(nil)
	if err != nil {
		return err
	}

	deployedRealtimeAPIs := strset.New()

	for _, virtualService := range virtualServices {
		if virtualService.Labels["apiKind"] == userconfig.RealtimeAPIKind.String() {
			deployedRealtimeAPIs.Add(virtualService.Labels["apiName"])
		}
	}

	realtimeAPIs := InclusiveFilterAPIsByKind(apis, userconfig.RealtimeAPIKind)

	for i := range apis {
		api := &apis[i]
		if api.Kind == userconfig.RealtimeAPIKind || api.Kind == userconfig.BatchAPIKind ||
			api.Kind == userconfig.TaskAPIKind || api.Kind == userconfig.AsyncAPIKind {

			if err := spec.ValidateAPI(api, nil, projectFiles, config.Provider, config.AWS, config.GCP, config.K8s); err != nil {
				return errors.Wrap(err, api.Identify())
			}

			if err := validateEndpointCollisions(api, virtualServices); err != nil {
				return err
			}
		}

		if api.Kind == userconfig.TrafficSplitterKind {
			if err := spec.ValidateTrafficSplitter(api); err != nil {
				return errors.Wrap(err, api.Identify())
			}
			if err := checkIfAPIExists(api.APIs, realtimeAPIs, deployedRealtimeAPIs); err != nil {
				return errors.Wrap(err, api.Identify())
			}
			if err := validateEndpointCollisions(api, virtualServices); err != nil {
				return errors.Wrap(err, api.Identify())
			}
		}
	}

	if config.IsManaged() && config.Provider == types.AWSProviderType {
		maxMemMap, err := operator.UpdateMemoryCapacityConfigMap()
		if err != nil {
			return err
		}

		for i := range apis {
			api := &apis[i]
			if api.Kind == userconfig.RealtimeAPIKind || api.Kind == userconfig.AsyncAPIKind || api.Kind == userconfig.BatchAPIKind || api.Kind == userconfig.TaskAPIKind {
				if err := awsManagedValidateK8sCompute(api.Compute, maxMemMap); err != nil {
					return err
				}
			}
		}
	}

	dups := spec.FindDuplicateNames(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}
	dups = findDuplicateEndpoints(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateEndpointInOneDeploy(dups)
	}

	return nil
}

/*
CPU Reservations:

FluentBit 100
StatsDExporter 100
NodeExporter 100
KubeProxy 100
AWS cni 10
Reserved (150 + 150) see eks.yaml for details
*/
var _cortexCPUReserve = kresource.MustParse("710m")

/*
Memory Reservations:

FluentBit 150
StatsDExporter 100
NodeExporter 180
Reserved (300 + 300 + 200) see eks.yaml for details
*/
var _cortexMemReserve = kresource.MustParse("1230Mi")

var _nvidiaCPUReserve = kresource.MustParse("100m")
var _nvidiaMemReserve = kresource.MustParse("100Mi")

var _nvidiaDCGMExporterCPUReserve = kresource.MustParse("50m")
var _nvidiaDCGMExporterMemReserve = kresource.MustParse("50Mi")

var _inferentiaCPUReserve = kresource.MustParse("100m")
var _inferentiaMemReserve = kresource.MustParse("100Mi")

func awsManagedValidateK8sCompute(compute *userconfig.Compute, maxMemMap map[string]kresource.Quantity) error {
	instancesMetadata := config.AWSInstancesMetadata()
	if len(instancesMetadata) == 0 {
		return errors.ErrorUnexpected("unable to find instance metadata; likely because this is not a cortex managed cluster")
	}

	allErrors := []error{}
	successfulLoops := 0
	for _, instanceMetadata := range instancesMetadata {
		maxMemLoop := maxMemMap[instanceMetadata.Type]
		maxMemLoop.Sub(_cortexMemReserve)

		maxCPU := instanceMetadata.CPU
		maxCPU.Sub(_cortexCPUReserve)

		maxGPU := instanceMetadata.GPU
		if maxGPU > 0 {
			// Reserve resources for nvidia device plugin daemonset
			maxCPU.Sub(_nvidiaCPUReserve)
			maxMemLoop.Sub(_nvidiaMemReserve)
			// Reserve resources for nvidia dcgm prometheus exporter
			maxCPU.Sub(_nvidiaDCGMExporterCPUReserve)
			maxMemLoop.Sub(_nvidiaDCGMExporterMemReserve)
		}

		maxInf := instanceMetadata.Inf
		if maxInf > 0 {
			// Reserve resources for inferentia device plugin daemonset
			maxCPU.Sub(_inferentiaCPUReserve)
			maxMemLoop.Sub(_inferentiaMemReserve)
		}

		loopErrors := []error{}
		if compute.CPU != nil && maxCPU.Cmp(compute.CPU.Quantity) < 0 {
			loopErrors = append(loopErrors, ErrorNoAvailableNodeComputeLimit("CPU", compute.CPU.String(), maxCPU.String()))
		}
		if compute.Mem != nil && maxMemLoop.Cmp(compute.Mem.Quantity) < 0 {
			loopErrors = append(loopErrors, ErrorNoAvailableNodeComputeLimit("memory", compute.Mem.String(), maxMemLoop.String()))
		}
		if compute.GPU > maxGPU {
			loopErrors = append(loopErrors, ErrorNoAvailableNodeComputeLimit("GPU", fmt.Sprintf("%d", compute.GPU), fmt.Sprintf("%d", maxGPU)))
		}
		if compute.Inf > maxInf {
			loopErrors = append(loopErrors, ErrorNoAvailableNodeComputeLimit("Inf", fmt.Sprintf("%d", compute.Inf), fmt.Sprintf("%d", maxInf)))
		}
		if errors.HasError(loopErrors) {
			allErrors = append(allErrors, errors.FirstError(loopErrors...))
		} else {
			successfulLoops++
		}
	}

	if successfulLoops == 0 {
		return errors.FirstError(allErrors...)
	}

	return nil
}

func validateEndpointCollisions(api *userconfig.API, virtualServices []istioclientnetworking.VirtualService) error {
	for i := range virtualServices {
		virtualService := virtualServices[i]
		gateways := k8s.ExtractVirtualServiceGateways(&virtualService)
		if !gateways.Has("apis-gateway") {
			continue
		}

		endpoints := k8s.ExtractVirtualServiceEndpoints(&virtualService)
		for endpoint := range endpoints {
			if s.EnsureSuffix(endpoint, "/") == s.EnsureSuffix(*api.Networking.Endpoint, "/") && virtualService.Labels["apiName"] != api.Name {
				return errors.Wrap(spec.ErrorDuplicateEndpoint(virtualService.Labels["apiName"]), userconfig.NetworkingKey, userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}

func findDuplicateEndpoints(apis []userconfig.API) []userconfig.API {
	endpoints := make(map[string][]userconfig.API)

	for _, api := range apis {
		endpoints[*api.Networking.Endpoint] = append(endpoints[*api.Networking.Endpoint], api)
	}

	for endpoint := range endpoints {
		if len(endpoints[endpoint]) > 1 {
			return endpoints[endpoint]
		}
	}

	return nil
}

// InclusiveFilterAPIsByKind includes only provided Kinds
func InclusiveFilterAPIsByKind(apis []userconfig.API, kindsToInclude ...userconfig.Kind) []userconfig.API {
	kindsToIncludeSet := strset.New()
	for _, kind := range kindsToInclude {
		kindsToIncludeSet.Add(kind.String())
	}
	fileredAPIs := []userconfig.API{}
	for _, api := range apis {
		if kindsToIncludeSet.Has(api.Kind.String()) {
			fileredAPIs = append(fileredAPIs, api)
		}
	}
	return fileredAPIs
}

func ExclusiveFilterAPIsByKind(apis []userconfig.API, kindsToExclude ...userconfig.Kind) []userconfig.API {
	kindsToExcludeSet := strset.New()
	for _, kind := range kindsToExclude {
		kindsToExcludeSet.Add(kind.String())
	}
	fileredAPIs := []userconfig.API{}
	for _, api := range apis {
		if !kindsToExcludeSet.Has(api.Kind.String()) {
			fileredAPIs = append(fileredAPIs, api)
		}
	}
	return fileredAPIs
}

// checkIfAPIExists checks if referenced apis in trafficsplitter are either defined in yaml or already deployed
func checkIfAPIExists(trafficSplitterAPIs []*userconfig.TrafficSplit, apis []userconfig.API, deployedRealtimeAPIs strset.Set) error {
	var missingAPIs []string
	// check if apis named in trafficsplitter are either defined in same yaml or already deployed
	for _, trafficSplitAPI := range trafficSplitterAPIs {
		//check if already deployed
		deployed := deployedRealtimeAPIs.Has(trafficSplitAPI.Name)

		// check defined apis
		for _, definedAPI := range apis {
			if trafficSplitAPI.Name == definedAPI.Name {
				deployed = true
			}
		}
		if deployed == false {
			missingAPIs = append(missingAPIs, trafficSplitAPI.Name)
		}
	}
	if len(missingAPIs) != 0 {
		return ErrorAPIsNotDeployed(missingAPIs)
	}
	return nil

}
