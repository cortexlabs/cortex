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
			PromptOpts: &cr.PromptOptions{
				Prompt: "What is your question or issue?",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "EmailAddress",
			PromptOpts: &cr.PromptOptions{
				Prompt: "What is your email address?",
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
	Short: "request support from cortex developers",
	Long: `
This command sends a support request to Cortex developers.`,
	Run: func(cmd *cobra.Command, args []string) {
		support := &SupportRequest{}
		err := cr.ReadPrompt(support, supportPrompValidation)
		if err != nil {
			errors.Exit(err)
		}
		support.Timestamp = time.Now()
		support.Source = "cli.support"
		support.ID = uuid.New().String()

		byteArray, _ := json.Marshal(support)

		resp, err := http.Post(consts.TelemetryURL+"/support", "application/json", bytes.NewReader(byteArray))
		if err != nil {
			errors.PrintError(err)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Println("Your request for support has been sent, Cortex developers will get back to you soon at: " + support.EmailAddress)
			return
		}

		fmt.Println("Failed to send support request, please file an issue in our GitHub: https://github.com/cortexlabs/cortex")
	},
}
