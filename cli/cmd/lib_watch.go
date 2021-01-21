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

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
)

func getTerminalWidth() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	dimensions := strings.Split(strings.TrimSpace(string(out)), " ")
	if len(dimensions) != 2 {
		return 0
	}
	widthStr := dimensions[1]
	width, ok := s.ParseInt(widthStr)
	if !ok {
		return 0
	}
	return width
}

func watchHeader() string {
	timeStr := libtime.LocalHourNow()
	width := getTerminalWidth()
	numExtraChars := 4
	padding := strings.Repeat(" ", libmath.MaxInt(width-len(_cmdStr)-len(timeStr)-numExtraChars, 0))
	return fmt.Sprintf("$ %s  %s%s", _cmdStr, padding, libtime.LocalHourNow())
}

func rerun(f func() (string, error)) {
	if _flagWatch {
		print("\033[H\033[2J") // clear the screen

		var prevStrSlice []string

		for true {
			nextStr, err := f()
			if err != nil {
				fmt.Println()
				exit.Error(err)
			}

			nextStr = watchHeader() + "\n\n" + s.EnsureSingleTrailingNewLine(nextStr)
			nextStrSlice := strings.Split(nextStr, "\n")

			terminalWidth := getTerminalWidth()
			if terminalWidth <= 0 {
				exit.Error(ErrorNoTerminalWidth())
			}

			nextNumLines := 0
			for _, strLine := range nextStrSlice {
				nextNumLines += (len(strLine)-1)/terminalWidth + 1
			}
			prevNumLines := 0
			for _, strLine := range prevStrSlice {
				prevNumLines += (len(strLine)-1)/terminalWidth + 1
			}

			for i := prevNumLines; i > nextNumLines; i-- {
				fmt.Printf("\033[%dA\033[2K", 1) // move the cursor up and clear the line
			}

			for i := 0; i < prevNumLines; i++ {
				fmt.Printf("\033[%dA", 1) // move the cursor up
			}

			for _, strLine := range nextStrSlice {
				fmt.Printf("\033[2K%s\n", strLine) // clear the line and print the new line
			}

			prevStrSlice = nextStrSlice

			time.Sleep(time.Second * 2)
		}
	} else {
		str, err := f()
		if err != nil {
			exit.Error(err)
		}
		fmt.Print(s.EnsureSingleTrailingNewLine(str))
	}
}
