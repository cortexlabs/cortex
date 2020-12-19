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

package tensorflow

// Value to set for TF_CPP_MIN_LOG_LEVEL environment variable. Default is 0.
func NumericLogLevelFromLogLevel(logLevel string) int {
	var tfLogLevelNumber int
	switch logLevel {
	case "debug":
		tfLogLevelNumber = 0
	case "info":
		tfLogLevelNumber = 0
	case "warning":
		tfLogLevelNumber = 1
	case "error":
		tfLogLevelNumber = 2
	}
	return tfLogLevelNumber
}
