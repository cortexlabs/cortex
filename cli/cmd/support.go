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
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
)

type SupportRequest struct {
	Timestamp    time.Time `json:"timestamp"`
	EmailAddress string    `json:"email_address"`
	ID           string    `json:"support_id"`
	Source       string    `json:"source"`
	Body         string    `json:"body"`
}

var supportPrompValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "Body",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "What is your question or issue",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "EmailAddress",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "What is your email address",
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
	Short: "request support from Cortex maintainers",
	Long: `
This command sends a support request to Cortex maintainers.`,
	Run: func(cmd *cobra.Command, args []string) {
		supportRequest := &SupportRequest{}
		fmt.Println("")
		err := cr.ReadPrompt(supportRequest, supportPrompValidation)
		if err != nil {
			errors.Exit(err)
		}
		supportRequest.Timestamp = time.Now()
		supportRequest.Source = "cli.support"
		supportRequest.ID = uuid.New().String()

		byteArray, _ := json.Marshal(supportRequest)

		resp, err := http.Post(consts.TelemetryURL+"/support", "application/json", bytes.NewReader(byteArray))
		if err != nil {
			errors.PrintError(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Println("An error occured when submitting your request, please file an issue on GitHub (https://github.com/cortexlabs/cortex) or email us at hello@cortex.dev")
			return
		}

		fmt.Println("Thanks for letting us know, we will get back to you soon at " + supportRequest.EmailAddress)
	},
}
