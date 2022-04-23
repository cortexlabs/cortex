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

package consts

import (
	"os"
	"time"

	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var (
	CortexVersion      = "master" // CORTEX_VERSION
	CortexVersionMinor = "master" // CORTEX_VERSION_MINOR

	DefaultNamespace    = "default"
	KubeSystemNamespace = "kube-system"
	IstioNamespace      = "istio-system"
	PrometheusNamespace = "prometheus"
	LoggingNamespace    = "logging"

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

	/*
		CPU Pod Reservations:
		- FluentBit 100
		- NodeExporter 50 (it has two containers)
		- KubeProxy 100
		- AWS cni 10
	*/
	CortexCPUPodReserved = kresource.MustParse("260m")
	/*
		CPU Node Reservations:
		- Reserved (150 + 150) see generate_eks.py for details
	*/
	CortexCPUK8sReserved = kresource.MustParse("300m")

	/*
		Memory Pod Reservations:
		- FluentBit 150
		- NodeExporter 200 (it has two containers)
	*/
	CortexMemPodReserved = kresource.MustParse("350Mi")
	/*
		Memory Node Reservations:
		- Reserved (300 + 300 + 200) see generate_eks.py for details
	*/
	CortexMemK8sReserved = kresource.MustParse("800Mi")

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
	CortexQueueURLHeader      = "X-Cortex-Queue-URL"

	WaitForReadyReplicasTimeout = 20 * time.Minute
)

func DefaultRegistry() string {
	if registryOverride := os.Getenv("CORTEX_DEV_DEFAULT_IMAGE_REGISTRY"); registryOverride != "" {
		return registryOverride
	}
	return "quay.io/cortexlabs"
}
