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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

type SupportRequest struct {
	EmailAddress string `json:"email_address"`
	Body         string `json:"body"`
}

var supportPrompValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "Body",
			PromptOpts: &prompt.Options{
				Prompt: "what is your question or issue",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "EmailAddress",
			PromptOpts: &prompt.Options{
				Prompt: "what is your email address",
			},
			StringValidation: &cr.StringValidation{
				Required:  true,
				Validator: cr.EmailValidator(),
			},
		},
	},
}

var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "send a support request to the maintainers",
	Run: func(cmd *cobra.Command, args []string) {
		supportRequest := &SupportRequest{}
		err := cr.ReadPrompt(supportRequest, supportPrompValidation)
		if err != nil {
			exit.Error(err)
		}

		if !isTelemetryEnabled() {
			initTelemetry()
		}

		telemetry.RecordEmail(supportRequest.EmailAddress)
		telemetry.Event("cli.support", map[string]interface{}{
			"message": supportRequest.Body,
			"email":   supportRequest.EmailAddress,
		})

		if !isTelemetryEnabled() {
			telemetry.Close()
		}

		fmt.Println("thanks for letting us know, we will get back to you soon at " + supportRequest.EmailAddress)
	},
}
