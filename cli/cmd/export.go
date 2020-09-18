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
	"path"
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

var _flagExportEnv string

func exportInit() {
	_exportCmd.Flags().SortFlags = false
	_exportCmd.Flags().StringVarP(&_flagExportEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
}

var _exportCmd = &cobra.Command{
	Use:   "export API_NAME",
	Short: "export an API",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env, err := ReadOrConfigureEnv(_flagExportEnv)
		if err != nil {
			telemetry.Event("cli.export")
			exit.Error(err)
		}
		telemetry.Event("cli.export", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(_flagLogsEnv, cmd)
		if err != nil {
			exit.Error(err)
		}

		if env.Provider == types.LocalProviderType {
			print.BoldFirstLine("`cortex export` is not supported in the local environment")
			return
		}

		info, err := cluster.Info(MustGetOperatorConfig(env.Name))
		if err != nil {
			exit.Error(err)
		}

		apiRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), args[0])
		if err != nil {
			// note: if modifying this string, search the codebase for it and change all occurrences
			if strings.HasSuffix(errors.Message(err), "is not deployed") {
				fmt.Println(console.Bold(errors.Message(err)))
				return
			}
			exit.Error(err)
		}

		var apiSpec *spec.API

		if apiRes.RealtimeAPI != nil {
			apiSpec = &apiRes.RealtimeAPI.Spec
		} else if apiRes.BatchAPI != nil {
			apiSpec = &apiRes.BatchAPI.Spec
		} else if apiRes.TrafficSplitter != nil {
			apiSpec = &apiRes.TrafficSplitter.Spec
		}

		baseDir := apiSpec.Name + "-" + apiSpec.ID

		fmt.Println("exporting api to", baseDir)

		err = os.Mkdir(baseDir, os.ModePerm)
		if err != nil {
			exit.Error(err)
		}

		var awsClient *aws.Client
		if env.AWSAccessKeyID != nil {
			awsClient, err = aws.NewFromCreds(*info.ClusterConfig.Region, *env.AWSAccessKeyID, *env.AWSSecretAccessKey)
			if err != nil {
				exit.Error(err)
			}
		} else {
			awsClient, err = aws.NewAnonymousClient()
			if err != nil {
				exit.Error(err)
			}
		}

		err = awsClient.DownloadFileFromS3(info.ClusterConfig.Bucket, apiSpec.RawAPIKey(), path.Join(baseDir, apiSpec.FileName))
		if err != nil {
			exit.Error(err)
		}

		if apiSpec.Kind != userconfig.TrafficSplitterKind {
			zipFileLocation := path.Join(baseDir, path.Base(apiSpec.ProjectKey))
			err = awsClient.DownloadFileFromS3(info.ClusterConfig.Bucket, apiSpec.ProjectKey, path.Join(baseDir, path.Base(apiSpec.ProjectKey)))
			if err != nil {
				exit.Error(err)
			}

			_, err = zip.UnzipFileToDir(zipFileLocation, baseDir)
			if err != nil {
				exit.Error(err)
			}

			err := os.Remove(zipFileLocation)
			if err != nil {
				exit.Error(err)
			}
		}
	},
}
