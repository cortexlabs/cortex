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

package exit

import (
	"fmt"
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

func Ok() {
	telemetry.Close()
	os.Exit(0)
}

func Error(err interface{}, errs ...interface{}) {
	mergedErr := errors.MergeErrItems(append(errs, err))

	if mergedErr == nil {
		fmt.Println("an error occurred")
		telemetry.Error(errors.New("an error occurred"))
	} else {
		errors.PrintError(mergedErr)
		telemetry.Error(mergedErr)
	}

	telemetry.Close()

	os.Exit(1)
}

func ErrorNoTelemetry(err interface{}, errs ...interface{}) {
	telemetry.Close()

	mergedErr := errors.MergeErrItems(append(errs, err))
	if mergedErr == nil {
		fmt.Println("an error occurred")
	} else {
		errors.PrintError(mergedErr)
	}

	os.Exit(1)
}

func ErrorNoPrint(errs ...interface{}) {
	mergedErr := errors.MergeErrItems(errs)

	if mergedErr == nil {
		telemetry.Error(errors.New("an error occurred"))
	} else {
		telemetry.Error(mergedErr)
	}

	telemetry.Close()

	os.Exit(1)
}

func ErrorNoPrintNoTelemetry() {
	telemetry.Close()
	os.Exit(1)
}

func Panic(errs ...interface{}) {
	err := errors.MergeErrItems(errs...)

	if err == nil {
		telemetry.Error(errors.New("a panic occurred"))
	} else {
		telemetry.Error(err)
	}

	telemetry.Close()

	if err == nil {
		panic("a panic occurred")
	} else {
		panic(err)
	}
}

func RecoverAndExit(strs ...string) {
	if errInterface := recover(); errInterface != nil {
		err := errors.CastRecoverError(errInterface, strs...)
		errors.PrintError(err)
		os.Exit(1)
	}
}
