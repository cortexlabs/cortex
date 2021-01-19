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

package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	input "github.com/cortexlabs/go-input"
)

var _ui = &input.UI{
	Writer: os.Stdout,
	Reader: os.Stdin,
}

type Options struct {
	Prompt              string
	DefaultStr          string
	HideDefault         bool
	MaskDefault         bool
	HideTyping          bool
	MaskTyping          bool
	TypingMaskVal       string
	SkipTrailingNewline bool
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

	val, err := _ui.Ask(prompt, &input.Options{
		Default:     opts.DefaultStr,
		Hide:        opts.HideTyping,
		Mask:        opts.MaskTyping,
		MaskVal:     opts.TypingMaskVal,
		Required:    false,
		HideDefault: true,
		HideOrder:   true,
		Loop:        false,
		SkipNewline: opts.SkipTrailingNewline,
	})

	if err != nil {
		if errors.Message(err) == "interrupted" {
			exit.Error(ErrorUserCtrlC())
		}
		if strings.Contains(errors.Message(err), "not a terminal") {
			err = errors.Append(err, "\n\nyou may be able to pass flags into this command to provide all required inputs and/or skip prompts (e.g. via `--yes`)")
		}
		exit.Error(err)
	}

	return val
}

func YesOrExit(prompt string, yesMessage string, noMessage string) {
	for true {
		str := Prompt(&Options{
			Prompt:      prompt + " (y/n)",
			HideDefault: true,
		})

		if strings.ToLower(str) == "y" {
			if yesMessage != "" {
				fmt.Println(yesMessage)
			}
			return
		}

		if strings.ToLower(str) == "n" {
			if noMessage != "" {
				fmt.Println(noMessage)
			}
			exit.Error(ErrorUserNoContinue())
		}

		fmt.Println("please enter \"y\" or \"n\"")
		fmt.Println()
	}
}

func YesOrNo(prompt string, yesMessage string, noMessage string) bool {
	for true {
		str := Prompt(&Options{
			Prompt:      prompt + " (y/n)",
			HideDefault: true,
		})

		if strings.ToLower(str) == "y" {
			if yesMessage != "" {
				fmt.Println(yesMessage)
			}
			return true
		}

		if strings.ToLower(str) == "n" {
			if noMessage != "" {
				fmt.Println(noMessage)
			}
			return false
		}

		fmt.Println("please enter \"y\" or \"n\"")
		fmt.Println()
	}
	return false
}
