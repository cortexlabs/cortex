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
package local

import (
	"fmt"

	"github.com/cortexlabs/cortex/cli/types/flags"
)

// Can be overwritten by CLI commands
var OutputType flags.OutputType = flags.PrettyOutputType

func localPrintln(a ...interface{}) {
	if OutputType != flags.JSONOutputType {
		fmt.Println(a...)
	}
}

func localPrint(a ...interface{}) {
	if OutputType != flags.JSONOutputType {
		fmt.Print(a...)
	}
}

func localPrintf(format string, a ...interface{}) {
	if OutputType != flags.JSONOutputType {
		fmt.Printf(format, a...)
	}
}
