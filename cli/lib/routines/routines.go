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

package routines

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

func RunWithPanicHandler(f func(), exitOnPanic bool) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := errors.CastRecoverError(r)
				errors.PrintStacktrace(err)
				if exitOnPanic {
					exit.Error(err)
				}
				telemetry.Error(err)
			}
		}()
		f()
	}()
}
