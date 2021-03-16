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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/archive"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/clusterstate"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/yaml"
	"github.com/spf13/cobra"
)

var (
	_flagClusterUpEnv              string
	_flagClusterInfoEnv            string
	_flagClusterConfigureEnv       string
	_flagClusterConfig             string
	_flagClusterName               string
	_flagClusterRegion             string
	_flagClusterInfoDebug          bool
	_flagClusterDisallowPrompt     bool
	_flagClusterDownKeepVolumes    bool
	_flagAWSAccessKeyID            string
	_flagAWSSecretAccessKey        string
	_flagClusterAWSAccessKeyID     string
	_flagClusterAWSSecretAccessKey string
)

func clusterInit() {
	_clusterUpCmd.Flags().SortFlags = false
	_clusterUpCmd.Flags().StringVarP(&_flagClusterUpEnv, "configure-env", "e", "aws", "name of environment to configure")
	_clusterUpCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterUpCmd)

	_clusterInfoCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterInfoCmd)
	addClusterNameFlag(_clusterInfoCmd)
	addClusterRegionFlag(_clusterInfoCmd)
	_clusterInfoCmd.Flags().StringVarP(&_flagClusterInfoEnv, "configure-env", "e", "", "name of environment to configure")
	_clusterInfoCmd.Flags().BoolVarP(&_flagClusterInfoDebug, "debug", "d", false, "save the current cluster state to a file")
	_clusterInfoCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterInfoCmd)

	_clusterConfigureCmd.Flags().SortFlags = false
	_clusterConfigureCmd.Flags().StringVarP(&_flagClusterConfigureEnv, "configure-env", "e", "", "name of environment to configure")
	_clusterConfigureCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_clusterConfigureCmd)

	_clusterDownCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterDownCmd)
	addClusterNameFlag(_clusterDownCmd)
	addClusterRegionFlag(_clusterDownCmd)
	_clusterDownCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterDownCmd.Flags().BoolVar(&_flagClusterDownKeepVolumes, "keep-volumes", false, "keep cortex provisioned persistent volumes")
	_clusterCmd.AddCommand(_clusterDownCmd)

	_clusterExportCmd.Flags().SortFlags = false
	addClusterConfigFlag(_clusterExportCmd)
	addClusterNameFlag(_clusterExportCmd)
	addClusterRegionFlag(_clusterExportCmd)
	_clusterCmd.AddCommand(_clusterExportCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterConfig, "config", "c", "", "path to a cluster configuration file")
	cmd.Flags().SetAnnotation("config", cobra.BashCompFilenameExt, _configFileExts)
}

func addClusterNameFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterName, "name", "n", "", "name of the cluster")
}

func addClusterRegionFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterRegion, "region", "r", "", "aws region of the cluster")
}

var _clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage AWS clusters (contains subcommands)",
}

