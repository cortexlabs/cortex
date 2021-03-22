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
	"fmt"
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var (
	CortexVersion      = "0.31.1" // CORTEX_VERSION
	CortexVersionMinor = "0.31"   // CORTEX_VERSION_MINOR

	SingleModelName = "_cortex_default"

	DefaultImagePythonPredictorCPU   = fmt.Sprintf("%s/python-predictor-cpu:%s", defaultRegistry(), CortexVersion)
	DefaultImagePythonPredictorGPU   = fmt.Sprintf("%s/python-predictor-gpu:%s-cuda10.2-cudnn8", defaultRegistry(), CortexVersion)
	DefaultImagePythonPredictorInf   = fmt.Sprintf("%s/python-predictor-inf:%s", defaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingCPU = fmt.Sprintf("%s/tensorflow-serving-cpu:%s", defaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingGPU = fmt.Sprintf("%s/tensorflow-serving-gpu:%s", defaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingInf = fmt.Sprintf("%s/tensorflow-serving-inf:%s", defaultRegistry(), CortexVersion)
	DefaultImageTensorFlowPredictor  = fmt.Sprintf("%s/tensorflow-predictor:%s", defaultRegistry(), CortexVersion)
	DefaultImageONNXPredictorCPU     = fmt.Sprintf("%s/onnx-predictor-cpu:%s", defaultRegistry(), CortexVersion)
	DefaultImageONNXPredictorGPU     = fmt.Sprintf("%s/onnx-predictor-gpu:%s", defaultRegistry(), CortexVersion)
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

	DefaultMaxReplicaConcurrency = int64(1024)
	NeuronCoresPerInf            = int64(4)
	AuthHeader                   = "X-Cortex-Authorization"
)

func defaultRegistry() string {
	if registryOverride := os.Getenv("CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY"); registryOverride != "" {
		return registryOverride
	}
	return "quay.io/cortexlabs"
}
