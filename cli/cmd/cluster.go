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

// cx cluster up/down

package cmd

import (
	"fmt"
	"os"
	"time"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/spf13/cobra"
)

var _flagClusterConfig string
var _flagDebug bool

func init() {
	addClusterConfigFlag(_updateCmd)
	addEnvFlag(_updateCmd)
	_clusterCmd.AddCommand(_updateCmd)

	addClusterConfigFlag(_infoCmd)
	addEnvFlag(_infoCmd)
	_infoCmd.PersistentFlags().BoolVarP(&_flagDebug, "debug", "d", false, "save the current cluster state to a file")
	_clusterCmd.AddCommand(_infoCmd)

	addClusterConfigFlag(_upCmd)
	addEnvFlag(_upCmd)
	_clusterCmd.AddCommand(_upCmd)

	addClusterConfigFlag(_downCmd)
	_clusterCmd.AddCommand(_downCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&_flagClusterConfig, "config", "c", "", "path to a cluster configuration file")
	cmd.PersistentFlags().SetAnnotation("config", cobra.BashCompFilenameExt, _configFileExts)
}

var _clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage a cluster",
}

var _upCmd = &cobra.Command{
	Use:   "up",
	Short: "spin up a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.EventNotify("cli.cluster.up")

		if err := checkDockerRunning(); err != nil {
			exit.Error(err)
		}

		promptForEmail()
		awsCreds, err := getAWSCredentials(_flagClusterConfig)
		if err != nil {
			exit.Error(err)
		}

		clusterConfig, err := getInstallClusterConfig(awsCreds)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerUpdateCommand("/root/install.sh", clusterConfig, awsCreds)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := "\nDebugging tips (may not apply to this error):"
			helpStr += fmt.Sprintf("\n* if your cluster started spinning up but was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", *clusterConfig.Region)
			helpStr += fmt.Sprintf("\n* if your cluster started spinning up, please ensure that your CloudFormation stacks for this cluster have been fully deleted before trying to spin up this cluster again: https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=-%s-", *clusterConfig.Region, clusterConfig.ClusterName)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUp(out + helpStr))
		}
	},
}

var _updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.update")

		if err := checkDockerRunning(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig)
		if err != nil {
			exit.Error(err)
		}

		cachedClusterConfig := refreshCachedClusterConfig(awsCreds)

		clusterConfig, err := getClusterUpdateConfig(cachedClusterConfig, awsCreds)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerUpdateCommand("/root/install.sh --update", clusterConfig, awsCreds)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := "\nDebugging tips (may not apply to this error):"
			helpStr += fmt.Sprintf("\n* if your cluster was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the  \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", *clusterConfig.Region)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUpdate(out + helpStr))
		}
	},
}

var _infoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.info")

		if err := checkDockerRunning(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig)
		if err != nil {
			exit.Error(err)
		}

		if _flagDebug {
			accessConfig, err := getClusterAccessConfig()
			if err != nil {
				exit.Error(err)
			}

			out, exitCode, err := runManagerAccessCommand("/root/debug.sh", *accessConfig, awsCreds)
			if err != nil {
				exit.Error(err)
			}
			if exitCode == nil || *exitCode != 0 {
				exit.Error(ErrorClusterDebug(out))
			}

			timestamp := time.Now().UTC().Format("2006-01-02-15-04-05")
			userDebugPath := fmt.Sprintf("cortex-debug-%s.tgz", timestamp) // note: if modifying this string, also change it in files.IgnoreCortexDebug()
			err = os.Rename(_debugPath, userDebugPath)
			if err != nil {
				exit.Error(errors.WithStack(err))
			}

			fmt.Println("saved cluster info to ./" + userDebugPath)
			return
		}

		clusterConfig := refreshCachedClusterConfig(awsCreds)

		out, exitCode, err := runManagerAccessCommand("/root/info.sh", clusterConfig.ToAccessConfig(), awsCreds)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			exit.Error(ErrorClusterInfo(out))
		}

		fmt.Println()

		httpResponse, err := HTTPGet("/info")
		if err != nil {
			fmt.Println(clusterConfig.UserStr())
			fmt.Println("\n" + errors.Message(err, "unable to connect to operator"))
			return
		}

		var infoResponse schema.InfoResponse
		err = json.Unmarshal(httpResponse, &infoResponse)
		if err != nil {
			fmt.Println(clusterConfig.UserStr())
			fmt.Println("\n" + errors.Message(err, "unable to parse operator response"))
			return
		}
		infoResponse.ClusterConfig.Config = clusterConfig

		var items table.KeyValuePairs
		items.Add("aws access key id", infoResponse.MaskedAWSAccessKeyID)
		items.AddAll(infoResponse.ClusterConfig.UserTable())

		items.Print()
	},
}