var _clusterUpCmd = &cobra.Command{
	Use:   "up [CLUSTER_CONFIG_FILE]",
	Short: "spin up a cluster on aws",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.EventNotify("cli.cluster.up", map[string]interface{}{"provider": types.AWSProviderType})

		clusterConfigFile := args[0]

		envExists, err := isEnvConfigured(_flagClusterUpEnv)
		if err != nil {
			exit.Error(err)
		}
		if envExists {
			if _flagClusterDisallowPrompt {
				fmt.Printf("found an existing environment named \"%s\", which will be overwritten to connect to this cluster once it's created\n\n", _flagClusterUpEnv)
			} else {
				prompt.YesOrExit(fmt.Sprintf("found an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster once it's created?", _flagClusterUpEnv), "", "you can specify a different environment name to be configured to connect to this cluster by specifying the --configure-env flag (e.g. `cortex cluster up --configure-env prod`); or you can list your environments with `cortex env list` and delete an environment with `cortex env delete ENV_NAME`")
			}
		}

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getNewClusterAccessConfig(clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := newAWSClient(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}

		clusterConfig, err := getInstallClusterConfig(awsClient, clusterConfigFile, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		err = clusterstate.AssertClusterStatus(accessConfig.ClusterName, accessConfig.Region, clusterState.Status, clusterstate.StatusNotFound, clusterstate.StatusDeleteComplete)
		if err != nil {
			exit.Error(err)
		}

		err = createS3BucketIfNotFound(awsClient, clusterConfig.Bucket, clusterConfig.Tags)
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

		out, exitCode, err := runManagerWithClusterConfig("/root/install.sh", clusterConfig, awsClient, nil, nil)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			eksCluster, err := awsClient.EKSClusterOrNil(clusterConfig.ClusterName)
			if err != nil {
				helpStr := "\ndebugging tips (may or may not apply to this error):"
				helpStr += fmt.Sprintf("\n* if your cluster started spinning up but was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
				helpStr += "\n* if your cluster started spinning up, please run `cortex cluster down` to delete the cluster before trying to create this cluster again"
				fmt.Println(helpStr)
				exit.Error(ErrorClusterUp(out + helpStr))
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

		loadBalancer, err := getAWSOperatorLoadBalancer(clusterConfig.ClusterName, awsClient)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterUpEnv)))
		}

		newEnvironment := cliconfig.Environment{
			Name:             _flagClusterUpEnv,
			Provider:         types.AWSProviderType,
			OperatorEndpoint: "https://" + *loadBalancer.DNSName,
		}

		err = addEnvToCLIConfig(newEnvironment, true)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterUpEnv)))
		}

		if envExists {
			fmt.Printf(console.Bold("\nthe environment named \"%s\" has been updated to point to this cluster (and was set as the default environment)\n"), _flagClusterUpEnv)
		} else {
			fmt.Printf(console.Bold("\nan environment named \"%s\" has been configured to point to this cluster (and was set as the default environment)\n"), _flagClusterUpEnv)
		}
	},
}

var _clusterConfigureCmd = &cobra.Command{
	Use:   "configure [CLUSTER_CONFIG_FILE]",
	Short: "update a cluster's configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.configure", map[string]interface{}{"provider": types.AWSProviderType})

		clusterConfigFile := args[0]

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getNewClusterAccessConfig(clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := newAWSClient(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		err = clusterstate.AssertClusterStatus(accessConfig.ClusterName, accessConfig.Region, clusterState.Status, clusterstate.StatusCreateComplete, clusterstate.StatusUpdateComplete, clusterstate.StatusUpdateRollbackComplete)
		if err != nil {
			exit.Error(err)
		}

		cachedClusterConfig := refreshCachedClusterConfig(*awsClient, accessConfig, _flagClusterDisallowPrompt)

		clusterConfig, err := getConfigureClusterConfig(cachedClusterConfig, clusterConfigFile, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerWithClusterConfig("/root/install.sh --update", clusterConfig, awsClient, nil, nil)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := "\ndebugging tips (may or may not apply to this error):"
			helpStr += fmt.Sprintf("\n* if your cluster was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the  \"Activity\" or \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", clusterConfig.Region)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterConfigure(out + helpStr))
		}

		if _flagClusterConfigureEnv != "" {
			loadBalancer, err := getAWSOperatorLoadBalancer(clusterConfig.ClusterName, awsClient)
			if err != nil {
				exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterConfigureEnv)))
			}
			operatorEndpoint := "https://" + *loadBalancer.DNSName
			err = updateAWSCLIEnv(_flagClusterConfigureEnv, operatorEndpoint, _flagClusterDisallowPrompt)
			if err != nil {
				exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterConfigureEnv)))
			}
		}
	},
}

var _clusterInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.info", map[string]interface{}{"provider": types.AWSProviderType})

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfigWithCache()
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := newAWSClient(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}

		if _flagClusterInfoDebug {
			cmdDebug(awsClient, accessConfig)
		} else {
			cmdInfo(awsClient, accessConfig, _flagClusterDisallowPrompt)
		}
	},
}

