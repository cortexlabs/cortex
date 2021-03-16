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
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/yaml"
	"github.com/spf13/cobra"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

var (
	_flagClusterGCPUpEnv           string
	_flagClusterGCPInfoEnv         string
	_flagClusterGCPInfoDebug       bool
	_flagClusterGCPConfig          string
	_flagClusterGCPName            string
	_flagClusterGCPZone            string
	_flagClusterGCPProject         string
	_flagClusterGCPDisallowPrompt  bool
	_flagClusterGCPDownKeepVolumes bool
)

func clusterGCPInit() {
	_clusterGCPUpCmd.Flags().SortFlags = false
	_clusterGCPUpCmd.Flags().StringVarP(&_flagClusterGCPUpEnv, "configure-env", "e", "gcp", "name of environment to configure")
	addClusterGCPDisallowPromptFlag(_clusterGCPUpCmd)
	_clusterGCPCmd.AddCommand(_clusterGCPUpCmd)

	_clusterGCPInfoCmd.Flags().SortFlags = false
	addClusterGCPConfigFlag(_clusterGCPInfoCmd)
	addClusterGCPNameFlag(_clusterGCPInfoCmd)
	addClusterGCPProjectFlag(_clusterGCPInfoCmd)
	addClusterGCPZoneFlag(_clusterGCPInfoCmd)
	_clusterGCPInfoCmd.Flags().StringVarP(&_flagClusterGCPInfoEnv, "configure-env", "e", "", "name of environment to configure")
	_clusterGCPInfoCmd.Flags().BoolVarP(&_flagClusterGCPInfoDebug, "debug", "d", false, "save the current cluster state to a file")
	_clusterGCPInfoCmd.Flags().BoolVarP(&_flagClusterGCPDisallowPrompt, "yes", "y", false, "skip prompts")
	_clusterGCPCmd.AddCommand(_clusterGCPInfoCmd)

	_clusterGCPDownCmd.Flags().SortFlags = false
	addClusterGCPConfigFlag(_clusterGCPDownCmd)
	addClusterGCPNameFlag(_clusterGCPDownCmd)
	addClusterGCPProjectFlag(_clusterGCPDownCmd)
	addClusterGCPZoneFlag(_clusterGCPDownCmd)
	addClusterGCPDisallowPromptFlag(_clusterGCPDownCmd)
	_clusterGCPDownCmd.Flags().BoolVar(&_flagClusterGCPDownKeepVolumes, "keep-volumes", false, "keep cortex provisioned persistent volumes")
	_clusterGCPCmd.AddCommand(_clusterGCPDownCmd)
}

func addClusterGCPConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterGCPConfig, "config", "c", "", "path to a cluster configuration file")
	cmd.Flags().SetAnnotation("config", cobra.BashCompFilenameExt, _configFileExts)
}

func addClusterGCPNameFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterGCPName, "name", "n", "", "name of the cluster")
}

func addClusterGCPZoneFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterGCPZone, "zone", "z", "", "gcp zone of the cluster")
}

func addClusterGCPProjectFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&_flagClusterGCPProject, "project", "p", "", "gcp project id")
}

func addClusterGCPDisallowPromptFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&_flagClusterGCPDisallowPrompt, "yes", "y", false, "skip prompts")
}

var _clusterGCPCmd = &cobra.Command{
	Use:   "cluster-gcp",
	Short: "manage GCP clusters (contains subcommands)",
}

