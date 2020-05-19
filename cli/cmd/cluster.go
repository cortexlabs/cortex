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

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
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
	"github.com/spf13/cobra"
)

var (
	_flagClusterEnv            string
	_flagClusterConfig         string
	_flagClusterInfoDebug      bool
	_flagClusterDisallowPrompt bool
)

func clusterInit() {
	defaultEnv := getDefaultEnv(_clusterCommandType)

	_upCmd.Flags().SortFlags = false
	addClusterConfigFlag(_upCmd)
	_upCmd.Flags().StringVarP(&_flagClusterEnv, "env", "e", defaultEnv, "environment to configure")
	_upCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_upCmd)

	_infoCmd.Flags().SortFlags = false
	addClusterConfigFlag(_infoCmd)
	_infoCmd.Flags().StringVarP(&_flagClusterEnv, "env", "e", defaultEnv, "environment to configure")
	_infoCmd.Flags().BoolVarP(&_flagClusterInfoDebug, "debug", "d", false, "save the current cluster state to a file")
	_infoCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_infoCmd)

	_configureCmd.Flags().SortFlags = false
	addClusterConfigFlag(_configureCmd)
	_configureCmd.Flags().StringVarP(&_flagClusterEnv, "env", "e", defaultEnv, "environment to configure")
	_configureCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_configureCmd)

	_downCmd.Flags().SortFlags = false
	addClusterConfigFlag(_downCmd)
	_downCmd.Flags().BoolVarP(&_flagClusterDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterCmd.AddCommand(_downCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterConfig, "config", "c", "", "path to a cluster configuration file")
	cmd.Flags().SetAnnotation("config", cobra.BashCompFilenameExt, _configFileExts)
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

		if _flagClusterEnv == "local" {
			exit.Error(ErrorNotSupportedInLocalEnvironment())
		}

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		if !_flagClusterDisallowPrompt {
			promptForEmail()
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig, _flagClusterEnv, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		clusterConfig, err := getInstallClusterConfig(awsCreds, _flagClusterEnv, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		accessConfig := clusterConfig.ToAccessConfig()

		awsClient, err := newAWSClient(*accessConfig.Region, awsCreds)
		if err != nil {
			exit.Error(err)
		}

		clusterState, err := clusterstate.GetClusterState(awsClient, &accessConfig)
		if err != nil {
			if errors.GetKind(err) == clusterstate.ErrUnexpectedCloudFormationStatus {
				fmt.Println(fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; please run `cortex cluster down` to delete the cluster, or delete the cloudformation stacks manually in your AWS console (%s)", clusterConfig.ClusterName, *clusterConfig.Region, getCloudFormationURL(*clusterConfig.Region, clusterConfig.ClusterName)))
			}
			exit.Error(err)
		}

		err = assertClusterStatus(&accessConfig, clusterState.Status, clusterstate.StatusNotFound, clusterstate.StatusDeleteComplete)
		if err != nil {
			exit.Error(err)
		}

		err = CreateBucketIfNotFound(awsClient, clusterConfig.Bucket)
		if err != nil {
			exit.Error(err)
		}
		err = awsClient.TagBucket(clusterConfig.Bucket, clusterConfig.Tags)
		if err != nil {
			exit.Error(err)
		}

		err = CreateLogGroupIfNotFound(awsClient, clusterConfig.LogGroup)
		if err != nil {
			exit.Error(err)
		}
		err = awsClient.TagLogGroup(clusterConfig.LogGroup, clusterConfig.Tags)
		if err != nil {
			exit.Error(err)
		}

		// create Cloudwatch Dashboard
		err = CreateCloudWatchDashboard(awsClient, clusterConfig.ClusterName)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerUpdateCommand("/root/install.sh", clusterConfig, awsCreds, _flagClusterEnv)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := "\nDebugging tips (may or may not apply to this error):"
			helpStr += fmt.Sprintf("\n* if your cluster started spinning up but was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", *clusterConfig.Region)
			helpStr += fmt.Sprintf("\n* if your cluster started spinning up, please ensure that your CloudFormation stacks for this cluster have been fully deleted before trying to spin up this cluster again: https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=-%s-", *clusterConfig.Region, clusterConfig.ClusterName)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUp(out + helpStr))
		}

		fmt.Printf(console.Bold("\nan environment named \"%s\" has been configured for this cluster; append `--env %s` to cortex commands to connect to it (e.g. `cortex deploy --env %s`), or set it as your default with `cortex env default %s`\n"), _flagClusterEnv, _flagClusterEnv, _flagClusterEnv, _flagClusterEnv)
	},
}

