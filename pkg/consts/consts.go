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
	"fmt"
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var (
	CortexVersion      = "master" // CORTEX_VERSION
	CortexVersionMinor = "master" // CORTEX_VERSION_MINOR

	SingleModelName = "_cortex_default"

	DefaultImagePythonPredictorCPU   = defaultDockerImage("python-predictor-cpu")
	DefaultImagePythonPredictorGPU   = defaultDockerImage("python-predictor-gpu")
	DefaultImagePythonPredictorInf   = defaultDockerImage("python-predictor-inf")
	DefaultImageTensorFlowServingCPU = defaultDockerImage("tensorflow-serving-cpu")
	DefaultImageTensorFlowServingGPU = defaultDockerImage("tensorflow-serving-gpu")
	DefaultImageTensorFlowServingInf = defaultDockerImage("tensorflow-serving-inf")
	DefaultImageTensorFlowPredictor  = defaultDockerImage("tensorflow-predictor")
	DefaultImageONNXPredictorCPU     = defaultDockerImage("onnx-predictor-cpu")
	DefaultImageONNXPredictorGPU     = defaultDockerImage("onnx-predictor-gpu")
	DefaultImagePathsSet             = strset.New(
		DefaultImagePythonPredictorCPU,
		DefaultImagePythonPredictorGPU,
		DefaultImagePythonPredictorInf,
		DefaultImageTensorFlowServingCPU,
		DefaultImageTensorFlowServingGPU,
		DefaultImageTensorFlowServingInf,
		DefaultImageTensorFlowPredictor,
		DefaultImageONNXPredictorCPU,
		DefaultImageONNXPredictorGPU,
	)

	MaxClassesPerMonitoringRequest = 20 // cloudwatch.GeMetricData can get up to 100 metrics per request, avoid multiple requests and have room for other stats
	DashboardTitle                 = "# cortex monitoring dashboard"
	DefaultMaxReplicaConcurrency   = int64(1024)
	NeuronCoresPerInf              = int64(4)
)

func defaultDockerImage(imageName string) string {
	// override default image paths in development
	if imageOverride := os.Getenv("CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY"); imageOverride != "" {
		return fmt.Sprintf("%s/%s:latest", imageOverride, imageName)
	}

	return fmt.Sprintf("cortexlabs/%s:%s", imageName, CortexVersion)
}
