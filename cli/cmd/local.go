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

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"path/filepath"
// 	"strconv"
// 	"strings"

// 	"github.com/cortexlabs/cortex/cli/local"
// 	"github.com/cortexlabs/cortex/pkg/lib/cast"
// 	"github.com/cortexlabs/cortex/pkg/lib/console"
// 	"github.com/cortexlabs/cortex/pkg/lib/debug"
// 	"github.com/cortexlabs/cortex/pkg/lib/errors"
// 	"github.com/cortexlabs/cortex/pkg/lib/exit"
// 	"github.com/cortexlabs/cortex/pkg/lib/files"
// 	"github.com/cortexlabs/cortex/pkg/lib/json"
// 	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
// 	"github.com/cortexlabs/cortex/pkg/lib/pointer"
// 	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
// 	s "github.com/cortexlabs/cortex/pkg/lib/strings"
// 	"github.com/cortexlabs/cortex/pkg/lib/table"
// 	"github.com/cortexlabs/cortex/pkg/operator/schema"
// 	"github.com/cortexlabs/cortex/pkg/types"
// 	"github.com/cortexlabs/cortex/pkg/types/metrics"
// 	"github.com/cortexlabs/cortex/pkg/types/spec"
// 	"github.com/cortexlabs/cortex/pkg/types/status"
// 	"github.com/cortexlabs/cortex/pkg/types/userconfig"
// 	dockertypes "github.com/docker/docker/api/types"
// 	"github.com/docker/docker/api/types/filters"
// 	"github.com/spf13/cobra"
// )

// func init() {
// 	localCmd.PersistentFlags()
// 	addEnvFlag(localCmd, types.LocalProviderType.String())
// 	_localWorkSpace = filepath.Join(_localDir, "local_workspace")
// }

// var localCmd = &cobra.Command{
// 	Use:   "local",
// 	Short: "local an application",
// 	Long:  "local an application.",
// 	Args:  cobra.ExactArgs(0),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		configPath := getConfigPath(args)
// 		deploymentMap := deploymentBytes(configPath, false)

// 	},
// }

// var localGet = &cobra.Command{
// 	Use:   "local-get",
// 	Short: "local an application",
// 	Long:  "local an application.",
// 	Args:  cobra.RangeArgs(0, 1),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		if len(args) == 1 {
// 			output, err := getLocalAPI(args[0])
// 			if err != nil {
// 				// TODO
// 			}
// 			fmt.Println(output)
// 			return
// 		}
// 		containers, err := GetAllContainers()
// 		if err != nil {
// 			exit.Error(err)
// 		}

// 		if len(containers) == 0 {
// 			fmt.Println("no apis deployed")
// 			exit.Ok()
// 		}

// 		apiSet := strset.New()
// 		for _, container := range containers {
// 			apiSet.Add(container.Labels["apiName"])
// 		}

// 		apiSpecList := []spec.API{}

// 		for apiName := range apiSet {
// 			filepaths, err := files.ListDirRecursive(filepath.Join(local.LocalWorkspace, "apis", apiName), false)
// 			if err != nil {
// 				// TODO
// 			}
// 			var apiSpec spec.API
// 			found := false
// 			for _, filepath := range filepaths {
// 				if strings.HasSuffix(filepath, "spec.msgpack") {
// 					bytes, err := files.ReadFileBytes(filepath)
// 					if err != nil {
// 						// TODO
// 					}
// 					err = msgpack.Unmarshal(bytes, &apiSpec)
// 					if err != nil {
// 						// TODO
// 					}
// 					debug.Pp(apiSpec)
// 					apiSpecList = append(apiSpecList, apiSpec)
// 					break
// 				}
// 			}
// 			if !found {
// 				// TODO
// 			}
// 		}

// 		statusList := []status.Status{}
// 		metricsList := []metrics.Metrics{}
// 		for _, apiSpec := range apiSpecList {
// 			apiStatus, err := GetAPIStatus(&apiSpec)
// 			if err != nil {
// 				// TODO
// 			}

// 			statusList = append(statusList, apiStatus)
// 			metrics, err := GetAPIMetrics(&apiSpec)
// 			if err != nil {
// 				// TODO
// 			}
// 			metricsList = append(metricsList, metrics)
// 		}

// 		table := apiTable(apiSpecList, statusList, metricsList, false)
// 		fmt.Println(table.MustFormat())
// 	},
// }

// func GetAPIMetrics(api *spec.API) (metrics.Metrics, error) {
// 	apiWorkspace := filepath.Join(local.LocalWorkspace, filepath.Dir(api.Key))

