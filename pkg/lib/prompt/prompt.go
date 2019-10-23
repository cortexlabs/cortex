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

package prompt

import (
	"fmt"
	"os"
	"strings"

	input "github.com/tcnksm/go-input"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

var ui = &input.UI{
	Writer: os.Stdout,
	Reader: os.Stdin,
}

type Options struct {
	Prompt        string
	DefaultStr    string
	HideDefault   bool
	MaskDefault   bool
	HideTyping    bool
	MaskTyping    bool
	TypingMaskVal string
}

func Prompt(opts *Options) string {
	prompt := opts.Prompt

	if opts.DefaultStr != "" && !opts.HideDefault {
		defaultStr := opts.DefaultStr
		if opts.MaskDefault {
			defaultStr = s.MaskString(defaultStr, 4)
		}
		prompt = fmt.Sprintf("%s [%s]", opts.Prompt, defaultStr)
	}

	val, err := ui.Ask(prompt, &input.Options{
		Default:     opts.DefaultStr,
		Hide:        opts.HideTyping,
		Mask:        opts.MaskTyping,
		MaskVal:     opts.TypingMaskVal,
		Required:    false,
		HideDefault: true,
		HideOrder:   true,
		Loop:        false,
	})

	if err != nil {
		errors.Panic(err)
	}

	return val
}

func ForceYes(prompt string, exitMessage string) {
	for true {
		str := Prompt(&Options{
			Prompt:      prompt + " [y/n]",
			HideDefault: true,
		})

		if strings.ToLower(str) == "y" {
			return
		}

		if strings.ToLower(str) == "n" {
			if exitMessage != "" {
				fmt.Println(exitMessage + "\n")
			}
			os.Exit(1)
		}

		fmt.Println("please enter \"y\" or \"n\"")
		fmt.Println()
	}
}