var _clusterDownCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.down", map[string]interface{}{"provider": types.AWSProviderType})

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfigWithCache()
		if err != nil {
			exit.Error(err)
		}

		// Check AWS access
		awsClient, err := newAWSClient(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}

		accountID, _, err := awsClient.GetCachedAccountID()
		if err != nil {
			exit.Error(err)
		}

		warnIfNotAdmin(awsClient)

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil && errors.GetKind(err) != clusterstate.ErrUnexpectedCloudFormationStatus {
			exit.Error(err)
		}
		if err == nil {
			switch clusterState.Status {
			case clusterstate.StatusNotFound:
				exit.Error(clusterstate.ErrorClusterDoesNotExist(accessConfig.ClusterName, accessConfig.Region))
			case clusterstate.StatusDeleteComplete:
				// silently clean up
				awsClient.DeleteQueuesWithPrefix(clusterconfig.SQSNamePrefix(accessConfig.ClusterName))
				awsClient.DeletePolicy(clusterconfig.DefaultPolicyARN(accountID, accessConfig.ClusterName, accessConfig.Region))
				if !_flagClusterDownKeepVolumes {
					volumes, err := listPVCVolumesForCluster(awsClient, accessConfig.ClusterName)
					if err == nil {
						for _, volume := range volumes {
							awsClient.DeleteVolume(*volume.VolumeId)
						}
					}
				}
				exit.Error(clusterstate.ErrorClusterAlreadyDeleted(accessConfig.ClusterName, accessConfig.Region))
			}
		}

		// updating CLI env is best-effort, so ignore errors
		loadBalancer, _ := getAWSOperatorLoadBalancer(accessConfig.ClusterName, awsClient)

		if _flagClusterDisallowPrompt {
			fmt.Printf("your cluster named \"%s\" in %s will be spun down and all apis will be deleted\n\n", accessConfig.ClusterName, accessConfig.Region)
		} else {
			prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s will be spun down and all apis will be deleted, are you sure you want to continue?", accessConfig.ClusterName, accessConfig.Region), "", "")
		}

		fmt.Print("￮ deleting sqs queues ")
		err = awsClient.DeleteQueuesWithPrefix(clusterconfig.SQSNamePrefix(accessConfig.ClusterName))
		if err != nil {
			fmt.Printf("\n\nfailed to delete all sqs queues; please delete queues starting with the name %s via the cloudwatch console: https://%s.console.aws.amazon.com/sqs/v2/home\n", clusterconfig.SQSNamePrefix(accessConfig.ClusterName), accessConfig.Region)
			errors.PrintError(err)
			fmt.Println()
		} else {
			fmt.Println("✓")
		}

		fmt.Print("￮ spinning down the cluster ...")

		out, exitCode, err := runManagerAccessCommand("/root/uninstall.sh", *accessConfig, awsClient, nil, nil)
		if err != nil {
			errors.PrintError(err)
			fmt.Println()
		} else if exitCode == nil || *exitCode != 0 {
			helpStr := fmt.Sprintf("\nNote: if this error cannot be resolved, please ensure that all CloudFormation stacks for this cluster eventually become fully deleted (%s). If the stack deletion process has failed, please delete the stacks directly from the AWS console (this may require manually deleting particular AWS resources that are blocking the stack deletion)", clusterstate.CloudFormationURL(accessConfig.ClusterName, accessConfig.Region))
			fmt.Println(helpStr)
			exit.Error(ErrorClusterDown(out + helpStr))
		}

		// delete policy after spinning down the cluster (which deletes the roles) because policies can't be deleted if they are attached to roles
		policyARN := clusterconfig.DefaultPolicyARN(accountID, accessConfig.ClusterName, accessConfig.Region)
		fmt.Printf("￮ deleting auto-generated iam policy %s ", policyARN)
		err = awsClient.DeletePolicy(policyARN)
		if err != nil {
			fmt.Printf("\n\nfailed to delete auto-generated cortex policy %s; please delete the policy via the iam console: https://console.aws.amazon.com/iam/home#/policies\n", policyARN)
			errors.PrintError(err)
			fmt.Println()
		} else {
			fmt.Println("✓")
		}

		// delete EBS volumes
		if !_flagClusterDownKeepVolumes {
			volumes, err := listPVCVolumesForCluster(awsClient, accessConfig.ClusterName)
			if err != nil {
				fmt.Println("\nfailed to list volumes for deletion; please delete any volumes associated with your cluster via the ec2 console: https://console.aws.amazon.com/ec2/v2/home?#Volumes")
				errors.PrintError(err)
				fmt.Println()
			} else {
				fmt.Print("￮ deleting ebs volumes ")
				var failedToDeleteVolumes []string
				var lastErr error
				for _, volume := range volumes {
					err := awsClient.DeleteVolume(*volume.VolumeId)
					if err != nil {
						failedToDeleteVolumes = append(failedToDeleteVolumes, *volume.VolumeId)
						lastErr = err
					}
				}
				if lastErr != nil {
					fmt.Printf("\n\nfailed to delete %s %s; please delete %s via the ec2 console: https://console.aws.amazon.com/ec2/v2/home?#Volumes\n", s.PluralS("volume", len(failedToDeleteVolumes)), s.UserStrsAnd(failedToDeleteVolumes), s.PluralCustom("it", "them", len(failedToDeleteVolumes)))
					errors.PrintError(lastErr)
					fmt.Println()
				} else {
					fmt.Println("✓")
				}
			}
		}

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

		fmt.Printf("\nplease check CloudFormation to ensure that all resources for the %s cluster eventually become successfully deleted: %s\n", accessConfig.ClusterName, clusterstate.CloudFormationURL(accessConfig.ClusterName, accessConfig.Region))

		cachedClusterConfigPath := cachedClusterConfigPath(accessConfig.ClusterName, accessConfig.Region)
		os.Remove(cachedClusterConfigPath)
	},
}

