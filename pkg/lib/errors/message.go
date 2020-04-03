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

package errors

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/lib/print"
)

func PrintError(err error, strs ...string) {
	fmt.Println(errorStr(err, strs...))
	PrintStacktrace(err)
}

func PrintErrorForUser(err error, strs ...string) {
	print.BoldFirstLine(errorStr(err, strs...))
	PrintStacktrace(err)
}

func errorStr(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)
	return "error: " + strings.TrimSpace(Message(wrappedErr))
}

func Message(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)
	errStr := wrappedErr.Error()
	return strings.TrimSpace(errStr)
}

func MessageFirstLine(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)

	var errStr string
	if _, ok := CauseOrSelf(wrappedErr).(awserr.Error); ok {
		errStr = strings.Split(wrappedErr.Error(), "\n")[0]
	} else {
		errStr = wrappedErr.Error()
	}

	return strings.TrimSpace(errStr)
}
