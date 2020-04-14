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

package consts

import (
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var (
	CortexVersion      = "master" // CORTEX_VERSION
	CortexVersionMinor = "master" // CORTEX_VERSION_MINOR

	DefaultImagePythonServe    = "cortexlabs/python-serve:" + CortexVersion
	DefaultImagePythonServeGPU = "cortexlabs/python-serve-gpu:" + CortexVersion
	DefaultImageTFServe        = "cortexlabs/tf-serve:" + CortexVersion
	DefaultImageTFServeGPU     = "cortexlabs/tf-serve-gpu:" + CortexVersion
	DefaultImageTFAPI          = "cortexlabs/tf-api:" + CortexVersion
	DefaultImageONNXServe      = "cortexlabs/onnx-serve:" + CortexVersion
	DefaultImageONNXServeGPU   = "cortexlabs/onnx-serve-gpu:" + CortexVersion
	DefaultImagePathsSet       = strset.New(
		DefaultImagePythonServe,
		DefaultImagePythonServeGPU,
		DefaultImageTFServe,
		DefaultImageTFServeGPU,
		DefaultImageTFAPI,
		DefaultImageONNXServe,
		DefaultImageONNXServeGPU,
	)

	MaxClassesPerTrackerRequest = 20 // cloudwatch.GeMetricData can get up to 100 metrics per request, avoid multiple requests and have room for other stats
)
