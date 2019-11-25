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
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

var flagClusterConfig string

func init() {
	addClusterConfigFlag(updateCmd)
	clusterCmd.AddCommand(updateCmd)

	addClusterConfigFlag(infoCmd)
	addEnvFlag(infoCmd)
	clusterCmd.AddCommand(infoCmd)

	addClusterConfigFlag(upCmd)
	clusterCmd.AddCommand(upCmd)

	addClusterConfigFlag(downCmd)
	clusterCmd.AddCommand(downCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagClusterConfig, "config", "c", "", "path to a Cortex cluster configuration file")
	cmd.PersistentFlags().SetAnnotation("config", cobra.BashCompFilenameExt, configFileExts)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage a Cortex cluster",
	Long:  "Manage a Cortex cluster",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "spin up a Cortex cluster",
	Long:  `This command spins up a Cortex cluster on your AWS account.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkDockerRunning(); err != nil {
			errors.Exit(err)
		}

		promptForEmail()
		awsCreds, err := getAWSCredentials(flagClusterConfig)
		if err != nil {
			errors.Exit(err)
		}

		clusterConfig, err := getInstallClusterConfig(awsCreds)
		if err != nil {
			errors.Exit(err)
		}

		_, err = runManagerCommand("/root/install.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a Cortex cluster",
	Long:  `This command updates a Cortex cluster.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkDockerRunning(); err != nil {
			errors.Exit(err)
		}

		awsCreds, err := getAWSCredentials(flagClusterConfig)
		if err != nil {
			errors.Exit(err)
		}

		fmt.Println("fetching cluster configuration ..." + "\n")
		cachedClusterConfig := refreshCachedClusterConfig(awsCreds)

		clusterConfig, err := getUpdateClusterConfig(cachedClusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
		_, err = runManagerCommand("/root/install.sh --update", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a Cortex cluster",
	Long:  `This command gets information about a Cortex cluster.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkDockerRunning(); err != nil {
			errors.Exit(err)
		}
		awsCreds, err := getAWSCredentials(flagClusterConfig)
		if err != nil {
			errors.Exit(err)
		}

		fmt.Println("fetching cluster configuration ..." + "\n")
		clusterConfig := refreshCachedClusterConfig(awsCreds)

		out, err := runManagerCommand("/root/info.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}
		if strings.Contains(out, "there isn't a cortex cluster") {
			errors.Exit()
		}

		fmt.Println()

		httpResponse, err := HTTPGet("/info")
		if err != nil {
			fmt.Println(clusterConfig.UserFacingString())
			fmt.Println("\n" + errors.Wrap(err, "unable to connect to operator").Error())
			return
		}
		var infoResponse schema.InfoResponse
		err = json.Unmarshal(httpResponse, &infoResponse)
		if err != nil {
			fmt.Println(clusterConfig.UserFacingString())
			fmt.Println("\n" + errors.Wrap(err, "unable to parse operator response").Error())
			return
		}
		infoResponse.ClusterConfig.ClusterConfig = *clusterConfig

		var items []table.KV
		items = append(items, table.KV{K: "aws access key id", V: infoResponse.MaskedAWSAccessKeyID})
		items = append(items, infoResponse.ClusterConfig.UserFacingTable()...)

		fmt.Println(table.AlignKeyValue(items, ":", 1))
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a Cortex cluster",
	Long:  `This command spins down a Cortex cluster.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkDockerRunning(); err != nil {
			errors.Exit(err)
		}

		prompt.YesOrExit("your cluster will be spun down and all apis will be deleted, are you sure you want to uninstall cortex?", "")

		awsCreds, err := getAWSCredentials(flagClusterConfig)
		if err != nil {
			errors.Exit(err)
		}

		clusterConfig := refreshCachedClusterConfig(awsCreds)

		_, err = runManagerCommand("/root/uninstall.sh", clusterConfig, awsCreds)
		if err != nil {
			errors.Exit(err)
		}

		os.Remove(cachedClusterConfigPath)
	},
}

var emailPrompValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "EmailAddress",
			PromptOpts: &prompt.Options{
				Prompt: "email address [press enter to skip]",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required:  false,
				Validator: cr.EmailValidator(),
			},
		},
	},
}

func promptForEmail() {
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
			Source:       "cli.cluster.install",
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

func refreshCachedClusterConfig(awsCreds *AWSCredentials) *clusterconfig.ClusterConfig {
	userClusterConfig := &clusterconfig.ClusterConfig{}
	err := clusterconfig.SetFileDefaults(userClusterConfig)
	if err != nil {
		errors.Exit(err)
	}

	if flagClusterConfig != "" {
		err := readClusterConfigFile(userClusterConfig, flagClusterConfig)
		if err != nil {
			errors.Exit(err)
		}
	}

	cachedClusterConfig := &clusterconfig.ClusterConfig{}
	readClusterConfigFile(cachedClusterConfig, cachedClusterConfigPath)

	if userClusterConfig.Region == nil {
		userClusterConfig.Region = cachedClusterConfig.Region
	}

	if userClusterConfig.Region == nil {
		errors.Exit(fmt.Sprintf("unable to find %s; please configure \"%s\" to the s3 region of an existing cortex cluster or create a cortex cluster with `cortex cluster up`", clusterconfig.RegionKey, clusterconfig.RegionKey))
	}

	out, err := runRefreshClusterConfig(userClusterConfig, awsCreds)
	if err != nil {
		errors.Exit(err)
	}
	if strings.Contains(out, "there isn't a cortex cluster") {
		errors.Exit()
	}

	refreshedClusterConfig := &clusterconfig.ClusterConfig{}
	readClusterConfigFile(refreshedClusterConfig, cachedClusterConfigPath)
	return refreshedClusterConfig
}
