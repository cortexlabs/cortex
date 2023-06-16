/*
Copyright 2022 Cortex Labs, Inc.

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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PEAT-AI/yaml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/health"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/clusterstate"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

var (
	_flagClusterUpEnv                string
	_flagClusterInfoEnv              string
	_flagClusterConfig               string
	_flagClusterName                 string
	_flagClusterRegion               string
	_flagClusterInfoDebug            bool
	_flagClusterInfoPrintConfig      bool
	_flagClusterDisallowPrompt       bool
	_flagClusterDownKeepAWSResources bool
)

var _eksctlPrefixRegex = regexp.MustCompile(`^.*[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} \[.+] {2}`)

func clusterInit() {
	_clusterUpCmd.Flags().SortFlags = false
	_clusterUpCmd.Flags().StringVarP(&_flagClusterUpEnv, "configure-env", "e", "", "name of environment to configure (default: the name of your cluster)")
	_clusterUpCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterUpCmd)

	_clusterInfoCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterInfoCmd)
	addClusterNameFlag(_clusterInfoCmd)
	addClusterRegionFlag(_clusterInfoCmd)
	_clusterInfoCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.OutputTypeStrings(), "|")))
	_clusterInfoCmd.Flags().StringVarP(&_flagClusterInfoEnv, "configure-env", "e", "", "name of environment to configure")
	_clusterInfoCmd.Flags().BoolVarP(&_flagClusterInfoDebug, "debug", "d", false, "save the current cluster state to a file")
	_clusterInfoCmd.Flags().BoolVarP(&_flagClusterInfoPrintConfig, "print-config", "", false, "print the cluster config")
	_clusterInfoCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterInfoCmd)

	_clusterConfigureCmd.Flags().SortFlags = false
	_clusterConfigureCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterConfigureCmd)

	_clusterDownCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterDownCmd)
	addClusterNameFlag(_clusterDownCmd)
	addClusterRegionFlag(_clusterDownCmd)
	_clusterDownCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterDownCmd.Flags().BoolVar(&_flagClusterDownKeepAWSResources, "keep-aws-resources", false, "skip deletion of resources that cortex provisioned on aws (bucket contents, ebs volumes, log group)")
	_clusterCmd.AddCommand(_clusterDownCmd)

	_clusterExportCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterExportCmd)
	addClusterNameFlag(_clusterExportCmd)
	addClusterRegionFlag(_clusterExportCmd)
	_clusterCmd.AddCommand(_clusterExportCmd)

	_clusterHealthCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterHealthCmd)
	addClusterNameFlag(_clusterHealthCmd)
	addClusterRegionFlag(_clusterHealthCmd)
	_clusterHealthCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.OutputTypeStringsExcluding(flags.YAMLOutputType), "|")))
	_clusterCmd.AddCommand(_clusterHealthCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterConfig, "config", "c", "", "path to a cluster configuration file")
	err := cmd.Flags().SetAnnotation("config", cobra.BashCompFilenameExt, _configFileExts)
	if err != nil {
		exit.Error(err) // should never happen
	}
}

func addClusterNameFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterName, "name", "n", "", "name of the cluster")
}

func addClusterRegionFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterRegion, "region", "r", "", "aws region of the cluster")
}

var _clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage cortex clusters (contains subcommands)",
}

var _clusterUpCmd = &cobra.Command{
	Use:   "up CLUSTER_CONFIG_FILE",
	Short: "spin up a cluster on aws",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.EventNotify("cli.cluster.up")

		clusterConfigFile := args[0]

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getNewClusterAccessConfig(clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		envName := _flagClusterUpEnv
		if envName == "" {
			envName = accessConfig.ClusterName
		}

		envExists, err := isEnvConfigured(envName)
		if err != nil {
			exit.Error(err)
		}
		if envExists {
			if _flagClusterDisallowPrompt {
				fmt.Printf("found an existing environment named \"%s\", which will be overwritten to connect to this cluster once it's created\n\n", envName)
			} else {
				prompt.YesOrExit(fmt.Sprintf("found an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster once it's created?", envName), "", "you can specify a different environment name to be configured to connect to this cluster by specifying the --configure-env flag (e.g. `cortex cluster up --configure-env prod`); or you can list your environments with `cortex env list` and delete an environment with `cortex env delete ENV_NAME`")
			}
		}

		awsClient, err := newAWSClient(accessConfig.Region, true)
		if err != nil {
			exit.Error(err)
		}

		stacks, err := clusterstate.GetClusterStacks(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		state := clusterstate.GetClusterState(stacks)
		if err := clusterstate.AssertClusterState(stacks, state, clusterstate.StateClusterDoesntExist); err != nil {
			exit.Error(err)
		}

		promptIfNotAdmin(awsClient, _flagClusterDisallowPrompt)

		clusterConfig, err := getInstallClusterConfig(awsClient, clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		confirmInstallClusterConfig(clusterConfig, awsClient, _flagClusterDisallowPrompt)

		err = createS3BucketIfNotFound(awsClient, clusterConfig.Bucket, clusterConfig.Tags)
		if err != nil {
			exit.Error(err)
		}

		err = setLifecycleRulesOnClusterUp(awsClient, clusterConfig.Bucket, clusterConfig.ClusterUID)
		if err != nil {
			exit.Error(err)
		}

		err = createLogGroupIfNotFound(awsClient, clusterConfig.ClusterName, clusterConfig.Tags)
		if err != nil {
			exit.Error(err)
		}

		accountID, _, err := awsClient.GetCachedAccountID()
		if err != nil {
			exit.Error(err)
		}

		err = clusterconfig.CreateDefaultPolicy(awsClient, clusterconfig.CortexPolicyTemplateArgs{
			ClusterName: clusterConfig.ClusterName,
			LogGroup:    clusterConfig.ClusterName,
			Bucket:      clusterConfig.Bucket,
			Region:      clusterConfig.Region,
			AccountID:   accountID,
		})
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerWithClusterConfig("/root/install.sh", clusterConfig, awsClient, nil, nil, nil)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			out = s.LastNChars(filterEKSCTLOutput(out), 8192) // get the last 8192 characters because that is the sentry message limit
			eksCluster, err := awsClient.EKSClusterOrNil(clusterConfig.ClusterName)
			if err != nil {
				helpStr := "\ndebugging tips (may or may not apply to this error):"
				helpStr += fmt.Sprintf("\n* if your cluster started spinning up but was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
				helpStr += "\n* if your cluster started spinning up, please run `cortex cluster down` to delete the cluster before trying to create this cluster again"
				fmt.Println(helpStr)
				exit.Error(ErrorClusterUp(out))
			}

			// the cluster never started spinning up
			if eksCluster == nil {
				exit.Error(ErrorClusterUp(out))
			}

			clusterTags := map[string]string{clusterconfig.ClusterNameTag: clusterConfig.ClusterName}
			asgs, err := awsClient.AutoscalingGroups(clusterTags)
			if err != nil {
				helpStr := "\ndebugging tips (may or may not apply to this error):"
				helpStr += fmt.Sprintf("\n* if your cluster was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
				helpStr += "\n* please run `cortex cluster down` to delete the cluster before trying to create this cluster again"
				fmt.Println(helpStr)
				exit.Error(ErrorClusterUp(out + helpStr))
			}

			// no autoscaling groups were created
			if len(asgs) == 0 {
				helpStr := "\nplease run `cortex cluster down` to delete the cluster before trying to create this cluster again"
				fmt.Println(helpStr)
				exit.Error(ErrorClusterUp(out + helpStr))
			}

			for _, asg := range asgs {
				activity, err := awsClient.MostRecentASGActivity(*asg.AutoScalingGroupName)
				if err != nil {
					helpStr := "\ndebugging tips (may or may not apply to this error):"
					helpStr += fmt.Sprintf("\n* if your cluster was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
					helpStr += "\n* please run `cortex cluster down` to delete the cluster before trying to create this cluster again"
					fmt.Println(helpStr)
					exit.Error(ErrorClusterUp(out + helpStr))
				}

				if activity != nil && (activity.StatusCode == nil || *activity.StatusCode != autoscaling.ScalingActivityStatusCodeSuccessful) {
					status := "(none)"
					if activity.StatusCode != nil {
						status = *activity.StatusCode
					}
					description := "(none)"
					if activity.Description != nil {
						description = *activity.Description
					}

					helpStr := "\nyour cluster was unable to provision EC2 instances; here is one of the encountered errors:"
					helpStr += fmt.Sprintf("\n\n> status: %s\n> description: %s", status, description)
					helpStr += fmt.Sprintf("\n\nadditional error information might be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
					helpStr += "\n\nplease run `cortex cluster down` to delete the cluster before trying to create this cluster again"
					fmt.Println(helpStr)
					exit.Error(ErrorClusterUp(out + helpStr))
				}
			}

			// No failed asg activities
			helpStr := "\nplease run `cortex cluster down` to delete the cluster before trying to create this cluster again"
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUp(out + helpStr))
		}

		loadBalancer, err := getNLBLoadBalancer(clusterConfig.ClusterName, OperatorLoadBalancer, awsClient)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", envName)))
		}

		newEnvironment := cliconfig.Environment{
			Name:             envName,
			OperatorEndpoint: "https://" + *loadBalancer.DNSName,
		}

		err = addEnvToCLIConfig(newEnvironment, true)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", envName)))
		}

		if envExists {
			fmt.Printf(console.Bold("\nthe environment named \"%s\" has been updated to point to this cluster (and was set as the default environment)\n"), envName)
		} else {
			fmt.Printf(console.Bold("\nan environment named \"%s\" has been configured to point to this cluster (and was set as the default environment)\n"), envName)
		}
	},
}

var _clusterConfigureCmd = &cobra.Command{
	Use:   "configure CLUSTER_CONFIG_FILE",
	Short: "update the cluster's configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.configure")

		clusterConfigFile := args[0]

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getNewClusterAccessConfig(clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := newAWSClient(accessConfig.Region, true)
		if err != nil {
			exit.Error(err)
		}

		restConfig, err := getClusterRESTConfig(awsClient, accessConfig.ClusterName)
		if err != nil {
			exit.Error(err)
		}

		scheme := runtime.NewScheme()
		if err := clientgoscheme.AddToScheme(scheme); err != nil {
			exit.Error(err)
		}

		k8sClient, err := k8s.New(consts.DefaultNamespace, false, restConfig, scheme)
		if err != nil {
			exit.Error(err)
		}

		stacks, err := clusterstate.GetClusterStacks(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		state := clusterstate.GetClusterState(stacks)
		if err := clusterstate.AssertClusterState(stacks, state, clusterstate.StateClusterExists); err != nil {
			exit.Error(err)
		}

		oldClusterConfig := refreshCachedClusterConfig(awsClient, accessConfig, true)

		promptIfNotAdmin(awsClient, _flagClusterDisallowPrompt)

		newClusterConfig, configureChanges, err := getConfigureClusterConfig(awsClient, k8sClient, stacks, oldClusterConfig, clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		if !configureChanges.HasChanges() {
			fmt.Println("your cluster is already up to date")
			exit.Ok()
		}

		confirmConfigureClusterConfig(configureChanges, oldClusterConfig, *newClusterConfig, _flagClusterDisallowPrompt)

		out, exitCode, err := runManagerWithClusterConfig("/root/install.sh --configure", newClusterConfig, awsClient, nil, nil, []string{
			"CORTEX_NODEGROUP_NAMES_TO_UPDATE=" + strings.Join(configureChanges.NodeGroupsToUpdate, " "),        // NodeGroupsToUpdate contain the cluster config node-group names
			"CORTEX_NODEGROUP_NAMES_TO_ADD=" + strings.Join(configureChanges.NodeGroupsToAdd, " "),              // NodeGroupsToAdd contain the cluster config node-group names
			"CORTEX_EKS_NODEGROUP_NAMES_TO_REMOVE=" + strings.Join(configureChanges.EKSNodeGroupsToRemove, " "), // EKSNodeGroupsToRemove contain the EKS node-group names
		})
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			out = s.LastNChars(out, 8192) // get the last 8192 characters because that is the sentry message limit

			helpStr := "\ndebugging tips (may or may not apply to this error):"
			helpStr += fmt.Sprintf(
				"\n* if your cluster was unable to provision/remove/scale some nodegroups, additional error information may be found in the description of your cloudformation stack (https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks)"+
					" or in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the  \"Activity\" or \"Activity History\" tab) (https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups)",
				oldClusterConfig.Region,
				oldClusterConfig.Region,
			)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterConfigure(out + helpStr))
		}
	},
}

var _clusterInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.info")

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfigWithCache(true)
		if err != nil {
			exit.Error(err)
		}

		if _flagClusterInfoPrintConfig && _flagOutput == flags.PrettyOutputType {
			_flagOutput = flags.YAMLOutputType
		}

		awsClient, err := newAWSClient(accessConfig.Region, _flagOutput == flags.PrettyOutputType)
		if err != nil {
			exit.Error(err)
		}

		if _flagClusterInfoPrintConfig && _flagClusterInfoDebug {
			exit.Error(ErrorMutuallyExclusiveFlags("--print-config", "--debug"))
		}
		if _flagClusterInfoDebug && _flagOutput != flags.PrettyOutputType {
			exit.Error(ErrorMutuallyExclusiveFlags("--debug", "--output"))
		}

		stacks, err := clusterstate.GetClusterStacks(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		state := clusterstate.GetClusterState(stacks)
		if err := clusterstate.AssertClusterState(stacks, state, clusterstate.StateClusterExists); err != nil {
			exit.Error(err)
		}

		if _flagClusterInfoDebug {
			cmdDebug(awsClient, accessConfig)
		} else if _flagClusterInfoPrintConfig {
			cmdPrintConfig(awsClient, accessConfig, _flagOutput)
		} else {
			cmdInfo(awsClient, accessConfig, stacks, _flagOutput, _flagClusterDisallowPrompt)
		}
	},
}

var _clusterDownCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.down")

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfigWithCache(true)
		if err != nil {
			exit.Error(err)
		}

		// Check AWS access
		awsClient, err := newAWSClient(accessConfig.Region, true)
		if err != nil {
			exit.Error(err)
		}

		accountID, _, err := awsClient.GetCachedAccountID()
		if err != nil {
			exit.Error(err)
		}
		bucketName := clusterconfig.BucketName(accountID, accessConfig.ClusterName, accessConfig.Region)

		warnIfNotAdmin(awsClient)

		if _flagClusterDisallowPrompt {
			fmt.Printf("your cluster named \"%s\" in %s will be spun down and all apis will be deleted\n\n", accessConfig.ClusterName, accessConfig.Region)
		} else {
			prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s will be spun down and all apis will be deleted, are you sure you want to continue?", accessConfig.ClusterName, accessConfig.Region), "", "")
		}

		var clusterExists bool
		errorsList := []error{}

		fmt.Print("￮ retrieving cluster ... ")
		stacks, err := clusterstate.GetClusterStacks(awsClient, accessConfig)
		if err != nil {
			errorsList = append(errorsList, err)
			fmt.Print("failed ✗")
			fmt.Printf("\n\ncouldn't retrieve cluster state; check the cluster stacks in the cloudformation console: https://%s.console.aws.amazon.com/cloudformation\n", accessConfig.Region)
			errors.PrintError(err)
			fmt.Println()
		} else {
			if clusterstate.GetClusterState(stacks) == clusterstate.StateClusterDoesntExist {
				fmt.Println("cluster doesn't exist ✓")
			} else {
				fmt.Println("✓")
				clusterExists = true
			}
		}

		// updating CLI env is best-effort, so ignore errors
		loadBalancer, _ := getNLBLoadBalancer(accessConfig.ClusterName, OperatorLoadBalancer, awsClient)

		fmt.Print("￮ deleting sqs queues ... ")
		numDeleted, err := awsClient.DeleteQueuesWithPrefix(clusterconfig.SQSNamePrefix(accessConfig.ClusterName))
		if err != nil {
			errorsList = append(errorsList, err)
			fmt.Print("failed ✗")
			fmt.Printf("\n\nfailed to delete all sqs queues; please delete queues starting with the name %s via the cloudwatch console: https://%s.console.aws.amazon.com/sqs/v2/home\n", clusterconfig.SQSNamePrefix(accessConfig.ClusterName), accessConfig.Region)
			errors.PrintError(err)
			fmt.Println()
		} else if numDeleted == 0 {
			fmt.Println("no sqs queues exist ✓")
		} else {
			fmt.Println("✓")
		}

		clusterDoesntExist := !clusterExists
		if clusterExists {
			fmt.Print("￮ spinning down the cluster ...")
			out, exitCode, err := runManagerAccessCommand("/root/uninstall.sh", *accessConfig, awsClient, nil, nil)
			if err != nil {
				errorsList = append(errorsList, err)
				fmt.Println()
				errors.PrintError(err)
			} else if exitCode == nil || *exitCode != 0 {
				template := "\nNote: if this error cannot be resolved, please ensure that all CloudFormation stacks for this cluster eventually become fully deleted (%s)."
				template += " If the stack deletion process has failed, please delete the stacks directly from the AWS console (this may require manually deleting particular AWS resources that are blocking the stack deletion)."
				template += " In addition to deleting the stacks manually from the AWS console, also make sure to empty and remove the %s bucket"
				helpStr := fmt.Sprintf(template, clusterstate.CloudFormationURL(accessConfig.ClusterName, accessConfig.Region), bucketName)
				fmt.Println(helpStr)
				errorsList = append(errorsList, ErrorClusterDown(filterEKSCTLOutput(out)+helpStr))
			} else {
				clusterDoesntExist = true
			}
			fmt.Println()
		}

		// set lifecycle policy to clean the bucket
		var bucketExists bool
		if !_flagClusterDownKeepAWSResources {
			fmt.Printf("￮ setting lifecycle policy to empty the %s bucket ... ", bucketName)
			bucketExists, err := awsClient.DoesBucketExist(bucketName)
			if err != nil {
				errorsList = append(errorsList, err)
				fmt.Print("failed ✗")
				fmt.Printf("\n\nfailed to set lifecycle policy to empty the %s bucket; you can remove the bucket manually via the s3 console: https://s3.console.aws.amazon.com/s3/management/%s\n", bucketName, bucketName)
				errors.PrintError(err)
				fmt.Println()
			} else if !bucketExists {
				fmt.Println("bucket doesn't exist ✗")
			} else {
				err = setLifecycleRulesOnClusterDown(awsClient, bucketName)
				if err != nil {
					errorsList = append(errorsList, err)
					fmt.Print("failed ✗")
					fmt.Printf("\n\nfailed to set lifecycle policy to empty the %s bucket; you can remove the bucket manually via the s3 console: https://s3.console.aws.amazon.com/s3/management/%s\n", bucketName, bucketName)
					errors.PrintError(err)
					fmt.Println()
				} else {
					fmt.Println("✓")
				}
			}
		}

		// delete policy after spinning down the cluster (which deletes the roles) because policies can't be deleted if they are attached to roles
		if clusterDoesntExist {
			policyARN := clusterconfig.DefaultPolicyARN(accountID, accessConfig.ClusterName, accessConfig.Region)
			fmt.Printf("￮ deleting auto-generated iam policy %s ... ", policyARN)
			if policy, err := awsClient.GetPolicyOrNil(policyARN); err != nil {
				errorsList = append(errorsList, err)
				fmt.Print("failed ✗")
				fmt.Printf("\n\nfailed to delete auto-generated cortex policy %s; please delete the policy via the iam console: https://console.aws.amazon.com/iam/home#/policies\n", policyARN)
				errors.PrintError(err)
				fmt.Println()
			} else if policy == nil {
				fmt.Println("policy doesn't exist ✓")
			} else {
				err = awsClient.DeletePolicy(policyARN)
				if err != nil {
					errorsList = append(errorsList, err)
					fmt.Print("failed ✗")
					fmt.Printf("\n\nfailed to delete auto-generated cortex policy %s; please delete the policy via the iam console: https://console.aws.amazon.com/iam/home#/policies\n", policyARN)
					errors.PrintError(err)
					fmt.Println()
				} else {
					fmt.Println("✓")
				}
			}
		}

		if !_flagClusterDownKeepAWSResources {
			fmt.Print("￮ deleting ebs volumes ... ")
			volumes, err := listPVCVolumesForCluster(awsClient, accessConfig.ClusterName)
			if err != nil {
				errorsList = append(errorsList, err)
				fmt.Println("\n\nfailed to list volumes for deletion; please delete any volumes associated with your cluster via the ec2 console: https://console.aws.amazon.com/ec2/v2/home?#Volumes")
				errors.PrintError(err)
				fmt.Println()
			} else {
				var failedToDeleteVolumes []string
				var lastErr error
				for _, volume := range volumes {
					err := awsClient.DeleteVolume(*volume.VolumeId)
					if err != nil {
						failedToDeleteVolumes = append(failedToDeleteVolumes, *volume.VolumeId)
						lastErr = err
					}
				}
				if len(volumes) == 0 {
					fmt.Println("no ebs volumes exist ✓")
				} else if lastErr != nil {
					errorsList = append(errorsList, lastErr)
					fmt.Printf("\n\nfailed to delete %s %s; please delete %s via the ec2 console: https://console.aws.amazon.com/ec2/v2/home?#Volumes\n", s.PluralS("volume", len(failedToDeleteVolumes)), s.UserStrsAnd(failedToDeleteVolumes), s.PluralCustom("it", "them", len(failedToDeleteVolumes)))
					errors.PrintError(lastErr)
					fmt.Println()
				} else {
					fmt.Println("✓")
				}
			}

			fmt.Printf("￮ deleting log group %s ... ", accessConfig.ClusterName)
			logGroupExists, err := awsClient.DoesLogGroupExist(accessConfig.ClusterName)
			if err != nil {
				errorsList = append(errorsList, err)
				fmt.Print("failed ✗")
				fmt.Printf("\n\nfailed to list log group for deletion; please delete the log group associated with your cluster via the ec2 console: https://%s.console.aws.amazon.com/cloudwatch/home?#logsV2:log-groups\n", accessConfig.Region)
				errors.PrintError(err)
				fmt.Println()
			} else {
				if !logGroupExists {
					fmt.Println("log group doesn't exist ✓")
				} else {
					err = awsClient.DeleteLogGroup(accessConfig.ClusterName)
					if err != nil {
						errorsList = append(errorsList, err)
						fmt.Print("failed ✗")
						fmt.Printf("\n\nfailed to delete log group %s; please delete the log group associated with your cluster via the ec2 console: https://%s.console.aws.amazon.com/cloudwatch/home?#logsV2:log-groups\n", accessConfig.ClusterName, accessConfig.Region)
						errors.PrintError(err)
						fmt.Println()
					} else {
						fmt.Println("✓")
					}
				}
			}
		}

		// best-effort deletion of cached config
		cachedClusterConfigPath := getCachedClusterConfigPath(accessConfig.ClusterName, accessConfig.Region)
		_ = os.Remove(cachedClusterConfigPath)

		if len(errorsList) > 0 {
			exit.Error(errors.ListOfErrors(ErrClusterDown, false, errorsList...))
		}
		fmt.Printf("\nplease check CloudFormation to ensure that all resources for the %s cluster eventually become successfully deleted: %s\n", accessConfig.ClusterName, clusterstate.CloudFormationURL(accessConfig.ClusterName, accessConfig.Region))
		if !_flagClusterDownKeepAWSResources && bucketExists {
			fmt.Printf("\na lifecycle rule has been applied to the cluster's %s bucket to empty its contents within the next 24 hours; you can delete the %s bucket via the s3 console once it has been emptied (or you can empty and delete it now): https://s3.console.aws.amazon.com/s3/management/%s\n", bucketName, bucketName, bucketName)
		}
		fmt.Println()

		// best-effort deletion of cli environment(s)
		if loadBalancer != nil {
			envNames, isDefaultEnv, _ := getEnvNamesByOperatorEndpoint(*loadBalancer.DNSName)
			if len(envNames) > 0 {
				for _, envName := range envNames {
					err := removeEnvFromCLIConfig(envName)
					if err != nil {
						exit.Error(err)
					}
				}
				fmt.Printf("deleted the %s environment configuration%s\n", s.StrsAnd(envNames), s.SIfPlural(len(envNames)))
				if isDefaultEnv {
					newDefaultEnv, err := getDefaultEnv()
					if err != nil {
						exit.Error(err)
					}
					if newDefaultEnv != nil {
						fmt.Println(fmt.Sprintf("set the default environment to %s", *newDefaultEnv))
					}
				}
			}
		}
	},
}

var _clusterExportCmd = &cobra.Command{
	Use:   "export",
	Short: "download the configurations for all APIs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.export")

		accessConfig, err := getClusterAccessConfigWithCache(true)
		if err != nil {
			exit.Error(err)
		}

		// Check AWS access
		awsClient, err := newAWSClient(accessConfig.Region, true)
		if err != nil {
			exit.Error(err)
		}
		warnIfNotAdmin(awsClient)

		stacks, err := clusterstate.GetClusterStacks(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		state := clusterstate.GetClusterState(stacks)
		if err := clusterstate.AssertClusterState(stacks, state, clusterstate.StateClusterExists); err != nil {
			exit.Error(err)
		}

		loadBalancer, err := getNLBLoadBalancer(accessConfig.ClusterName, OperatorLoadBalancer, awsClient)
		if err != nil {
			exit.Error(err)
		}

		operatorConfig := cluster.OperatorConfig{
			Telemetry:        isTelemetryEnabled(),
			ClientID:         clientID(),
			OperatorEndpoint: "https://" + *loadBalancer.DNSName,
		}

		var apisResponse []schema.APIResponse
		apisResponse, err = cluster.GetAPIs(operatorConfig)
		if err != nil {
			exit.Error(err)
		}
		if len(apisResponse) == 0 {
			fmt.Println(fmt.Sprintf("no apis found in your cluster named %s in %s", accessConfig.ClusterName, accessConfig.Region))
			exit.Ok()
		}

		exportPath := fmt.Sprintf("export-%s-%s", accessConfig.Region, accessConfig.ClusterName)

		err = files.CreateDir(exportPath)
		if err != nil {
			exit.Error(err)
		}

		for _, api := range apisResponse {
			apisWithSpec, err := cluster.GetAPI(operatorConfig, api.Metadata.Name)
			if err != nil {
				exit.Error(err)
			}

			specFilePath := filepath.Join(exportPath, api.Metadata.Name+".yaml")
			fmt.Println(fmt.Sprintf("exporting %s to %s", api.Metadata.Name, specFilePath))

			yamlBytes, err := yaml.Marshal(apisWithSpec[0].Spec.API.SubmittedAPISpec)
			if err != nil {
				exit.Error(err)
			}

			err = files.WriteFile(yamlBytes, specFilePath)
			if err != nil {
				exit.Error(err)
			}
		}
	},
}

var _clusterHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "inspect the health of components in the cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		accessConfig, err := getClusterAccessConfigWithCache(true)
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := awslib.NewForRegion(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}

		restConfig, err := getClusterRESTConfig(awsClient, accessConfig.ClusterName)
		if err != nil {
			exit.Error(err)
		}

		scheme := runtime.NewScheme()
		if err := clientgoscheme.AddToScheme(scheme); err != nil {
			exit.Error(err)
		}

		k8sClient, err := k8s.New(consts.DefaultNamespace, false, restConfig, scheme)
		if err != nil {
			exit.Error(err)
		}

		clusterHealth, err := health.Check(awsClient, k8sClient, accessConfig.ClusterName)
		if err != nil {
			exit.Error(err)
		}

		clusterWarnings, err := health.GetWarnings(k8sClient)
		if err != nil {
			exit.Error(err)
		}

		if _flagOutput == flags.JSONOutputType {
			fmt.Println(clusterHealth)
			return
		}

		healthTable := table.Table{
			Headers: []table.Header{
				{Title: ""},
				{Title: "live"},
				{Title: "warning", Hidden: !clusterWarnings.HasWarnings()},
			},
			Rows: [][]interface{}{
				{"operator", console.BoolColor(clusterHealth.Operator), ""},
				{"prometheus", console.BoolColor(clusterHealth.Prometheus), clusterWarnings.Prometheus},
				{"autoscaler", console.BoolColor(clusterHealth.Autoscaler), ""},
				{"activator", console.BoolColor(clusterHealth.Activator), ""},
				{"async gateway", console.BoolColor(clusterHealth.AsyncGateway), ""},
				{"grafana", console.BoolColor(clusterHealth.Grafana), ""},
				{"controller manager", console.BoolColor(clusterHealth.ControllerManager), ""},
				{"apis gateway", console.BoolColor(clusterHealth.APIsGateway), ""},
				{"operator gateway", console.BoolColor(clusterHealth.APIsGateway), ""},
				{"cluster autoscaler", console.BoolColor(clusterHealth.ClusterAutoscaler), ""},
				{"operator load balancer", console.BoolColor(clusterHealth.OperatorLoadBalancer), ""},
				{"apis load balancer", console.BoolColor(clusterHealth.APIsLoadBalancer), ""},
				{"fluent bit", console.BoolColor(clusterHealth.FluentBit), ""},
				{"node exporter", console.BoolColor(clusterHealth.NodeExporter), ""},
				{"dcgm exporter", console.BoolColor(clusterHealth.DCGMExporter), ""},
				{"statsd exporter", console.BoolColor(clusterHealth.StatsDExporter), ""},
				{"event exporter", console.BoolColor(clusterHealth.EventExporter), ""},
				{"kube state metrics", console.BoolColor(clusterHealth.KubeStateMetrics), ""},
			},
		}

		fmt.Println(healthTable.MustFormat())
	},
}

func cmdPrintConfig(awsClient *awslib.Client, accessConfig *clusterconfig.AccessConfig, outputType flags.OutputType) {
	clusterConfig := refreshCachedClusterConfig(awsClient, accessConfig, outputType == flags.PrettyOutputType)

	infoInterface := clusterConfig.CoreConfig

	if outputType == flags.JSONOutputType {
		outputBytes, err := libjson.Marshal(infoInterface)
		if err != nil {
			exit.Error(err)
		}
		fmt.Println(string(outputBytes))
	} else {
		outputBytes, err := yaml.Marshal(infoInterface)
		if err != nil {
			exit.Error(err)
		}
		fmt.Println(string(outputBytes))
	}
}

func cmdInfo(awsClient *awslib.Client, accessConfig *clusterconfig.AccessConfig, stacks clusterstate.ClusterStacks, outputType flags.OutputType, disallowPrompt bool) {
	clusterConfig := refreshCachedClusterConfig(awsClient, accessConfig, outputType == flags.PrettyOutputType)

	operatorLoadBalancer, err := getNLBLoadBalancer(accessConfig.ClusterName, OperatorLoadBalancer, awsClient)
	if err != nil {
		exit.Error(err)
	}
	operatorEndpoint := s.EnsurePrefix(*operatorLoadBalancer.DNSName, "https://")

	var apiEndpoint string
	if clusterConfig.APILoadBalancerType == clusterconfig.NLBLoadBalancerType {
		apiLoadBalancer, err := getNLBLoadBalancer(accessConfig.ClusterName, APILoadBalancer, awsClient)
		if err != nil {
			exit.Error(err)
		}
		apiEndpoint = *apiLoadBalancer.DNSName
	}
	if clusterConfig.APILoadBalancerType == clusterconfig.ELBLoadBalancerType {
		apiLoadBalancer, err := getELBLoadBalancer(accessConfig.ClusterName, APILoadBalancer, awsClient)
		if err != nil {
			exit.Error(err)
		}
		apiEndpoint = *apiLoadBalancer.DNSName
	}

	if outputType == flags.JSONOutputType || outputType == flags.YAMLOutputType {
		infoResponse, err := getInfoOperatorResponse(operatorEndpoint)
		if err != nil {
			exit.Error(err)
		}
		infoResponse.ClusterConfig.Config = clusterConfig

		infoInterface := map[string]interface{}{
			"cluster_config":      infoResponse.ClusterConfig.Config,
			"cluster_metadata":    infoResponse.ClusterConfig.OperatorMetadata,
			"worker_node_infos":   infoResponse.WorkerNodeInfos,
			"operator_node_infos": infoResponse.OperatorNodeInfos,
			"endpoint_operator":   operatorEndpoint,
			"endpoint_api":        apiEndpoint,
		}

		var outputBytes []byte
		if outputType == flags.JSONOutputType {
			outputBytes, err = libjson.Marshal(infoInterface)
		} else {
			outputBytes, err = yaml.Marshal(infoInterface)
		}
		if err != nil {
			exit.Error(err)
		}
		fmt.Println(string(outputBytes))
	}
	if outputType == flags.PrettyOutputType {
		fmt.Println(console.Bold("endpoints:"))
		fmt.Println("operator:         ", operatorEndpoint)
		fmt.Println("api load balancer:", apiEndpoint)
		fmt.Println()

		if err := printInfoOperatorResponse(clusterConfig, stacks, operatorEndpoint); err != nil {
			exit.Error(err)
		}
	}

	if _flagClusterInfoEnv != "" {
		if err := updateCLIEnv(_flagClusterInfoEnv, operatorEndpoint, disallowPrompt, outputType == flags.PrettyOutputType); err != nil {
			exit.Error(err)
		}
	}
}

func printInfoOperatorResponse(clusterConfig clusterconfig.Config, stacks clusterstate.ClusterStacks, operatorEndpoint string) error {
	fmt.Print("fetching cluster status ...\n\n")

	fmt.Println(stacks.TableString())

	yamlBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return err
	}
	yamlString := string(yamlBytes)

	infoResponse, err := getInfoOperatorResponse(operatorEndpoint)
	if err != nil {
		fmt.Println(yamlString)
		return err
	}
	infoResponse.ClusterConfig.Config = clusterConfig

	fmt.Println(console.Bold("cluster config:"))
	fmt.Println(fmt.Sprintf("cluster version: %s", infoResponse.ClusterConfig.APIVersion))
	fmt.Print(yamlString)

	printInfoPricing(infoResponse, clusterConfig)
	printInfoNodes(infoResponse)

	return nil
}

func getInfoOperatorResponse(operatorEndpoint string) (*schema.InfoResponse, error) {
	operatorConfig := cluster.OperatorConfig{
		Telemetry:        isTelemetryEnabled(),
		ClientID:         clientID(),
		OperatorEndpoint: operatorEndpoint,
	}
	return cluster.Info(operatorConfig)
}

func printInfoPricing(infoResponse *schema.InfoResponse, clusterConfig clusterconfig.Config) {
	eksPrice := awslib.EKSPrices[clusterConfig.Region]
	operatorInstancePrice := awslib.InstanceMetadatas[clusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := awslib.EBSMetadatas[clusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	prometheusInstancePrice := awslib.InstanceMetadatas[clusterConfig.Region][clusterConfig.PrometheusInstanceType].Price
	prometheusEBSPrice := awslib.EBSMetadatas[clusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	metricsEBSPrice := awslib.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * (40 + 2) / 30 / 24
	nlbPrice := awslib.NLBMetadatas[clusterConfig.Region].Price
	elbPrice := awslib.ELBMetadatas[clusterConfig.Region].Price
	natUnitPrice := awslib.NATMetadatas[clusterConfig.Region].Price

	var loadBalancersPrice float64
	usesELBForAPILoadBalancer := clusterConfig.APILoadBalancerType == clusterconfig.ELBLoadBalancerType
	if usesELBForAPILoadBalancer {
		loadBalancersPrice = nlbPrice + elbPrice
	} else {
		loadBalancersPrice = 2 * nlbPrice
	}

	headers := []table.Header{
		{Title: "aws resource"},
		{Title: "cost per hour"},
	}

	var rows [][]interface{}
	rows = append(rows, []interface{}{"1 eks cluster", s.DollarsMaxPrecision(eksPrice)})

	var totalNodeGroupsPrice float64
	for _, ng := range clusterConfig.NodeGroups {
		var ngNamePrefix string
		if ng.Spot {
			ngNamePrefix = "cx-ws-"
		} else {
			ngNamePrefix = "cx-wd-"
		}
		nodesInfo := infoResponse.GetNodesWithNodeGroupName(ngNamePrefix + ng.Name)
		numInstances := len(nodesInfo)

		ebsPrice := awslib.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
		if ng.InstanceVolumeType == clusterconfig.IO1VolumeType && ng.InstanceVolumeIOPS != nil {
			ebsPrice += awslib.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
		}
		if ng.InstanceVolumeType == clusterconfig.GP3VolumeType && ng.InstanceVolumeIOPS != nil && ng.InstanceVolumeThroughput != nil {
			ebsPrice += libmath.MaxFloat64(0, (awslib.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS-3000)*float64(*ng.InstanceVolumeIOPS)/30/24)
			ebsPrice += libmath.MaxFloat64(0, (awslib.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceThroughput-125)*float64(*ng.InstanceVolumeThroughput)/30/24)
		}
		totalEBSPrice := ebsPrice * float64(numInstances)

		totalInstancePrice := float64(0)
		for _, nodeInfo := range nodesInfo {
			totalInstancePrice += nodeInfo.Price
		}

		rows = append(rows, []interface{}{fmt.Sprintf("nodegroup %s: %d (out of %d) %s", ng.Name, numInstances, ng.MaxInstances, s.PluralS("instance", numInstances)), s.DollarsAndTenthsOfCents(totalInstancePrice+totalEBSPrice) + " total"})

		totalNodeGroupsPrice += totalEBSPrice + totalInstancePrice
	}

	operatorNodeGroupPrice := float64(len(infoResponse.OperatorNodeInfos)) * (operatorInstancePrice + operatorEBSPrice)
	prometheusNodeGroupPrice := prometheusInstancePrice + prometheusEBSPrice + metricsEBSPrice

	var natTotalPrice float64
	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(clusterConfig.AvailabilityZones))
	}
	totalPrice := eksPrice + totalNodeGroupsPrice + operatorNodeGroupPrice + prometheusNodeGroupPrice + loadBalancersPrice + natTotalPrice
	fmt.Printf(console.Bold("\nyour cluster currently costs %s per hour\n\n"), s.DollarsAndCents(totalPrice))

	rows = append(rows, []interface{}{fmt.Sprintf("%d t3.medium %s (cortex system)", len(infoResponse.OperatorNodeInfos), s.PluralS("instance", len(infoResponse.OperatorNodeInfos))), s.DollarsAndTenthsOfCents(operatorNodeGroupPrice) + " total"})
	rows = append(rows, []interface{}{fmt.Sprintf("1 %s instance (prometheus)", clusterConfig.PrometheusInstanceType), s.DollarsAndTenthsOfCents(prometheusNodeGroupPrice)})
	if usesELBForAPILoadBalancer {
		rows = append(rows, []interface{}{"1 network load balancer", s.DollarsMaxPrecision(nlbPrice)})
		rows = append(rows, []interface{}{"1 classic load balancer", s.DollarsMaxPrecision(elbPrice)})
	} else {
		rows = append(rows, []interface{}{"2 network load balancers", s.DollarsMaxPrecision(loadBalancersPrice) + " total"})
	}

	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		rows = append(rows, []interface{}{"1 nat gateway", s.DollarsMaxPrecision(natUnitPrice)})
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		numNATs := len(clusterConfig.AvailabilityZones)
		rows = append(rows, []interface{}{fmt.Sprintf("%d nat gateways", numNATs), s.DollarsMaxPrecision(natUnitPrice*float64(numNATs)) + " total"})
	}

	t := table.Table{
		Headers: headers,
		Rows:    rows,
	}
	t.MustPrint(&table.Opts{Sort: pointer.Bool(false)})
}

func printInfoNodes(infoResponse *schema.InfoResponse) {
	numAPIInstances := len(infoResponse.WorkerNodeInfos)

	var totalReplicas int
	var doesClusterHaveGPUs, doesClusterHaveInfs, doesClusterHaveEnqueuers bool
	for _, nodeInfo := range infoResponse.WorkerNodeInfos {
		totalReplicas += nodeInfo.NumReplicas
		if nodeInfo.ComputeUserCapacity.GPU > 0 {
			doesClusterHaveGPUs = true
		}
		if nodeInfo.ComputeUserCapacity.Inf > 0 {
			doesClusterHaveInfs = true
		}
		if nodeInfo.NumEnqueuerReplicas > 0 {
			doesClusterHaveEnqueuers = true
		}
	}

	var pendingReplicasStr string
	if infoResponse.NumPendingReplicas > 0 {
		pendingReplicasStr = fmt.Sprintf(", and %d unscheduled %s", infoResponse.NumPendingReplicas, s.PluralS("replica", infoResponse.NumPendingReplicas))
	}

	fmt.Printf(console.Bold("\nyour cluster has %d API %s running across %d %s%s\n"), totalReplicas, s.PluralS("replica", totalReplicas), numAPIInstances, s.PluralS("instance", numAPIInstances), pendingReplicasStr)

	if len(infoResponse.WorkerNodeInfos) == 0 {
		return
	}

	headers := []table.Header{
		{Title: "instance type"},
		{Title: "lifecycle"},
		{Title: "replicas"},
		{Title: "batch enqueuer replicas", Hidden: !doesClusterHaveEnqueuers},
		{Title: "CPU (requested / total allocatable)"},
		{Title: "memory (requested / total allocatable)"},
		{Title: "GPU (requested / total allocatable)", Hidden: !doesClusterHaveGPUs},
		{Title: "Inf (requested / total allocatable)", Hidden: !doesClusterHaveInfs},
	}

	var rows [][]interface{}
	for _, nodeInfo := range infoResponse.WorkerNodeInfos {
		lifecycle := "on-demand"
		if nodeInfo.IsSpot {
			lifecycle = "spot"
		}

		cpuStr := nodeInfo.ComputeUserRequested.CPU.MilliString() + " / " + nodeInfo.ComputeUserCapacity.CPU.MilliString()
		memStr := nodeInfo.ComputeUserRequested.Mem.String() + " / " + nodeInfo.ComputeUserCapacity.Mem.String()
		gpuStr := s.Int64(nodeInfo.ComputeUserRequested.GPU) + " / " + s.Int64(nodeInfo.ComputeUserCapacity.GPU)
		infStr := s.Int64(nodeInfo.ComputeUserRequested.Inf) + " / " + s.Int64(nodeInfo.ComputeUserCapacity.Inf)
		rows = append(rows, []interface{}{nodeInfo.InstanceType, lifecycle, nodeInfo.NumReplicas, nodeInfo.NumEnqueuerReplicas, cpuStr, memStr, gpuStr, infStr})
	}

	t := table.Table{
		Headers: headers,
		Rows:    rows,
	}
	fmt.Println()
	t.MustPrint(&table.Opts{Sort: pointer.Bool(false)})
}

func updateCLIEnv(envName string, operatorEndpoint string, disallowPrompt bool, printToStdout bool) error {
	prevEnv, err := readEnv(envName)
	if err != nil {
		return err
	}

	newEnvironment := cliconfig.Environment{
		Name:             envName,
		OperatorEndpoint: operatorEndpoint,
	}

	shouldWriteEnv := false
	envWasUpdated := false
	if prevEnv == nil {
		shouldWriteEnv = true
		if printToStdout {
			fmt.Println()
		}
	} else if prevEnv.OperatorEndpoint != operatorEndpoint {
		envWasUpdated = true
		if printToStdout {
			if disallowPrompt {
				shouldWriteEnv = true
				fmt.Println()
			} else {
				shouldWriteEnv = prompt.YesOrNo(fmt.Sprintf("\nfound an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster?", envName), "", "")
			}
		} else {
			shouldWriteEnv = true
		}
	}

	if shouldWriteEnv {
		err := addEnvToCLIConfig(newEnvironment, true)
		if err != nil {
			return err
		}

		if printToStdout {
			if envWasUpdated {
				fmt.Printf(console.Bold("the environment named \"%s\" has been updated to point to this cluster (and was set as the default environment)\n"), envName)
			} else {
				fmt.Printf(console.Bold("an environment named \"%s\" has been configured to point to this cluster (and was set as the default environment)\n"), envName)
			}
		}
	}

	return nil
}

func cmdDebug(awsClient *awslib.Client, accessConfig *clusterconfig.AccessConfig) {
	// note: if modifying this string, also change it in files.IgnoreCortexDebug()
	debugFileName := fmt.Sprintf("cortex-debug-%s.tgz", time.Now().UTC().Format("2006-01-02-15-04-05"))

	containerDebugPath := "/out/" + debugFileName
	copyFromPaths := []dockerCopyFromPath{
		{
			containerPath: containerDebugPath,
			localDir:      _cwd,
		},
	}

	out, exitCode, err := runManagerAccessCommand("/root/debug.sh "+containerDebugPath, *accessConfig, awsClient, nil, copyFromPaths)
	if err != nil {
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		exit.Error(ErrorClusterDebug(out))
	}

	fmt.Println("saved cluster info to ./" + debugFileName)
	return
}

func refreshCachedClusterConfig(awsClient *awslib.Client, accessConfig *clusterconfig.AccessConfig, printToStdout bool) clusterconfig.Config {
	// add empty file if cached cluster doesn't exist so that the file output by manager container maintains current user permissions
	cachedClusterConfigPath := getCachedClusterConfigPath(accessConfig.ClusterName, accessConfig.Region)
	containerConfigPath := fmt.Sprintf("/out/%s", filepath.Base(cachedClusterConfigPath))

	copyFromPaths := []dockerCopyFromPath{
		{
			containerPath: containerConfigPath,
			localDir:      files.Dir(cachedClusterConfigPath),
		},
	}

	if printToStdout {
		fmt.Print("syncing cluster configuration ...\n\n")
	}
	out, exitCode, err := runManagerAccessCommand("/root/refresh.sh "+containerConfigPath, *accessConfig, awsClient, nil, copyFromPaths)
	if err != nil {
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		exit.Error(ErrorClusterRefresh(out))
	}

	refreshedClusterConfig := &clusterconfig.Config{}
	err = readCachedClusterConfigFile(refreshedClusterConfig, cachedClusterConfigPath)
	if err != nil {
		exit.Error(err)
	}
	return *refreshedClusterConfig
}

func createS3BucketIfNotFound(awsClient *awslib.Client, bucket string, tags map[string]string) error {
	bucketFound, err := awsClient.DoesBucketExist(bucket)
	if err != nil {
		return err
	}
	if !bucketFound {
		fmt.Print("￮ creating a new s3 bucket: ", bucket)
		err = awsClient.CreateBucket(bucket)
		if err != nil {
			fmt.Print("\n\n")
			return err
		}
		err = awsClient.EnableBucketEncryption(bucket)
		if err != nil {
			fmt.Print("\n\n")
			return err
		}
	} else {
		fmt.Print("￮ using existing s3 bucket: ", bucket)
	}

	// retry since it's possible that it takes some time for the new bucket to be registered by AWS
	for i := 0; i < 10; i++ {
		err = awsClient.TagBucket(bucket, tags)
		if err == nil {
			fmt.Println(" ✓")
			return nil
		}
		if !awslib.IsNoSuchBucketErr(err) {
			break
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Print("\n\n")
	return err
}

func setLifecycleRulesOnClusterUp(awsClient *awslib.Client, bucket, newClusterUID string) error {
	err := awsClient.DeleteLifecycleRules(bucket)
	if err != nil {
		return err
	}

	clusterUIDs, err := awsClient.ListS3TopLevelDirs(bucket)
	if err != nil {
		return err
	}

	if len(clusterUIDs)+1 > consts.MaxBucketLifecycleRules {
		return ErrorClusterUIDsLimitInBucket(bucket)
	}

	expirationDate := libtime.GetCurrentUTCDate().Add(-24 * time.Hour)
	rules := []s3.LifecycleRule{}
	for _, clusterUID := range clusterUIDs {
		rules = append(rules, s3.LifecycleRule{
			Expiration: &s3.LifecycleExpiration{
				Date: &expirationDate,
			},
			ID: pointer.String("cluster-remove-" + clusterUID),
			Filter: &s3.LifecycleRuleFilter{
				Prefix: pointer.String(s.EnsureSuffix(clusterUID, "/")),
			},
			Status: pointer.String("Enabled"),
		})
	}

	rules = append(rules, s3.LifecycleRule{
		Expiration: &s3.LifecycleExpiration{
			Days: pointer.Int64(consts.AsyncWorkloadsExpirationDays),
		},
		ID: pointer.String("async-workloads-expiry-policy"),
		Filter: &s3.LifecycleRuleFilter{
			Prefix: pointer.String(s.EnsureSuffix(filepath.Join(newClusterUID, "workloads"), "/")),
		},
		Status: pointer.String("Enabled"),
	})

	return awsClient.SetLifecycleRules(bucket, rules)
}

func setLifecycleRulesOnClusterDown(awsClient *awslib.Client, bucket string) error {
	err := awsClient.DeleteLifecycleRules(bucket)
	if err != nil {
		return err
	}

	expirationDate := libtime.GetCurrentUTCDate().Add(-24 * time.Hour)
	return awsClient.SetLifecycleRules(bucket, []s3.LifecycleRule{
		{
			Expiration: &s3.LifecycleExpiration{
				Date: &expirationDate,
			},
			ID: pointer.String("bucket-cleaner"),
			Filter: &s3.LifecycleRuleFilter{
				Prefix: pointer.String(""),
			},
			Status: pointer.String("Enabled"),
		},
	})
}

func createLogGroupIfNotFound(awsClient *awslib.Client, logGroup string, tags map[string]string) error {
	logGroupFound, err := awsClient.DoesLogGroupExist(logGroup)
	if err != nil {
		return err
	}
	if !logGroupFound {
		fmt.Print("￮ creating a new cloudwatch log group: ", logGroup)
		err = awsClient.CreateLogGroup(logGroup, tags)
		if err != nil {
			fmt.Print("\n\n")
			return err
		}
		fmt.Println(" ✓")
		return nil
	}

	fmt.Print("￮ using existing cloudwatch log group: ", logGroup)

	// retry since it's possible that it takes some time for the new log group to be registered by AWS
	err = awsClient.TagLogGroup(logGroup, tags)
	if err != nil {
		fmt.Print("\n\n")
		return err
	}

	fmt.Println(" ✓")

	return nil
}

type LoadBalancer string

var (
	OperatorLoadBalancer LoadBalancer = "operator"
	APILoadBalancer      LoadBalancer = "api"
)

func (lb LoadBalancer) String() string {
	return string(lb)
}

// Will return error if the load balancer can't be found
func getNLBLoadBalancer(clusterName string, whichLB LoadBalancer, awsClient *awslib.Client) (*elbv2.LoadBalancer, error) {
	loadBalancer, err := awsClient.FindLoadBalancerV2(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   whichLB.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to locate %s load balancer", whichLB.String()))
	}

	if loadBalancer == nil {
		return nil, ErrorNoOperatorLoadBalancer(whichLB.String())
	}

	return loadBalancer, nil
}

// Will return error if the load balancer can't be found
func getELBLoadBalancer(clusterName string, whichLB LoadBalancer, awsClient *awslib.Client) (*elb.LoadBalancerDescription, error) {
	loadBalancer, err := awsClient.FindLoadBalancer(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   whichLB.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to locate %s load balancer", whichLB.String()))
	}

	if loadBalancer == nil {
		return nil, ErrorNoOperatorLoadBalancer(whichLB.String())
	}

	return loadBalancer, nil
}

func listPVCVolumesForCluster(awsClient *awslib.Client, clusterName string) ([]ec2.Volume, error) {
	return awsClient.ListVolumes(ec2.Tag{
		Key:   pointer.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)),
		Value: nil, // any value should be ok as long as the key is present
	})
}

func filterEKSCTLOutput(out string) string {
	return strings.Join(s.RemoveDuplicates(strings.Split(out, "\n"), _eksctlPrefixRegex), "\n")
}

func getClusterRESTConfig(awsClient *awslib.Client, clusterName string) (*rest.Config, error) {
	clusterOutput, err := awsClient.EKS().DescribeCluster(
		&eks.DescribeClusterInput{
			Name: aws.String(clusterName),
		},
	)
	if err != nil {
		return nil, err
	}

	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, err
	}

	opts := &token.GetTokenOptions{
		ClusterID: aws.StringValue(clusterOutput.Cluster.Name),
	}

	tok, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, err
	}

	ca, err := base64.StdEncoding.DecodeString(aws.StringValue(clusterOutput.Cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		Host:        aws.StringValue(clusterOutput.Cluster.Endpoint),
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
	}, nil
}