var _clusterExportCmd = &cobra.Command{
	Use:   "export [API_NAME] [API_ID]",
	Short: "download the code and configuration for APIs",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.export", map[string]interface{}{"provider": types.AWSProviderType})

		accessConfig, err := getClusterAccessConfigWithCache()
		if err != nil {
			exit.Error(err)
		}

		// Check AWS access
		awsClient, err := newAWSClient(accessConfig.Region)
		if err != nil {
			exit.Error(err)
		}
		warnIfNotAdmin(awsClient)

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil {
			exit.Error(err)
		}

		err = clusterstate.AssertClusterStatus(accessConfig.ClusterName, accessConfig.Region, clusterState.Status, clusterstate.StatusCreateComplete, clusterstate.StatusUpdateComplete, clusterstate.StatusUpdateRollbackComplete)
		if err != nil {
			exit.Error(err)
		}

		loadBalancer, err := getAWSOperatorLoadBalancer(accessConfig.ClusterName, awsClient)
		if err != nil {
			exit.Error(err)
		}

		operatorConfig := cluster.OperatorConfig{
			Telemetry:        isTelemetryEnabled(),
			ClientID:         clientID(),
			OperatorEndpoint: "https://" + *loadBalancer.DNSName,
			Provider:         types.AWSProviderType,
		}

		info, err := cluster.Info(operatorConfig)
		if err != nil {
			exit.Error(err)
		}

		var apisResponse []schema.APIResponse
		if len(args) == 0 {
			apisResponse, err = cluster.GetAPIs(operatorConfig)
			if err != nil {
				exit.Error(err)
			}
			if len(apisResponse) == 0 {
				fmt.Println(fmt.Sprintf("no apis found in your cluster named %s in %s", accessConfig.ClusterName, accessConfig.Region))
				exit.Ok()
			}
		} else if len(args) == 1 {
			apisResponse, err = cluster.GetAPI(operatorConfig, args[0])
			if err != nil {
				exit.Error(err)
			}
		} else if len(args) == 2 {
			apisResponse, err = cluster.GetAPIByID(operatorConfig, args[0], args[1])
			if err != nil {
				exit.Error(err)
			}
		}

		exportPath := fmt.Sprintf("export-%s-%s", accessConfig.Region, accessConfig.ClusterName)

		err = files.CreateDir(exportPath)
		if err != nil {
			exit.Error(err)
		}

		for _, apiResponse := range apisResponse {
			baseDir := filepath.Join(exportPath, apiResponse.Spec.Name, apiResponse.Spec.ID)

			fmt.Println(fmt.Sprintf("exporting %s to %s", apiResponse.Spec.Name, baseDir))

			err = files.CreateDir(baseDir)
			if err != nil {
				exit.Error(err)
			}

			yamlBytes, err := yaml.Marshal(apiResponse.Spec.API.SubmittedAPISpec)
			if err != nil {
				exit.Error(err)
			}

			err = files.WriteFile(yamlBytes, path.Join(baseDir, apiResponse.Spec.FileName))
			if err != nil {
				exit.Error(err)
			}

			if apiResponse.Spec.Kind != userconfig.TrafficSplitterKind {
				zipFileLocation := path.Join(baseDir, path.Base(apiResponse.Spec.ProjectKey))
				err = awsClient.DownloadFileFromS3(info.ClusterConfig.Bucket, apiResponse.Spec.ProjectKey, zipFileLocation)
				if err != nil {
					exit.Error(err)
				}

				_, err = archive.UnzipFileToDir(zipFileLocation, baseDir)
				if err != nil {
					exit.Error(err)
				}

				err := os.Remove(zipFileLocation)
				if err != nil {
					exit.Error(err)
				}
			}
		}
	},
}

