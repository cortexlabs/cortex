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

package cron

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type Cron struct {
	cronRun    chan struct{}
	cronCancel chan struct{}
}

func Run(f func() error, errHandler func(error), interval time.Duration) Cron {
	cronRun = make(chan struct{}, 1)
	cancelChannel = make(chan struct{}, 1)

	runCron := func() {
		defer recoverer(errHandler)
		err := f()
		if err != nil {
			errHandler(err)
		}
	}

	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()
		for {
			select {
			case <-cancelChannel:
				// cleanup?
				return
			case <-cronRun:
				runCron()
			case <-timer.C:
				runCron()
			}
			timer.Reset(interval)
		}
	}()

	return Cron{
		cronRun:       cronRun,
		cancelChannel: cancelChannel,
	}
}

func (c *Cron) RunNow() {
	c.cronRun <- struct{}{}
}

func (c *Cron) Cancel() {
	c.cancelChannel <- struct{}{}
}

func recoverer(errHandler func(error)) {
	if errInterface := recover(); errInterface != nil {
		errHandler(errors.CastRecoverError(errInterface))
	}
}
