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

package operator

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ClusterProjectFiles struct {
	ProjectByteMap map[string][]byte
}

func (cpf ClusterProjectFiles) GetAllPaths() []string {
	files := make([]string, len(cpf.ProjectByteMap))

	i := 0
	for path := range cpf.ProjectByteMap {
		files[i] = path
	}

	return files
}

func (cpf ClusterProjectFiles) GetFile(fileName string) ([]byte, error) {
	bytes, ok := cpf.ProjectByteMap[fileName]
	if !ok {
		return nil, files.ErrorFileDoesNotExist(fileName)
	}

	return bytes, nil
}

func ValidateClusterAPIs(apis []userconfig.API, projectFileMap map[string][]byte) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	virtualServices, maxMem, err := getValidationK8sResources()
	if err != nil {
		return err
	}

	projectFiles := ClusterProjectFiles{
		ProjectByteMap: projectFileMap,
	}

	for i := range apis {
		if err := spec.ValidateAPI(&apis[i], projectFiles, types.AWSProviderType, config.AWS); err != nil {
			return err
		}
		if err := validateK8s(&apis[i], config.Cluster, virtualServices, maxMem); err != nil {
			return err
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

func validateK8s(api *userconfig.API,
	config *clusterconfig.InternalConfig,
	virtualServices []kunstructured.Unstructured,
	maxMem *kresource.Quantity) error {
	if err := validateCompute(api.Compute, config, maxMem); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.ComputeKey)
	}

	if err := validateEndpointCollisions(api, virtualServices); err != nil {
		return err
	}

	return nil
}

func validateCompute(compute *userconfig.Compute, config *clusterconfig.InternalConfig, maxMem *kresource.Quantity) error {
	maxMem.Sub(_cortexMemReserve)

	maxCPU := config.InstanceMetadata.CPU
	maxCPU.Sub(_cortexCPUReserve)

	maxGPU := config.InstanceMetadata.GPU
	if maxGPU > 0 {
		// Reserve resources for nvidia device plugin daemonset
		maxCPU.Sub(_nvidiaCPUReserve)
		maxMem.Sub(_nvidiaMemReserve)
	}

	if maxCPU.Cmp(compute.CPU.Quantity) < 0 {
		return ErrorNoAvailableNodeComputeLimit("CPU", compute.CPU.String(), maxCPU.String())
	}
	if compute.Mem != nil {
		if maxMem.Cmp(compute.Mem.Quantity) < 0 {
			return ErrorNoAvailableNodeComputeLimit("memory", compute.Mem.String(), maxMem.String())
		}
	}
	if compute.GPU > maxGPU {
		return ErrorNoAvailableNodeComputeLimit("GPU", fmt.Sprintf("%d", compute.GPU), fmt.Sprintf("%d", maxGPU))
	}
	return nil
}

func validateEndpointCollisions(api *userconfig.API, virtualServices []kunstructured.Unstructured) error {
	for _, virtualService := range virtualServices {
		gateways, err := k8s.ExtractVirtualServiceGateways(&virtualService)
		if err != nil {
			return err
		}
		if !gateways.Has("apis-gateway") {
			continue
		}

		endpoints, err := k8s.ExtractVirtualServiceEndpoints(&virtualService)
		if err != nil {
			return err
		}

		for endpoint := range endpoints {
			if s.EnsureSuffix(endpoint, "/") == s.EnsureSuffix(*api.Endpoint, "/") && virtualService.GetLabels()["apiName"] != api.Name {
				return errors.Wrap(spec.ErrorDuplicateEndpoint(virtualService.GetLabels()["apiName"]), api.Identify(), userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}

func findDuplicateEndpoints(apis []userconfig.API) []userconfig.API {
	endpoints := make(map[string][]userconfig.API)

	for _, api := range apis {
		endpoints[*api.Endpoint] = append(endpoints[*api.Endpoint], api)
	}

	for endpoint := range endpoints {
		if len(endpoints[endpoint]) > 1 {
			return endpoints[endpoint]
		}
	}

	return nil
}

func getValidationK8sResources() ([]kunstructured.Unstructured, *kresource.Quantity, error) {
	var virtualServices []kunstructured.Unstructured
	var maxMem *kresource.Quantity

	err := parallel.RunFirstErr(
		func() error {
			var err error
			virtualServices, err = config.K8s.ListVirtualServices(nil)
			return err
		},
		func() error {
			var err error
			maxMem, err = updateMemoryCapacityConfigMap()
			return err
		},
	)

	return virtualServices, maxMem, err
}