func cmdInfo(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig, disallowPrompt bool) {
	if err := printInfoClusterState(awsClient, accessConfig); err != nil {
		exit.Error(err)
	}

	clusterConfig := refreshCachedClusterConfig(*awsClient, accessConfig, disallowPrompt)

	out, exitCode, err := runManagerWithClusterConfig("/root/info.sh", &clusterConfig, awsClient, nil, nil)
	if err != nil {
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		exit.Error(ErrorClusterInfo(out))
	}

	fmt.Println()

	var operatorEndpoint string
	for _, line := range strings.Split(out, "\n") {
		// before modifying this, search for this prefix
		if strings.HasPrefix(line, "operator: ") {
			operatorEndpoint = "https://" + strings.TrimSpace(strings.TrimPrefix(line, "operator: "))
			break
		}
	}

	if err := printInfoOperatorResponse(clusterConfig, operatorEndpoint); err != nil {
		exit.Error(err)
	}

	if _flagClusterInfoEnv != "" {
		if err := updateAWSCLIEnv(_flagClusterInfoEnv, operatorEndpoint, disallowPrompt); err != nil {
			exit.Error(err)
		}
	}
}

func printInfoClusterState(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig) error {
	clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
	if err != nil {
		return err
	}

	fmt.Println(clusterState.TableString())
	if clusterState.Status == clusterstate.StatusCreateFailed || clusterState.Status == clusterstate.StatusDeleteFailed {
		fmt.Println(fmt.Sprintf("more information can be found in your AWS console: %s", clusterstate.CloudFormationURL(accessConfig.ClusterName, accessConfig.Region)))
		fmt.Println()
	}

	err = clusterstate.AssertClusterStatus(accessConfig.ClusterName, accessConfig.Region, clusterState.Status, clusterstate.StatusCreateComplete, clusterstate.StatusUpdateComplete, clusterstate.StatusUpdateRollbackComplete)
	if err != nil {
		return err
	}

	return nil
}

func printInfoOperatorResponse(clusterConfig clusterconfig.Config, operatorEndpoint string) error {
	fmt.Print("fetching cluster status ...\n\n")

	yamlBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return err
	}
	yamlString := string(yamlBytes)

	operatorConfig := cluster.OperatorConfig{
		Telemetry:        isTelemetryEnabled(),
		ClientID:         clientID(),
		OperatorEndpoint: operatorEndpoint,
		Provider:         types.AWSProviderType,
	}

	infoResponse, err := cluster.Info(operatorConfig)
	if err != nil {
		fmt.Println(yamlString)
		return err
	}
	infoResponse.ClusterConfig.Config = clusterConfig

	fmt.Println(console.Bold("metadata:"))
	fmt.Println(fmt.Sprintf("aws access key id: %s", infoResponse.MaskedAWSAccessKeyID))
	fmt.Println(fmt.Sprintf("%s: %s", clusterconfig.APIVersionUserKey, infoResponse.ClusterConfig.APIVersion))

	fmt.Println()
	fmt.Println(console.Bold("cluster config:"))
	fmt.Print(yamlString)

	printInfoPricing(infoResponse, clusterConfig)
	printInfoNodes(infoResponse)

	return nil
}