var _clusterGCPUpCmd = &cobra.Command{
	Use:   "up [CLUSTER_CONFIG_FILE]",
	Short: "spin up a cluster on gcp",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.EventNotify("cli.cluster.up", map[string]interface{}{"provider": types.GCPProviderType})

		clusterConfigFile := args[0]

		envExists, err := isEnvConfigured(_flagClusterGCPUpEnv)
		if err != nil {
			exit.Error(err)
		}
		if envExists {
			if _flagClusterGCPDisallowPrompt {
				fmt.Printf("found an existing environment named \"%s\", which will be overwritten to connect to this cluster once it's created\n\n", _flagClusterGCPUpEnv)
			} else {
				prompt.YesOrExit(fmt.Sprintf("found an existing environment named \"%s\"; would you like to overwrite it to connect to this cluster once it's created?", _flagClusterGCPUpEnv), "", "you can specify a different environment name to be configured to connect to this cluster by specifying the --configure-env flag (e.g. `cortex cluster up --configure-env prod`); or you can list your environments with `cortex env list` and delete an environment with `cortex env delete ENV_NAME`")
			}
		}

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getNewGCPClusterAccessConfig(clusterConfigFile)
		if err != nil {
			exit.Error(err)
		}

		gcpClient, err := gcp.NewFromEnvCheckProjectID(accessConfig.Project)
		if err != nil {
			exit.Error(err)
		}

		clusterConfig, err := getGCPInstallClusterConfig(gcpClient, clusterConfigFile, _flagClusterGCPDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		fullyQualifiedClusterName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", clusterConfig.Project, clusterConfig.Zone, clusterConfig.ClusterName)

		clusterExists, err := gcpClient.ClusterExists(fullyQualifiedClusterName)
		if err != nil {
			exit.Error(err)
		}
		if clusterExists {
			exit.Error(ErrorGCPClusterAlreadyExists(clusterConfig.ClusterName, clusterConfig.Zone, clusterConfig.Project))
		}

		err = createGSBucketIfNotFound(gcpClient, clusterConfig.Bucket, gcp.ZoneToRegion(accessConfig.Zone))
		if err != nil {
			exit.Error(err)
		}

		err = createGKECluster(clusterConfig, gcpClient)
		if err != nil {
			exit.Error(err)
		}

		_, _, err = runGCPManagerWithClusterConfig("/root/install.sh", clusterConfig, nil, nil)
		if err != nil {
			exit.Error(err)
		}

		operatorLoadBalancerIP, err := getGCPOperatorLoadBalancerIP(fullyQualifiedClusterName, gcpClient)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterGCPUpEnv)))
		}

		newEnvironment := cliconfig.Environment{
			Name:             _flagClusterGCPUpEnv,
			Provider:         types.GCPProviderType,
			OperatorEndpoint: "https://" + operatorLoadBalancerIP,
		}

		err = addEnvToCLIConfig(newEnvironment, true)
		if err != nil {
			exit.Error(errors.Append(err, fmt.Sprintf("\n\nyou can attempt to resolve this issue and configure your cli environment by running `cortex cluster info --configure-env %s`", _flagClusterGCPUpEnv)))
		}

		if envExists {
			fmt.Printf(console.Bold("\nthe environment named \"%s\" has been updated to point to this cluster (and was set as the default environment)\n"), _flagClusterGCPUpEnv)
		} else {
			fmt.Printf(console.Bold("\nan environment named \"%s\" has been configured to point to this cluster (and was set as the default environment)\n"), _flagClusterGCPUpEnv)
		}
	},
}

var _clusterGCPInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.info", map[string]interface{}{"provider": types.GCPProviderType})

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getGCPClusterAccessConfigWithCache(_flagClusterGCPDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		// need to ensure that the google creds are configured for the manager
		_, err = gcp.NewFromEnvCheckProjectID(accessConfig.Project)
		if err != nil {
			exit.Error(err)
		}

		if _flagClusterGCPInfoDebug {
			cmdDebugGCP(accessConfig)
		} else {
			cmdInfoGCP(accessConfig, _flagClusterGCPDisallowPrompt)
		}
	},
}

