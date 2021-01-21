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

package parallel

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

// Alternative: https://golang.org/pkg/sync/#WaitGroup (with error channel)
// Alternative: https://godoc.org/golang.org/x/sync/errgroup

func Run(fn func() error, fns ...func() error) []error {
	allFns := append(fns, fn)

	errChannels := make([]chan error, len(allFns))
	for i := range errChannels {
		errChannels[i] = make(chan error)
	}

	for i := range allFns {
		fn := allFns[i]
		errChannel := errChannels[i]

		if fn == nil {
			errChannel <- nil
			continue
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errChannel <- errors.CastRecoverError(r)
				}
			}()
			errChannel <- fn()
		}()
	}

	errors := make([]error, len(allFns))
	for i := range allFns {
		errors[i] = <-errChannels[i]
	}
	return errors
}

func RunFirstErr(fn func() error, fns ...func() error) error {
	errs := Run(fn, fns...)
	return errors.FirstError(errs...)
}