func printInfoPricing(infoResponse *schema.InfoResponse, clusterConfig clusterconfig.Config) {
	eksPrice := aws.EKSPrices[clusterConfig.Region]
	operatorInstancePrice := aws.InstanceMetadatas[clusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * 20 / 30 / 24
	metricsEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * 40 / 30 / 24
	nlbPrice := aws.NLBMetadatas[clusterConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[clusterConfig.Region].Price

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

		ebsPrice := aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
		if ng.InstanceVolumeType.String() == "io1" && ng.InstanceVolumeIOPS != nil {
			ebsPrice += aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
		}
		totalEBSPrice := ebsPrice * float64(numInstances)

		totalInstancePrice := float64(0)
		for _, nodeInfo := range nodesInfo {
			totalInstancePrice += nodeInfo.Price
		}

		rows = append(rows, []interface{}{fmt.Sprintf("nodegroup %s: %d (out of %d) %s for your apis", ng.Name, numInstances, ng.MaxInstances, s.PluralS("instance", numInstances)), s.DollarsAndTenthsOfCents(totalInstancePrice) + " total"})
		rows = append(rows, []interface{}{fmt.Sprintf("nodegroup %s: %d (out of %d) %dgb ebs %s for your apis", ng.Name, numInstances, ng.MaxInstances, ng.InstanceVolumeSize, s.PluralS("volume", numInstances)), s.DollarsAndTenthsOfCents(totalEBSPrice) + " total"})

		totalNodeGroupsPrice += totalEBSPrice + totalInstancePrice
	}

	var natTotalPrice float64
	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(clusterConfig.AvailabilityZones))
	}
	totalPrice := eksPrice + totalNodeGroupsPrice + operatorInstancePrice*2 + operatorEBSPrice + metricsEBSPrice + nlbPrice*2 + natTotalPrice
	fmt.Printf(console.Bold("\nyour cluster currently costs %s per hour\n\n"), s.DollarsAndCents(totalPrice))

	rows = append(rows, []interface{}{"2 t3.medium instances for cortex", s.DollarsMaxPrecision(operatorInstancePrice * 2)})
	rows = append(rows, []interface{}{"1 20gb ebs volume for the operator", s.DollarsAndTenthsOfCents(operatorEBSPrice)})
	rows = append(rows, []interface{}{"1 40gb ebs volume for prometheus", s.DollarsAndTenthsOfCents(metricsEBSPrice)})
	rows = append(rows, []interface{}{"2 network load balancers", s.DollarsMaxPrecision(nlbPrice*2) + " total"})

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
	numAPIInstances := len(infoResponse.NodeInfos)

	var totalReplicas int
	var doesClusterHaveGPUs, doesClusterHaveInfs bool
	for _, nodeInfo := range infoResponse.NodeInfos {
		totalReplicas += nodeInfo.NumReplicas
		if nodeInfo.ComputeUserCapacity.GPU > 0 {
			doesClusterHaveGPUs = true
		}
		if nodeInfo.ComputeUserCapacity.Inf > 0 {
			doesClusterHaveInfs = true
		}
	}

	var pendingReplicasStr string
	if infoResponse.NumPendingReplicas > 0 {
		pendingReplicasStr = fmt.Sprintf(", and %d unscheduled %s", infoResponse.NumPendingReplicas, s.PluralS("replica", infoResponse.NumPendingReplicas))
	}

	fmt.Printf(console.Bold("\nyour cluster has %d API %s running across %d %s%s\n"), totalReplicas, s.PluralS("replica", totalReplicas), numAPIInstances, s.PluralS("instance", numAPIInstances), pendingReplicasStr)

	if len(infoResponse.NodeInfos) == 0 {
		return
	}

	headers := []table.Header{
		{Title: "instance type"},
		{Title: "lifecycle"},
		{Title: "replicas"},
		{Title: "CPU (requested / total allocatable)"},
		{Title: "memory (requested / total allocatable)"},
		{Title: "GPU (requested / total allocatable)", Hidden: !doesClusterHaveGPUs},
		{Title: "Inf (requested / total allocatable)", Hidden: !doesClusterHaveInfs},
	}

	var rows [][]interface{}
	for _, nodeInfo := range infoResponse.NodeInfos {
		lifecycle := "on-demand"
		if nodeInfo.IsSpot {
			lifecycle = "spot"
		}

		cpuStr := nodeInfo.ComputeUserRequested.CPU.MilliString() + " / " + nodeInfo.ComputeUserCapacity.CPU.MilliString()
		memStr := nodeInfo.ComputeUserRequested.Mem.String() + " / " + nodeInfo.ComputeUserCapacity.Mem.String()
		gpuStr := s.Int64(nodeInfo.ComputeUserRequested.GPU) + " / " + s.Int64(nodeInfo.ComputeUserCapacity.GPU)
		infStr := s.Int64(nodeInfo.ComputeUserRequested.Inf) + " / " + s.Int64(nodeInfo.ComputeUserCapacity.Inf)
		rows = append(rows, []interface{}{nodeInfo.InstanceType, lifecycle, nodeInfo.NumReplicas, cpuStr, memStr, gpuStr, infStr})
	}

	t := table.Table{
		Headers: headers,
		Rows:    rows,
	}
	fmt.Println()
	t.MustPrint(&table.Opts{Sort: pointer.Bool(false)})
}