var _clusterGCPDownCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.cluster.down", map[string]interface{}{"provider": types.GCPProviderType})

		if _, err := docker.GetDockerClient(); err != nil {
			exit.Error(err)
		}

		accessConfig, err := getGCPClusterAccessConfigWithCache(_flagClusterGCPDisallowPrompt)
		if err != nil {
			exit.Error(err)
		}

		gcpClient, err := gcp.NewFromEnvCheckProjectID(accessConfig.Project)
		if err != nil {
			exit.Error(err)
		}

		fullyQualifiedClusterName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", accessConfig.Project, accessConfig.Zone, accessConfig.ClusterName)
		bucketName := clusterconfig.GCPBucketName(accessConfig.ClusterName, accessConfig.Project, accessConfig.Zone)

		clusterExists, err := gcpClient.ClusterExists(fullyQualifiedClusterName)
		if err != nil {
			exit.Error(err)
		}
		if !clusterExists {
			gcpClient.DeleteBucket(bucketName) // silently try to delete the bucket in case it got left behind
			exit.Error(ErrorGCPClusterDoesntExist(accessConfig.ClusterName, accessConfig.Zone, accessConfig.Project))
		}

		// updating CLI env is best-effort, so ignore errors
		operatorLoadBalancerIP, _ := getGCPOperatorLoadBalancerIP(fullyQualifiedClusterName, gcpClient)

		if _flagClusterGCPDisallowPrompt {
			fmt.Printf("your cluster named \"%s\" in %s (zone: %s) will be spun down and all apis will be deleted\n\n", accessConfig.ClusterName, accessConfig.Project, accessConfig.Zone)
		} else {
			prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s (zone: %s) will be spun down and all apis will be deleted, are you sure you want to continue?", accessConfig.ClusterName, accessConfig.Project, accessConfig.Zone), "", "")
		}

		fmt.Print("￮ spinning down the cluster ")

		uninstallCmd := "/root/uninstall.sh"
		if _flagClusterGCPDownKeepVolumes {
			uninstallCmd += " --keep-volumes"
		}
		output, exitCode, err := runGCPManagerAccessCommand(uninstallCmd, *accessConfig, nil, nil)
		if (exitCode != nil && *exitCode != 0) || err != nil {
			if len(output) == 0 {
				fmt.Printf("\n")
			}
			fmt.Print("\n")

			gkePvcDiskPrefix := fmt.Sprintf("gke-%s", accessConfig.ClusterName)
			if err != nil {
				fmt.Print(fmt.Sprintf("￮ failed to delete persistent disks from storage, please visit https://console.cloud.google.com/compute/disks?project=%s to manually delete the disks starting with the %s prefix: %s", accessConfig.Project, gkePvcDiskPrefix, err.Error()))
				telemetry.Error(ErrorClusterDown(err.Error()))
			} else {
				fmt.Print(fmt.Sprintf("￮ failed to delete persistent disks from storage, please visit https://console.cloud.google.com/compute/disks?project=%s to manually delete the disks starting with the %s prefix", accessConfig.Project, gkePvcDiskPrefix))
				telemetry.Error(ErrorClusterDown(output))
			}

			fmt.Print("\n\n")
			fmt.Print("￮ proceeding with best-effort deletion of the cluster ")
		}

		_, err = gcpClient.DeleteCluster(fullyQualifiedClusterName)
		if err != nil {
			fmt.Print("\n\n")
			exit.Error(err)
		}

		fmt.Println("✓")

		// best-effort deletion of cli environment(s)
		if operatorLoadBalancerIP != "" {
			envNames, isDefaultEnv, _ := getEnvNamesByOperatorEndpoint(operatorLoadBalancerIP)
			if len(envNames) > 0 {
				for _, envName := range envNames {
					err := removeEnvFromCLIConfig(envName)
					if err != nil {
						exit.Error(err)
					}
				}
				fmt.Printf("✓ deleted the %s environment configuration%s\n", s.StrsAnd(envNames), s.SIfPlural(len(envNames)))
				if isDefaultEnv {
					newDefaultEnv, err := getDefaultEnv()
					if err != nil {
						exit.Error(err)
					}
					if newDefaultEnv != nil {
						fmt.Println(fmt.Sprintf("✓ set the default environment to %s", *newDefaultEnv))
					}
				}
			}
		}

		cachedClusterConfigPath := cachedGCPClusterConfigPath(accessConfig.ClusterName, accessConfig.Project, accessConfig.Zone)
		os.Remove(cachedClusterConfigPath)
	},
}

func cmdInfoGCP(accessConfig *clusterconfig.GCPAccessConfig, disallowPrompt bool) {
	fmt.Print("fetching cluster endpoints ...\n\n")
	out, exitCode, err := runGCPManagerAccessCommand("/root/info_gcp.sh", *accessConfig, nil, nil)
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

	if err := printInfoOperatorResponseGCP(accessConfig, operatorEndpoint); err != nil {
		exit.Error(err)
	}

	if _flagClusterGCPInfoEnv != "" {
		if err := updateGCPCLIEnv(_flagClusterGCPInfoEnv, operatorEndpoint, disallowPrompt); err != nil {
			exit.Error(err)
		}
	}
}

