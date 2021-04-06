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
	CortexVersion      = "master" // CORTEX_VERSION
	CortexVersionMinor = "master" // CORTEX_VERSION_MINOR

	SingleModelName = "_cortex_default"

	DefaultImagePythonPredictorCPU   = fmt.Sprintf("%s/python-predictor-cpu:%s", DefaultRegistry(), CortexVersion)
	DefaultImagePythonPredictorGPU   = fmt.Sprintf("%s/python-predictor-gpu:%s-cuda10.2-cudnn8", DefaultRegistry(), CortexVersion)
	DefaultImagePythonPredictorInf   = fmt.Sprintf("%s/python-predictor-inf:%s", DefaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingCPU = fmt.Sprintf("%s/tensorflow-serving-cpu:%s", DefaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingGPU = fmt.Sprintf("%s/tensorflow-serving-gpu:%s", DefaultRegistry(), CortexVersion)
	DefaultImageTensorFlowServingInf = fmt.Sprintf("%s/tensorflow-serving-inf:%s", DefaultRegistry(), CortexVersion)
	DefaultImageTensorFlowPredictor  = fmt.Sprintf("%s/tensorflow-predictor:%s", DefaultRegistry(), CortexVersion)
	DefaultImagePathsSet             = strset.New(
		DefaultImagePythonPredictorCPU,
		DefaultImagePythonPredictorGPU,
		DefaultImagePythonPredictorInf,
		DefaultImageTensorFlowServingCPU,
		DefaultImageTensorFlowServingGPU,
		DefaultImageTensorFlowServingInf,
		DefaultImageTensorFlowPredictor,
	)

	DefaultMaxReplicaConcurrency = int64(1024)
	NeuronCoresPerInf            = int64(4)
	AuthHeader                   = "X-Cortex-Authorization"
)

func DefaultRegistry() string {
	if registryOverride := os.Getenv("CORTEX_DEV_DEFAULT_IMAGE_REGISTRY"); registryOverride != "" {
		return registryOverride
	}
	return "quay.io/cortexlabs"
}
