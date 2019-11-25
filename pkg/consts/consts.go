/*
Copyright 2019 Cortex Labs, Inc.

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

var (
	CortexVersion = "0.11.0" // CORTEX_VERSION

	ContextCacheDir    = "/mnt/context"
	EmptyDirMountPath  = "/mnt"
	EmptyDirVolumeName = "mnt"

	ClusterConfigPath = "/configs/cluster/cluster.yaml"
	ClusterConfigName = "cluster-config"

	AppsDir             = "apps"
	DeploymentsDir      = "deployments"
	APIsDir             = "apis"
	RequestHandlersDir  = "request_handlers"
	ProjectsDir         = "projects"
	ContextsDir         = "contexts"
	ResourceStatusesDir = "resource_statuses"
	WorkloadSpecsDir    = "workload_specs"
	MetadataDir         = "metadata"

	K8sNamespace = "cortex"

	TelemetryURL = "https://telemetry.cortexlabs.dev"

	MaxClassesPerRequest = 20 // cloudwatch.GeMetricData can get up to 100 metrics per request, avoid multiple requests and have room for other stats
)