func printInfoOperatorResponseGCP(accessConfig *clusterconfig.GCPAccessConfig, operatorEndpoint string) error {
	fmt.Print("fetching cluster status ...\n\n")

	operatorConfig := cluster.OperatorConfig{
		Telemetry:        isTelemetryEnabled(),
		ClientID:         clientID(),
		OperatorEndpoint: operatorEndpoint,
		Provider:         types.GCPProviderType,
	}

	infoResponse, err := cluster.InfoGCP(operatorConfig)
	if err != nil {
		return err
	}

	yamlBytes, err := yaml.Marshal(infoResponse.ClusterConfig.GCPConfig)
	if err != nil {
		return err
	}
	yamlString := string(yamlBytes)

	fmt.Println(console.Bold("metadata:"))
	fmt.Println(fmt.Sprintf("%s: %s", clusterconfig.APIVersionUserKey, infoResponse.ClusterConfig.APIVersion))

	fmt.Println()
	fmt.Println(console.Bold("cluster config:"))
	fmt.Print(yamlString)

	return nil
}

func cmdDebugGCP(accessConfig *clusterconfig.GCPAccessConfig) {
	// note: if modifying this string, also change it in files.IgnoreCortexDebug()
	debugFileName := fmt.Sprintf("cortex-debug-%s.tgz", time.Now().UTC().Format("2006-01-02-15-04-05"))

	containerDebugPath := "/out/" + debugFileName
	copyFromPaths := []dockerCopyFromPath{
		{
			containerPath: containerDebugPath,
			localDir:      _cwd,
		},
	}

	out, exitCode, err := runGCPManagerAccessCommand("/root/debug_gcp.sh "+containerDebugPath, *accessConfig, nil, copyFromPaths)
	if err != nil {
		exit.Error(err)
	}
	if exitCode == nil || *exitCode != 0 {
		exit.Error(ErrorClusterDebug(out))
	}

	fmt.Println("saved cluster info to ./" + debugFileName)
	return
}

