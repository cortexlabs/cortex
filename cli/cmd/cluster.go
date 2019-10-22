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

// cx cluster up/down

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

var flagClusterConfig string

func init() {
	addClusterConfigFlag(upCmd)
	addClusterConfigFlag(updateCmd)
	addClusterConfigFlag(infoCmd)
	addClusterConfigFlag(downCmd)
	clusterCmd.AddCommand(upCmd)
	clusterCmd.AddCommand(updateCmd)
	clusterCmd.AddCommand(infoCmd)
	clusterCmd.AddCommand(downCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagClusterConfig, "config", "c", "", "path to a Cortex cluster configuration file")
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage a Cortex cluster",
	Long:  "Manage a Cortex cluster",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "spin up a Cortex cluster",
	Long: `
This command spins up a Cortex cluster on your AWS account.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		promptForEmail()

		clusterConfig, awsCreds, err := getInstallClusterConfig()
		if err != nil {
			errors.Exit(err)
		}

		err = runManagerCommand("/root/install.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a Cortex cluster",
	Long: `
This command updates a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, awsCreds, err := getInstallClusterConfig()
		if err != nil {
			errors.Exit(err)
		}

		err = runManagerCommand("/root/install.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a Cortex cluster",
	Long: `
This command gets information about a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, awsCreds, err := getAccessClusterConfig()
		if err != nil {
			errors.Exit(err)
		}

		err = runManagerCommand("/root/info.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a Cortex cluster",
	Long: `
This command spins down a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		prompt.ForceYes("Are you sure you want to uninstall Cortex? (Your cluster will be spun down and all APIs will be deleted)")

		clusterConfig, awsCreds, err := getAccessClusterConfig()
		if err != nil {
			errors.Exit(err)
		}

		err = runManagerCommand("/root/uninstall_eks.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var emailPrompValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "EmailAddress",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Email address [press enter to skip]",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required:  false,
				Validator: cr.EmailValidator(),
			},
		},
	},
}

func promptForEmail() {
	fmt.Println("")
	emailAddressContainer := &struct {
		EmailAddress *string
	}{}
	err := cr.ReadPrompt(emailAddressContainer, emailPrompValidation)
	if err != nil {
		errors.Exit(err)
	}

	if emailAddressContainer.EmailAddress != nil {
		supportRequest := &SupportRequest{
			Timestamp:    time.Now(),
			Source:       "cli.cluster.install", // TODO add notificaton for this
			ID:           uuid.New().String(),
			EmailAddress: *emailAddressContainer.EmailAddress,
		}

		byteArray, _ := json.Marshal(supportRequest)

		go func() {
			resp, err := http.Post(consts.TelemetryURL+"/support", "application/json", bytes.NewReader(byteArray))
			if err == nil {
				defer resp.Body.Close()
			}
		}()
	}
}
