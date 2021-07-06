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

package consts

import (
	"os"
	"time"

	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var (
	CortexVersion      = "0.38.0" // CORTEX_VERSION
	CortexVersionMinor = "0.38"   // CORTEX_VERSION_MINOR

	DefaultMaxQueueLength = int64(100)
	DefaultMaxConcurrency = int64(1)

	DefaultUserPodPortInt32 = int32(8080)

	ProxyPortStr   = "8888"
	ProxyPortInt32 = int32(8888)

	ActivatorName      = "activator"
	ActivatorPortInt32 = int32(8000)

	AdminPortName  = "admin"
	AdminPortStr   = "15000"
	AdminPortInt32 = int32(15000)

	AuthHeader = "X-Cortex-Authorization"

	CortexProxyCPU    = kresource.MustParse("100m")
	CortexProxyMem    = kresource.MustParse("100Mi")
	CortexDequeuerCPU = kresource.MustParse("100m")
	CortexDequeuerMem = kresource.MustParse("100Mi")

	DefaultInClusterConfigPath   = "/configs/cluster/cluster.yaml"
	MaxBucketLifecycleRules      = 100
	AsyncWorkloadsExpirationDays = int64(7)

	ReservedContainerPorts = []int32{
		ProxyPortInt32,
		AdminPortInt32,
	}
	ReservedContainerNames = []string{
		"dequeuer",
		"proxy",
	}

	UserAgentKey             = "User-Agent"
	KubeProbeUserAgentPrefix = "kube-probe/"

	CortexAPINameHeader       = "X-Cortex-API-Name"
	CortexTargetServiceHeader = "X-Cortex-Target-Service"
	CortexProbeHeader         = "X-Cortex-Probe"
	CortexOriginHeader        = "X-Cortex-Origin"

	WaitForInitializingReplicasTimeout = 15 * time.Minute
	WaitForReadyReplicasTimeout        = 20 * time.Minute
)

func DefaultRegistry() string {
	if registryOverride := os.Getenv("CORTEX_DEV_DEFAULT_IMAGE_REGISTRY"); registryOverride != "" {
		return registryOverride
	}
	return "quay.io/cortexlabs"
}
