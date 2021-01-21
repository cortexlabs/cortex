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

//
// These tests must be run and verified manually:
// go test github.com/cortexlabs/cortex/pkg/lib/parallel -run TestRunInParallel -v
//

// import (
// 	"testing"

// 	"github.com/stretchr/testify/require"

// 	"github.com/cortexlabs/cortex/utils/util"
// )

// func delayAndPrint(jobID string, secs int) error {
// 	time.Sleep(time.Duration(secs) * time.Second)
// 	t := time.Now()
// 	fmt.Printf("Done with job %s at %d:%d:%s\n", jobID, t.Minute(), t.Second(), util.MillisecsStr(t))
// 	return nil
// }

// func delayAndError(jobID string, secs int) error {
// 	delayAndPrint(jobID, secs)
// 	return errors.New("Error in job " + jobID)
// }

// func TestRunInParallel(t *testing.T) {
// 	var errs []error
// 	var err error

// 	errs = util.RunInParallel(
// 		func() error {
// 			return delayAndPrint("1", 1)
// 		},
// 		func() error {
// 			return delayAndPrint("2", 2)
// 		},
// 		func() error {
// 			return delayAndPrint("3", 3)
// 		},
// 	)
// 	err = util.FirstError(errs...)
// 	require.NoError(t, err)

// 	errs = util.RunInParallel(
// 		func() error {
// 			return delayAndPrint("3", 3)
// 		},
// 		func() error {
// 			return delayAndPrint("2", 2)
// 		},
// 		func() error {
// 			return delayAndPrint("1", 1)
// 		},
// 	)
// 	err = util.FirstError(errs...)
// 	require.NoError(t, err)

// 	errs = util.RunInParallel(
// 		func() error {
// 			return delayAndError("3", 3)
// 		},
// 		func() error {
// 			return delayAndError("2", 2)
// 		},
// 		func() error {
// 			return delayAndError("1", 1)
// 		},
// 	)
// 	expectedErrs := []error{
// 		errors.New("Error in job 3"),
// 		errors.New("Error in job 2"),
// 		errors.New("Error in job 1"),
// 	}
// 	require.Equal(t, expectedErrs, errs)
// }