var _downCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.down")

		if err := checkDockerRunning(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig)
		if err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfig()
		if err != nil {
			exit.Error(err)
		}

		prompt.YesOrExit(fmt.Sprintf("your cluster (%s in %s) will be spun down and all apis will be deleted, are you sure you want to continue?", *accessConfig.ClusterName, *accessConfig.Region), "", "")

		out, exitCode, err := runManagerAccessCommand("/root/uninstall.sh", *accessConfig, awsCreds)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := fmt.Sprintf("\nNote: if this error cannot be resolved, please ensure that all CloudFormation stacks for this cluster eventually become been fully deleted (https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=-%s-). If the stack deletion process has failed, please manually delete the stack from the AWS console (this may require manually deleting particular AWS resources that are blocking the stack deletion)", *accessConfig.Region, *accessConfig.ClusterName)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterDown(out + helpStr))
		}

		cachedConfigPath := cachedClusterConfigPath(*accessConfig.ClusterName, *accessConfig.Region)
		os.Remove(cachedConfigPath)
	},
}

var _emailPrompValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "EmailAddress",
			PromptOpts: &prompt.Options{
				Prompt: "email address [press enter to skip]",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required:  false,
				Validator: cr.EmailValidator,
			},
		},
	},
}

func promptForEmail() {
	if email, err := files.ReadFile(_emailPath); err == nil && email != "" {
		return
	}

	emailAddressContainer := &struct {
		EmailAddress *string
	}{}
	err := cr.ReadPrompt(emailAddressContainer, _emailPrompValidation)
	if err != nil {
		exit.Error(err)
	}

	if emailAddressContainer.EmailAddress != nil {
		if !isTelemetryEnabled() {
			initTelemetry()
		}

		telemetry.RecordEmail(*emailAddressContainer.EmailAddress)

		if !isTelemetryEnabled() {
			telemetry.Close()
		}

		files.WriteFile([]byte(*emailAddressContainer.EmailAddress), _emailPath)
	}
}

func refreshCachedClusterConfig(awsCreds AWSCredentials) clusterconfig.Config {
	accessConfig, err := getClusterAccessConfig()
	if err != nil {
		exit.Error(err)
	}

	// add empty file if cached cluster doesn't exist so that the file output by manager container maintains current user permissions
	cachedConfigPath := cachedClusterConfigPath(*accessConfig.ClusterName, *accessConfig.Region)
	if !files.IsFile(cachedConfigPath) {
		files.MakeEmptyFile(cachedConfigPath)
	}

	mountedConfigPath := mountedClusterConfigPath(*accessConfig.ClusterName, *accessConfig.Region)

	fmt.Println("fetching cluster configuration ..." + "\n")
	out, exitCode, err := runManagerAccessCommand("/root/refresh.sh "+mountedConfigPath, *accessConfig, awsCreds)
	if err != nil {
		os.Remove(cachedConfigPath)
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		os.Remove(cachedConfigPath)
		exit.Error(ErrorClusterRefresh(out))
	}

	refreshedClusterConfig := &clusterconfig.Config{}
	readCachedClusterConfigFile(refreshedClusterConfig, cachedConfigPath)
	return *refreshedClusterConfig
}