// 	networkStats := metrics.NetworkStats{}

// 	filepaths, err := files.ListDir(apiWorkspace, false)
// 	if err != nil {
// 		// TODO
// 	}

// 	totalRequestTime := 0.0
// 	for _, filepath := range filepaths {
// 		if strings.HasSuffix(filepath, "2XX") {
// 			fileContent, err := files.ReadFile(filepath)
// 			if err != nil {
// 				// TODO
// 			}

// 			count, err := strconv.ParseInt(fileContent, 10, 64)
// 			networkStats.Code2XX += int(count)
// 		}

// 		if strings.HasSuffix(filepath, "4XX") {
// 			fileContent, err := files.ReadFile(filepath)
// 			if err != nil {
// 				// TODO
// 			}

// 			count, err := strconv.ParseInt(fileContent, 10, 64)
// 			networkStats.Code4XX += int(count)
// 		}

// 		if strings.HasSuffix(filepath, "5XX") {
// 			fileContent, err := files.ReadFile(filepath)
// 			if err != nil {
// 				// TODO
// 			}

// 			count, err := strconv.ParseInt(fileContent, 10, 64)
// 			networkStats.Code5XX += int(count)
// 		}

// 		if strings.HasSuffix(filepath, "request_count") {
// 			fileContent, err := files.ReadFile(filepath)
// 			if err != nil {
// 				// TODO
// 			}

// 			requestTime, err := strconv.ParseFloat(fileContent, 64)
// 			totalRequestTime += requestTime
// 		}
// 	}

// 	totalRequests := networkStats.Code2XX + networkStats.Code4XX + networkStats.Code5XX
// 	networkStats.Total = totalRequests
// 	if totalRequests != 0 {
// 		networkStats.Latency = pointer.Float64(totalRequestTime / float64(totalRequests))
// 	}
// 	return metrics.Metrics{
// 		APIName:      api.Name,
// 		NetworkStats: &networkStats,
// 	}, nil
// }

// func GetAPIStatus(api *spec.API) (status.Status, error) {
// 	apiStatus := status.Status{
// 		APIID:   api.ID,
// 		APIName: api.Name,
// 		ReplicaCounts: status.ReplicaCounts{
// 			Requested: 1,
// 		},
// 	}

// 	containers, err := local.GetContainerByAPI(api.Name)
// 	if err != nil {
// 		return status.Status{}, err
// 	}

// 	if api.Predictor.Type == userconfig.TensorFlowPredictorType && len(containers) != 2 {
// 		apiStatus.ReplicaCounts.Updated.Failed = 1
// 		apiStatus.Code = status.Error
// 		return apiStatus, nil
// 	}

// 	if !files.IsFile(filepath.Join(local.LocalWorkspace, "apis", api.Name, api.ID, "api_readiness.txt")) {
// 		apiStatus.ReplicaCounts.Updated.Initializing = 1
// 		apiStatus.Code = status.Updating
// 		return apiStatus, nil
// 	}

// 	for _, container := range containers {
// 		if container.State != "running" {
// 			apiStatus.ReplicaCounts.Updated.Failed = 1
// 			apiStatus.Code = status.Error
// 			return apiStatus, nil
// 		}
// 	}

// 	apiStatus.ReplicaCounts.Updated.Ready = 1
// 	apiStatus.Code = status.Live
// 	return apiStatus, nil
// }

// func GetAllContainers() ([]dockertypes.Container, error) {
// 	docker, err := getDockerClient()
// 	if err != nil {
// 		return nil, err
// 	}

// 	dargs := filters.NewArgs()
// 	dargs.Add("label", "cortex=true")
// 	containers, err := docker.ContainerList(context.Background(), dockertypes.ContainerListOptions{
// 		All:     true,
// 		Filters: dargs,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return containers, nil
// }

// var localDelete = &cobra.Command{
// 	Use:   "local-delete",
// 	Short: "local an application",
// 	Long:  "local an application.",
// 	Args:  cobra.ExactArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		apiName := args[0]
// 		err := local.DeleteContainers(apiName)
// 		if err != nil {
// 			fmt.Println(err.Error())
// 		}
// 		files.DeleteDirIfPresent(filepath.Join(local.LocalWorkspace, "apis", apiName))
// 	},
// }

// var localLogs = &cobra.Command{
// 	Use:   "local-logs",
// 	Short: "local an application",
// 	Long:  "local an application.",
// 	Args:  cobra.ExactArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {

// 		containers, err := local.GetContainerByAPI(args[0])
// 		if err != nil {
// 			// TODO
// 		}

