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
	CortexVersion = "master" // CORTEX_VERSION

	ContextCacheDir    = "/mnt/context"
	EmptyDirMountPath  = "/mnt"
	EmptyDirVolumeName = "mnt"

	CortexConfigPath = "/configs/cortex"
	CortexConfigName = "cortex-config"

	RequirementsTxt = "requirements.txt"
	PackageDir      = "packages"

	AppsDir             = "apps"
	DeploymentsDir      = "deployments"
	APIsDir             = "apis"
	PythonPackagesDir   = "python_packages"
	RequestHandlersDir  = "request_handlers"
	ProjectDir          = "projects"
	ContextsDir         = "contexts"
	ResourceStatusesDir = "resource_statuses"
	WorkloadSpecsDir    = "workload_specs"
	MetadataDir         = "metadata"

	TelemetryURL = "https://telemetry.cortexlabs.dev"

	MaxClassesPerRequest = 75 // cloudwatch.GeMetricData can get up to 100 metrics per request, avoid multiple requests and have room for other stats

	DefaultTFServingSignatureKey = "predict"
)