var _configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "update a cluster's configuration",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.configure")

		if _flagClusterEnv == "local" {
			exit.Error(ErrorNotSupportedInLocalEnvironment())
		}

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig, _flagClusterEnv, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfig(_flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		awsClient, err := newAWSClient(*accessConfig.Region, awsCreds)
		if err != nil {
			exit.Error(err)
		}

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil {
			if errors.GetKind(err) == clusterstate.ErrUnexpectedCloudFormationStatus {
				fmt.Println(fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; please run `cortex cluster down` to delete the cluster, or delete the cloudformation stacks manually in your AWS console (%s)", *accessConfig.ClusterName, *accessConfig.Region, getCloudFormationURLWithAccessConfig(accessConfig)))
			}
			exit.Error(err)
		}

		err = assertClusterStatus(accessConfig, clusterState.Status, clusterstate.StatusCreateComplete)
		if err != nil {
			exit.Error(err)
		}

		cachedClusterConfig := refreshCachedClusterConfig(awsCreds, _flagClusterDisallowPrompt)

		clusterConfig, err := getConfigureClusterConfig(cachedClusterConfig, awsCreds, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerUpdateCommand("/root/install.sh --update", clusterConfig, awsCreds, _flagClusterEnv)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := "\nDebugging tips (may or may not apply to this error):"
			helpStr += fmt.Sprintf("\n* if your cluster was unable to provision instances, additional error information may be found in the activity history of your cluster's autoscaling groups (select each autoscaling group and click the  \"Activity History\" tab): https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:", *clusterConfig.Region)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterConfigure(out + helpStr))
		}
	},
}

var _infoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.info")
		if _flagClusterEnv == "local" {
			exit.Error(ErrorNotSupportedInLocalEnvironment())
		}

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig, _flagClusterEnv, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfig(_flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		if _flagClusterInfoDebug {
			cmdDebug(awsCreds, accessConfig)
		} else {
			cmdInfo(awsCreds, accessConfig, _flagClusterDisallowPrompt)
		}
	},
}

