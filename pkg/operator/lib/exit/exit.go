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

package exit

import (
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
)

var operatorLogger = logging.GetOperatorLogger()

func ErrorNoTelemetry(err error, wrapStrs ...string) {
	for _, str := range wrapStrs {
		err = errors.Wrap(err, str)
	}

	if err != nil && !errors.IsNoPrint(err) {
		operatorLogger.Error(err)
	}

	telemetry.Close()

	os.Exit(1)
}

func Error(err error, wrapStrs ...string) {
	for _, str := range wrapStrs {
		err = errors.Wrap(err, str)
	}

	if err != nil && !errors.IsNoTelemetry(err) {
		telemetry.Error(err)
	}

	if err != nil && !errors.IsNoPrint(err) {
		operatorLogger.Error(err)
	}

	telemetry.Close()

	os.Exit(1)
}
