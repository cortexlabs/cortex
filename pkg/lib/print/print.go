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

package print

import (
	"fmt"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/console"
)

var _maxBoldLength = 150

func BoldFirstLine(msg string) {
	msgParts := strings.Split(msg, "\n")

	if len(msgParts[0]) > _maxBoldLength {
		fmt.Println(msg)
		return
	}

	fmt.Println(console.Bold(msgParts[0]))

	if len(msgParts) > 1 {
		fmt.Println(strings.Join(msgParts[1:], "\n"))
	}
}

func StderrBoldFirstLine(msg string) {
	msgParts := strings.Split(msg, "\n")

	if len(msgParts[0]) > _maxBoldLength {
		StderrPrintln(msg)
		return
	}

	StderrPrintln(console.Bold(msgParts[0]))

	if len(msgParts) > 1 {
		StderrPrintln(strings.Join(msgParts[1:], "\n"))
	}
}

func BoldFirstBlock(msg string) {
	msgParts := strings.Split(msg, "\n\n")

	if len(msgParts[0]) > _maxBoldLength {
		fmt.Println(msg)
		return
	}

	fmt.Println(console.Bold(msgParts[0]))

	if len(msgParts) > 1 {
		fmt.Println("\n" + strings.Join(msgParts[1:], "\n\n"))
	}
}

func StderrBoldFirstBlock(msg string) {
	msgParts := strings.Split(msg, "\n\n")

	if len(msgParts[0]) > _maxBoldLength {
		StderrPrintln(msg)
		return
	}

	StderrPrintln(console.Bold(msgParts[0]))

	if len(msgParts) > 1 {
		StderrPrintln("\n" + strings.Join(msgParts[1:], "\n\n"))
	}
}

func Dot() error {
	fmt.Print(".")
	return nil
}

func StderrPrintln(str string) {
	os.Stderr.WriteString(str + "\n")
}
