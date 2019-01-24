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

package util

import (
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

// Alternative: https://golang.org/pkg/sync/#WaitGroup (with error channel)
// Alternative: https://godoc.org/golang.org/x/sync/errgroup

func RunInParallel(fns ...func() error) []error {
	if len(fns) == 0 {
		return nil
	}

	errChannels := make([]chan error, len(fns))
	for i := range errChannels {
		errChannels[i] = make(chan error)
	}

	for i := range fns {
		fn := fns[i]
		errChannel := errChannels[i]
		go func() {
			errChannel <- fn()
		}()
	}

	errors := make([]error, len(fns))
	for i := range fns {
		errors[i] = <-errChannels[i]
	}
	return errors
}

func RunInParallelFirstErr(fns ...func() error) error {
	errs := RunInParallel(fns...)
	return errors.FirstError(errs...)
}