func updateAWSCLIEnv(envName string, operatorEndpoint string, disallowPrompt bool) error {
	prevEnv, err := readEnv(envName)
	if err != nil {
		return err
	}

	newEnvironment := cliconfig.Environment{
		Name:             envName,
		Provider:         types.AWSProviderType,
		OperatorEndpoint: operatorEndpoint,
	}

	shouldWriteEnv := false
	envWasUpdated := false
	if prevEnv == nil {
		shouldWriteEnv = true
		fmt.Println()
	} else if prevEnv.OperatorEndpoint != operatorEndpoint {
		envWasUpdated = true
		if disallowPrompt {
			shouldWriteEnv = true
			fmt.Println()
		} else {
			shouldWriteEnv = prompt.YesOrNo(fmt.Sprintf("\nfound an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster?", envName), "", "")
		}
	}

	if shouldWriteEnv {
		err := addEnvToCLIConfig(newEnvironment, true)
		if err != nil {
			return err
		}

		if envWasUpdated {
			fmt.Printf(console.Bold("the environment named \"%s\" has been updated to point to this cluster (and was set as the default environment)\n"), envName)
		} else {
			fmt.Printf(console.Bold("an environment named \"%s\" has been configured to point to this cluster (and was set as the default environment)\n"), envName)
		}
	}

	return nil
}

func cmdDebug(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig) {
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

func refreshCachedClusterConfig(awsClient aws.Client, accessConfig *clusterconfig.AccessConfig, disallowPrompt bool) clusterconfig.Config {
	// add empty file if cached cluster doesn't exist so that the file output by manager container maintains current user permissions
	cachedClusterConfigPath := cachedClusterConfigPath(accessConfig.ClusterName, accessConfig.Region)
	containerConfigPath := fmt.Sprintf("/out/%s", filepath.Base(cachedClusterConfigPath))

	copyFromPaths := []dockerCopyFromPath{
		{
			containerPath: containerConfigPath,
			localDir:      files.Dir(cachedClusterConfigPath),
		},
	}

	fmt.Print("syncing cluster configuration ...\n\n")
	out, exitCode, err := runManagerAccessCommand("/root/refresh.sh "+containerConfigPath, *accessConfig, &awsClient, nil, copyFromPaths)
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

func createS3BucketIfNotFound(awsClient *aws.Client, bucket string, tags map[string]string) error {
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
		if !aws.IsNoSuchBucketErr(err) {
			break
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Print("\n\n")
	return err
}

func createLogGroupIfNotFound(awsClient *aws.Client, logGroup string, tags map[string]string) error {
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

// Will return error if load balancer can't be found
func getAWSOperatorLoadBalancer(clusterName string, awsClient *aws.Client) (*elbv2.LoadBalancer, error) {
	loadBalancer, err := awsClient.FindLoadBalancer(map[string]string{
		clusterconfig.ClusterNameTag: clusterName,
		"cortex.dev/load-balancer":   "operator",
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to locate operator load balancer")
	}

	if loadBalancer == nil {
		return nil, ErrorNoOperatorLoadBalancer()
	}

	return loadBalancer, nil
}

func listPVCVolumesForCluster(awsClient *aws.Client, clusterName string) ([]ec2.Volume, error) {
	return awsClient.ListVolumes(ec2.Tag{
		Key:   pointer.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)),
		Value: nil, // any value should be ok as long as the key is present
	})
}