// 		containerIDs := []string{}
// 		for _, container := range containers {
// 			containerIDs = append(containerIDs, container.ID)
// 		}

// 		streamDockerLogs(containerIDs[0], containerIDs[1:]...)
// 	},
// }

// func getLocalAPI(apiName string) (string, error) {
// 	apiWorkspace := filepath.Join(local.LocalWorkspace, "apis", apiName)
// 	if !files.IsDir(apiWorkspace) {
// 		// TODO
// 	}
// 	filepaths, err := files.ListDirRecursive(apiWorkspace, false)
// 	if err != nil {
// 		// TODO
// 	}

// 	var apiSpec spec.API
// 	found := false
// 	for _, filepath := range filepaths {
// 		if strings.HasSuffix(filepath, "spec.msgpack") {
// 			bytes, err := files.ReadFileBytes(filepath)
// 			if err != nil {
// 				// TODO
// 			}
// 			err = msgpack.Unmarshal(bytes, &apiSpec)
// 			if err != nil {
// 				// TODO
// 			}
// 			break
// 		}
// 	}
// 	if !found {
// 		// TODO
// 	}

// 	apiStatus, err := GetAPIStatus(&apiSpec)
// 	if err != nil {
// 		// TODO
// 	}

// 	apiMetrics, err := GetAPIMetrics(&apiSpec)
// 	if err != nil {
// 		// TODO
// 	}

// 	t := apiTable([]spec.API{apiSpec}, []status.Status{apiStatus}, []metrics.Metrics{apiMetrics}, false)
// 	out := t.MustFormat()

// 	containers, err := local.GetContainerByAPI(apiName)
// 	if err != nil {
// 		// TODO
// 	}

// 	if len(containers) == 0 {
// 		// TODO
// 	}
// 	apiContainer := containers[0]
// 	if len(containers) == 2 && apiContainer.Labels["type"] != "api" {
// 		apiContainer = containers[1]
// 	}

// 	apiPort := ""
// 	for _, port := range apiContainer.Ports {
// 		if port.PrivatePort == 8888 {
// 			apiPort = s.Uint16(port.PublicPort)
// 		}
// 	}

// 	apiEndpoint := "localhost:" + apiPort + "/predict"
// 	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

// 	out += fmt.Sprintf("\n%s curl %s?debug=true -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), apiEndpoint)

// 	if apiSpec.Predictor.Type == userconfig.TensorFlowPredictorType || apiSpec.Predictor.Type == userconfig.ONNXPredictorType {
// 		out += "\n" + describeLocalModelInput(&apiStatus, apiEndpoint)
// 	}

// 	out += titleStr("configuration") + strings.TrimSpace(apiSpec.UserStr())

// 	return out, nil
// }

// func describeLocalModelInput(status *status.Status, apiEndpoint string) string {
// 	if status.Updated.Ready+status.Stale.Ready == 0 {
// 		return "the model's input schema will be available when the api is live\n"
// 	}

// 	apiSummary, err := getLocalAPISummary(apiEndpoint)
// 	if err != nil {
// 		return "error retrieving the model's input schema: " + errors.Message(err) + "\n"
// 	}

// 	rows := make([][]interface{}, len(apiSummary.ModelSignature))
// 	rowNum := 0
// 	for inputName, featureSignature := range apiSummary.ModelSignature {
// 		shapeStr := make([]string, len(featureSignature.Shape))
// 		for idx, dim := range featureSignature.Shape {
// 			shapeStr[idx] = s.ObjFlatNoQuotes(dim)
// 		}
// 		rows[rowNum] = []interface{}{
// 			inputName,
// 			featureSignature.Type,
// 			"(" + strings.Join(shapeStr, ", ") + ")",
// 		}
// 		rowNum++
// 	}

// 	t := table.Table{
// 		Headers: []table.Header{
// 			{Title: "model input", MaxWidth: 32},
// 			{Title: "type", MaxWidth: 10},
// 			{Title: "shape", MaxWidth: 20},
// 		},
// 		Rows: rows,
// 	}

// 	return t.MustFormat()
// }

// func getLocalAPISummary(apiEndpoint string) (*schema.APISummary, error) {
// 	req, err := http.NewRequest("GET", "http://"+apiEndpoint, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to request api summary")
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	response, err := _apiClient.MakeRequest(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var apiSummary schema.APISummary
// 	err = json.DecodeWithNumber(response, &apiSummary)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to parse api summary response")
// 	}

// 	for _, featureSignature := range apiSummary.ModelSignature {
// 		featureSignature.Shape = cast.JSONNumbers(featureSignature.Shape)
// 	}

// 	return &apiSummary, nil
// }