var _downCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.down")

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		awsCreds, err := getAWSCredentials(_flagClusterConfig, _flagClusterEnv, _flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		accessConfig, err := getClusterAccessConfig(_flagClusterDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		// Check AWS access
		awsClient, err := newAWSClient(*accessConfig.Region, awsCreds)
		if err != nil {
			exit.Error(err)
		}
		warnIfNotAdmin(awsClient)

		clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
		if err != nil {
			if errors.GetKind(err) == clusterstate.ErrUnexpectedCloudFormationStatus {
				fmt.Println(fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; please delete the cloudformation stacks manually in your AWS console (%s)", *accessConfig.ClusterName, *accessConfig.Region, getCloudFormationURLWithAccessConfig(accessConfig)))
			}
			exit.Error(err)
		}

		switch clusterState.Status {
		case clusterstate.StatusNotFound:
			exit.Error(ErrorClusterDoesNotExist(*accessConfig.ClusterName, *accessConfig.Region))
		case clusterstate.StatusDeleteComplete:
			exit.Error(ErrorClusterAlreadyDeleted(*accessConfig.ClusterName, *accessConfig.Region))
		}

		if !_flagClusterDisallowPrompt {
			prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s will be spun down and all apis will be deleted, are you sure you want to continue?", *accessConfig.ClusterName, *accessConfig.Region), "", "")
		}

		// delete Cloudwatch Dashboard
		err = awsClient.DeleteDashboard(*accessConfig.ClusterName)
		if err != nil {
			exit.Error(err)
		}

		out, exitCode, err := runManagerAccessCommand("/root/uninstall.sh", *accessConfig, awsCreds, _flagClusterEnv)
		if err != nil {
			exit.Error(err)
		}
		if exitCode == nil || *exitCode != 0 {
			helpStr := fmt.Sprintf("\nNote: if this error cannot be resolved, please ensure that all CloudFormation stacks for this cluster eventually become been fully deleted (%s). If the stack deletion process has failed, please manually delete the stack from the AWS console (this may require manually deleting particular AWS resources that are blocking the stack deletion)", getCloudFormationURLWithAccessConfig(accessConfig))
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

func cmdInfo(awsCreds AWSCredentials, accessConfig *clusterconfig.AccessConfig, disallowPrompt bool) {
	awsClient, err := newAWSClient(*accessConfig.Region, awsCreds)
	if err != nil {
		exit.Error(err)
	}

	if err := printInfoClusterState(awsClient, accessConfig); err != nil {
		exit.Error(err)
	}

	clusterConfig := refreshCachedClusterConfig(awsCreds, disallowPrompt)

	out, exitCode, err := runManagerAccessCommand("/root/info.sh", *accessConfig, awsCreds, _flagClusterEnv)
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
		if strings.HasPrefix(line, "operator endpoint: ") {
			operatorEndpoint = "https://" + strings.TrimSpace(strings.TrimPrefix(line, "operator endpoint: "))
			break
		}
	}

	if err := printInfoOperatorResponse(clusterConfig, operatorEndpoint, awsCreds); err != nil {
		exit.Error(err)
	}

	if err := updateInfoEnvironment(operatorEndpoint, awsCreds, disallowPrompt); err != nil {
		exit.Error(err)
	}
}

func printInfoClusterState(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig) error {
	clusterState, err := clusterstate.GetClusterState(awsClient, accessConfig)
	if err != nil {
		if errors.GetKind(err) == clusterstate.ErrUnexpectedCloudFormationStatus {
			fmt.Println(fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; please run `cortex cluster down` to delete the cluster, or delete the cloudformation stacks manually in your AWS console (%s)", *accessConfig.ClusterName, *accessConfig.Region, getCloudFormationURLWithAccessConfig(accessConfig)))
		}
		return err
	}

	fmt.Println(clusterState.TableString())
	if clusterState.Status == clusterstate.StatusCreateFailed || clusterState.Status == clusterstate.StatusDeleteFailed {
		fmt.Println(fmt.Sprintf("more information can be found in your AWS console %s", getCloudFormationURLWithAccessConfig(accessConfig)))
		fmt.Println()
	}

	err = assertClusterStatus(accessConfig, clusterState.Status, clusterstate.StatusCreateComplete)
	if err != nil {
		return err
	}

	return nil
}

func printInfoOperatorResponse(clusterConfig clusterconfig.Config, operatorEndpoint string, awsCreds AWSCredentials) error {
	fmt.Print("fetching cluster status ...\n\n")

	operatorConfig := cluster.OperatorConfig{
		Telemetry:          isTelemetryEnabled(),
		EnvName:            _flagClusterEnv,
		ClientID:           clientID(),
		OperatorEndpoint:   operatorEndpoint,
		AWSAccessKeyID:     awsCreds.AWSAccessKeyID,
		AWSSecretAccessKey: awsCreds.AWSSecretAccessKey,
	}

	infoResponse, err := cluster.Info(operatorConfig)
	if err != nil {
		return err
	}
	infoResponse.ClusterConfig.Config = clusterConfig

	printInfoClusterConfig(infoResponse)
	printInfoPricing(infoResponse, clusterConfig)
	printInfoNodes(infoResponse)

	return nil
}

func printInfoClusterConfig(infoResponse *schema.InfoResponse) {
	var items table.KeyValuePairs
	items.Add("aws access key id", infoResponse.MaskedAWSAccessKeyID)
	items.AddAll(infoResponse.ClusterConfig.UserTable())
	items.Print()
}

func printInfoPricing(infoResponse *schema.InfoResponse, clusterConfig clusterconfig.Config) {
	numAPIInstances := len(infoResponse.NodeInfos)

	var totalAPIInstancePrice float64
	for _, nodeInfo := range infoResponse.NodeInfos {
		totalAPIInstancePrice += nodeInfo.Price
	}

	eksPrice := aws.EKSPrices[*clusterConfig.Region]
	operatorInstancePrice := aws.InstanceMetadatas[*clusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[*clusterConfig.Region]["gp2"].PriceGB * 20 / 30 / 24
	nlbPrice := aws.NLBMetadatas[*clusterConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[*clusterConfig.Region].Price
	apiEBSPrice := aws.EBSMetadatas[*clusterConfig.Region][clusterConfig.InstanceVolumeType.String()].PriceGB * float64(clusterConfig.InstanceVolumeSize) / 30 / 24
	if clusterConfig.InstanceVolumeType.String() == "io1" && clusterConfig.InstanceVolumeIOPS != nil {
		apiEBSPrice += aws.EBSMetadatas[*clusterConfig.Region][clusterConfig.InstanceVolumeType.String()].PriceIOPS * float64(*clusterConfig.InstanceVolumeIOPS) / 30 / 24
	}

	var natTotalPrice float64
	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(clusterConfig.AvailabilityZones))
	}

	totalPrice := eksPrice + totalAPIInstancePrice + apiEBSPrice*float64(numAPIInstances) + operatorInstancePrice + operatorEBSPrice + nlbPrice*2 + natTotalPrice
	fmt.Printf(console.Bold("\nyour cluster currently costs %s per hour\n\n"), s.DollarsAndCents(totalPrice))

	headers := []table.Header{
		{Title: "aws resource"},
		{Title: "cost per hour"},
	}

	rows := [][]interface{}{}
	rows = append(rows, []interface{}{"1 eks cluster", s.DollarsMaxPrecision(eksPrice)})
	rows = append(rows, []interface{}{fmt.Sprintf("%d %s for your apis", numAPIInstances, s.PluralS("instance", numAPIInstances)), s.DollarsAndTenthsOfCents(totalAPIInstancePrice) + " total"})
	rows = append(rows, []interface{}{fmt.Sprintf("%d %dgb ebs %s for your apis", numAPIInstances, clusterConfig.InstanceVolumeSize, s.PluralS("volume", numAPIInstances)), s.DollarsAndTenthsOfCents(apiEBSPrice*float64(numAPIInstances)) + " total"})
	rows = append(rows, []interface{}{"1 t3.medium instance for the operator", s.DollarsMaxPrecision(operatorInstancePrice)})
	rows = append(rows, []interface{}{"1 20gb ebs volume for the operator", s.DollarsAndTenthsOfCents(operatorEBSPrice)})
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
	var doesClusterHaveGPUs bool
	for _, nodeInfo := range infoResponse.NodeInfos {
		totalReplicas += nodeInfo.NumReplicas
		if nodeInfo.ComputeCapacity.GPU > 0 {
			doesClusterHaveGPUs = true
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
		{Title: "CPU (free / total)"},
		{Title: "memory (free / total)"},
		{Title: "GPU (free / total)", Hidden: !doesClusterHaveGPUs},
	}

	rows := [][]interface{}{}
	for _, nodeInfo := range infoResponse.NodeInfos {
		lifecycle := "on-demand"
		if nodeInfo.IsSpot {
			lifecycle = "spot"
		}
		cpuStr := nodeInfo.ComputeAvailable.CPU.String() + " / " + nodeInfo.ComputeCapacity.CPU.String()
		memStr := nodeInfo.ComputeAvailable.Mem.String() + " / " + nodeInfo.ComputeCapacity.Mem.String()
		gpuStr := s.Int64(nodeInfo.ComputeAvailable.GPU) + " / " + s.Int64(nodeInfo.ComputeCapacity.GPU)
		rows = append(rows, []interface{}{nodeInfo.InstanceType, lifecycle, nodeInfo.NumReplicas, cpuStr, memStr, gpuStr})
	}

	t := table.Table{
		Headers: headers,
		Rows:    rows,
	}
	fmt.Println()
	t.MustPrint(&table.Opts{Sort: pointer.Bool(false)})
}

func updateInfoEnvironment(operatorEndpoint string, awsCreds AWSCredentials, disallowPrompt bool) error {
	prevEnv, err := readEnv(_flagClusterEnv)
	if err != nil {
		return err
	}

	newEnvironment := cliconfig.Environment{
		Name:               _flagClusterEnv,
		Provider:           types.AWSProviderType,
		OperatorEndpoint:   pointer.String(operatorEndpoint),
		AWSAccessKeyID:     pointer.String(awsCreds.AWSAccessKeyID),
		AWSSecretAccessKey: pointer.String(awsCreds.AWSSecretAccessKey),
	}

	shouldWriteEnv := false
	if prevEnv == nil {
		shouldWriteEnv = true
		fmt.Println()
	} else if *prevEnv.OperatorEndpoint != operatorEndpoint || *prevEnv.AWSAccessKeyID != awsCreds.AWSAccessKeyID || *prevEnv.AWSSecretAccessKey != awsCreds.AWSSecretAccessKey {
		if disallowPrompt {
			fmt.Print(fmt.Sprintf("\nfound an existing environment named \"%s\"; overwriting it to connect to this cluster\n", _flagClusterEnv))
			shouldWriteEnv = true
		} else {
			shouldWriteEnv = prompt.YesOrNo(fmt.Sprintf("\nfound an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster?", _flagClusterEnv), "", "")
		}
	}

	if shouldWriteEnv {
		err := addEnvToCLIConfig(newEnvironment)
		if err != nil {
			return err
		}

		fmt.Printf(console.Bold("configured %s environment\n"), _flagClusterEnv)
	}

	return nil
}

func cmdDebug(awsCreds AWSCredentials, accessConfig *clusterconfig.AccessConfig) {
	out, exitCode, err := runManagerAccessCommand("/root/debug.sh", *accessConfig, awsCreds, _flagClusterEnv)
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

func refreshCachedClusterConfig(awsCreds AWSCredentials, disallowPrompt bool) clusterconfig.Config {
	accessConfig, err := getClusterAccessConfig(disallowPrompt)
	if err != nil {
		exit.Error(err)
	}

	// add empty file if cached cluster doesn't exist so that the file output by manager container maintains current user permissions
	cachedConfigPath := cachedClusterConfigPath(*accessConfig.ClusterName, *accessConfig.Region)
	if !files.IsFile(cachedConfigPath) {
		files.MakeEmptyFile(cachedConfigPath)
	}

	mountedConfigPath := mountedClusterConfigPath(*accessConfig.ClusterName, *accessConfig.Region)

	fmt.Println("syncing cluster configuration ..." + "\n")
	out, exitCode, err := runManagerAccessCommand("/root/refresh.sh "+mountedConfigPath, *accessConfig, awsCreds, _flagClusterEnv)
	if err != nil {
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		exit.Error(ErrorClusterRefresh(out))
	}

	refreshedClusterConfig := &clusterconfig.Config{}
	readCachedClusterConfigFile(refreshedClusterConfig, cachedConfigPath)
	return *refreshedClusterConfig
}

func assertClusterStatus(accessConfig *clusterconfig.AccessConfig, status clusterstate.Status, allowedStatuses ...clusterstate.Status) error {
	for _, allowedStatus := range allowedStatuses {
		if status == allowedStatus {
			return nil
		}
	}

	switch status {
	case clusterstate.StatusCreateInProgress:
		return ErrorClusterUpInProgress(*accessConfig.ClusterName, *accessConfig.Region)
	case clusterstate.StatusCreateComplete:
		return ErrorClusterAlreadyCreated(*accessConfig.ClusterName, *accessConfig.Region)
	case clusterstate.StatusDeleteInProgress:
		return ErrorClusterDownInProgress(*accessConfig.ClusterName, *accessConfig.Region)
	case clusterstate.StatusNotFound:
		return ErrorClusterDoesNotExist(*accessConfig.ClusterName, *accessConfig.Region)
	case clusterstate.StatusDeleteComplete:
		return ErrorClusterAlreadyDeleted(*accessConfig.ClusterName, *accessConfig.Region)
	default:
		return ErrorFailedClusterStatus(status, *accessConfig.ClusterName, *accessConfig.Region)
	}
}

func getCloudFormationURLWithAccessConfig(accessConfig *clusterconfig.AccessConfig) string {
	return getCloudFormationURL(*accessConfig.ClusterName, *accessConfig.Region)
}

func CreateBucketIfNotFound(awsClient *aws.Client, bucket string) error {
	bucketFound, err := awsClient.DoesBucketExist(bucket)
	if err != nil {
		return err
	}
	if !bucketFound {
		fmt.Print("￮ creating a new s3 bucket: ", bucket)
		err = awsClient.CreateBucket(bucket)
		if err != nil {
			return err
		}
		fmt.Println(" ✓")
	} else {
		fmt.Println("￮ using existing s3 bucket:", bucket, "✓")
	}
	return nil
}

func CreateLogGroupIfNotFound(awsClient *aws.Client, logGroup string) error {
	logGroupFound, err := awsClient.DoesLogGroupExist(logGroup)
	if err != nil {
		return err
	}
	if !logGroupFound {
		fmt.Print("￮ creating a new cloudwatch log group: ", logGroup)
		err = awsClient.CreateLogGroup(logGroup)
		if err != nil {
			return err
		}
		fmt.Println(" ✓")
	} else {
		fmt.Println("￮ using existing cloudwatch log group:", logGroup, "✓")
	}

	return nil
}

// CreateCloudWatchDashboard creates new dashboard
// If dashboard already exists deletes old one and create a new one
func CreateCloudWatchDashboard(awsClient *aws.Client, dashboardName string) error {
	dashboardFound, err := awsClient.DoesDashboardExist(dashboardName)
	if err != nil {
		return err
	}

	if dashboardFound {
		fmt.Println("￮ updating cloudwatch dashboard: ", dashboardName)
		err = awsClient.DeleteDashboard(dashboardName)
		if err != nil {
			return err
		}
	} else {
		fmt.Print("￮ creating cloudwatch dashboard: ", dashboardName)
	}

	err = awsClient.CreateDashboard(dashboardName)
	if err != nil {
		return err
	}

	fmt.Println(" ✓")

	return nil
}