func updateGCPCLIEnv(envName string, operatorEndpoint string, disallowPrompt bool) error {
	prevEnv, err := readEnv(envName)
	if err != nil {
		return err
	}

	newEnvironment := cliconfig.Environment{
		Name:             envName,
		Provider:         types.GCPProviderType,
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

func createGKECluster(clusterConfig *clusterconfig.GCPConfig, gcpClient *gcp.Client) error {
	fmt.Print("￮ creating GKE cluster ")

	gkeClusterParent := fmt.Sprintf("projects/%s/locations/%s", clusterConfig.Project, clusterConfig.Zone)
	fullyQualifiedClusterName := fmt.Sprintf("%s/clusters/%s", gkeClusterParent, clusterConfig.ClusterName)

	gkeClusterConfig := containerpb.Cluster{
		Name:                  clusterConfig.ClusterName,
		InitialClusterVersion: "1.18",
		LoggingService:        "none",
		NodePools: []*containerpb.NodePool{
			{
				Name: "ng-cortex-operator",
				Config: &containerpb.NodeConfig{
					MachineType: "n1-standard-2",
					OauthScopes: []string{
						"https://www.googleapis.com/auth/compute",
						"https://www.googleapis.com/auth/devstorage.read_only",
					},
					ServiceAccount: gcpClient.ClientEmail,
				},
				InitialNodeCount: 2,
			},
		},
		Locations: []string{clusterConfig.Zone},
	}

	for _, nodePool := range clusterConfig.NodePools {
		nodeLabels := map[string]string{"workload": "true"}
		initialNodeCount := int64(1)
		if nodePool.MinInstances > 0 {
			initialNodeCount = nodePool.MinInstances
		}

		var accelerators []*containerpb.AcceleratorConfig
		if nodePool.AcceleratorType != nil {
			accelerators = append(accelerators, &containerpb.AcceleratorConfig{
				AcceleratorCount: *nodePool.AcceleratorsPerInstance,
				AcceleratorType:  *nodePool.AcceleratorType,
			})
			nodeLabels["nvidia.com/gpu"] = "present"
		}

		if nodePool.Preemptible {
			gkeClusterConfig.NodePools = append(gkeClusterConfig.NodePools, &containerpb.NodePool{
				Name: "cx-ws-" + nodePool.Name,
				Config: &containerpb.NodeConfig{
					MachineType: nodePool.InstanceType,
					Labels:      nodeLabels,
					Taints: []*containerpb.NodeTaint{
						{
							Key:    "workload",
							Value:  "true",
							Effect: containerpb.NodeTaint_NO_SCHEDULE,
						},
					},
					Accelerators: accelerators,
					OauthScopes: []string{
						"https://www.googleapis.com/auth/compute",
						"https://www.googleapis.com/auth/devstorage.read_only",
					},
					ServiceAccount: gcpClient.ClientEmail,
					Preemptible:    true,
				},
				InitialNodeCount: int32(initialNodeCount),
			})
		} else {
			gkeClusterConfig.NodePools = append(gkeClusterConfig.NodePools, &containerpb.NodePool{
				Name: "cx-wd-" + nodePool.Name,
				Config: &containerpb.NodeConfig{
					MachineType: nodePool.InstanceType,
					Labels:      nodeLabels,
					Taints: []*containerpb.NodeTaint{
						{
							Key:    "workload",
							Value:  "true",
							Effect: containerpb.NodeTaint_NO_SCHEDULE,
						},
					},
					Accelerators: accelerators,
					OauthScopes: []string{
						"https://www.googleapis.com/auth/compute",
						"https://www.googleapis.com/auth/devstorage.read_only",
					},
					ServiceAccount: gcpClient.ClientEmail,
				},
				InitialNodeCount: int32(initialNodeCount),
			})
		}
	}

	if clusterConfig.Network != nil {
		gkeClusterConfig.Network = *clusterConfig.Network
	}
	if clusterConfig.Subnet != nil {
		gkeClusterConfig.Subnetwork = *clusterConfig.Subnet
	}

	_, err := gcpClient.CreateCluster(&containerpb.CreateClusterRequest{
		Parent:  gkeClusterParent,
		Cluster: &gkeClusterConfig,
	})
	if err != nil {
		fmt.Print("\n\n")
		if strings.Contains(errors.Message(err), "has no network named \"default\"") {
			err = errors.Append(err, "\n\nyou can specify a different network be setting the `network` field in your cluster configuration file (see https://docs.cortex.dev)")
		}
		return err
	}

	for {
		fmt.Print(".")
		time.Sleep(5 * time.Second)

		cluster, err := gcpClient.GetCluster(fullyQualifiedClusterName)
		if err != nil {
			return err
		}

		if cluster.Status == containerpb.Cluster_ERROR {
			fmt.Println(" ✗")
			helpStr := fmt.Sprintf("\nyour cluster couldn't be spun up; here is the error that was encountered: %s", cluster.StatusMessage)
			helpStr += fmt.Sprintf("\nadditional error information may be found on the cluster's page in the GCP console: https://console.cloud.google.com/kubernetes/clusters/details/%s/%s?project=%s", clusterConfig.Zone, clusterConfig.ClusterName, clusterConfig.Project)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUp(cluster.StatusMessage))
		}

		if cluster.Status != containerpb.Cluster_PROVISIONING {
			fmt.Println(" ✓")
			break
		}
	}

	return nil
}

func getGCPOperatorLoadBalancerIP(fullyQualifiedClusterName string, gcpClient *gcp.Client) (string, error) {
	cluster, err := gcpClient.GetCluster(fullyQualifiedClusterName)
	if err != nil {
		return "", err
	}
	restConfig, err := gcpClient.CreateK8SConfigFromCluster(cluster)
	if err != nil {
		return "", err
	}
	k8sIstio, err := k8s.New("istio-system", false, restConfig)
	if err != nil {
		return "", err
	}
	service, err := k8sIstio.GetService("ingressgateway-operator")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorNoOperatorLoadBalancer()
	}

	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", errors.ErrorUnexpected("unable to determine operator's endpoint")
	}

	if service.Status.LoadBalancer.Ingress[0].IP == "" {
		return "", errors.ErrorUnexpected("operator's endpoint is missing")
	}

	return service.Status.LoadBalancer.Ingress[0].IP, nil
}

func createGSBucketIfNotFound(gcpClient *gcp.Client, bucket string, location string) error {
	bucketFound, err := gcpClient.DoesBucketExist(bucket)
	if err != nil {
		return err
	}
	if !bucketFound {
		fmt.Print("￮ creating a new gs bucket: ", bucket)
		err = gcpClient.CreateBucket(bucket, location, false)
		if err != nil {
			fmt.Print("\n\n")
			return err
		}
	} else {
		fmt.Print("￮ using existing gs bucket: ", bucket)
	}
	fmt.Println(" ✓")
	return nil
}
