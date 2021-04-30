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
	"path"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

func prepareDebugInit() {
	_prepareDebugCmd.Flags().SortFlags = false
}

var _prepareDebugCmd = &cobra.Command{
	Use:   "prepare-debug CONFIG_FILE [API_NAME]",
	Short: "prepare artifacts to debug containers",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.prepare-debug")

		configPath := args[0]

		var apiName string
		if len(args) == 2 {
			apiName = args[1]
		}

		configBytes, err := files.ReadFileBytes(configPath)
		if err != nil {
			exit.Error(err)
		}

		projectRoot := files.Dir(files.UserRelToAbsPath(configPath))

		apis, err := spec.ExtractAPIConfigs(configBytes, args[0])
		if err != nil {
			exit.Error(err)
		}

		if apiName == "" && len(apis) > 1 {
			exit.Error(errors.Wrap(ErrorAPINameMustBeProvided(), configPath))
		}

		var apiToPrepare userconfig.API
		if apiName == "" {
			apiToPrepare = apis[0]
		} else {
			found := false
			for i := range apis {
				api := apis[i]
				if api.Name == apiName {
					found = true
					apiToPrepare = api
					break
				}
			}
			if !found {
				exit.Error(errors.Wrap(ErrorAPINotFoundInConfig(apiName), configPath))
			}
		}

		if apiToPrepare.Kind != userconfig.RealtimeAPIKind {
			exit.Error(ErrorNotSupportedForKindAndType(apiToPrepare.Kind, userconfig.UnknownHandlerType))
		}
		if apiToPrepare.Handler.Type != userconfig.PythonHandlerType {
			exit.Error(ErrorNotSupportedForKindAndType(apiToPrepare.Kind, apiToPrepare.Handler.Type))
		}

		apiSpec := spec.API{
			API: &apiToPrepare,
		}

		debugFileName := apiSpec.Name + ".debug.json"
		jsonStr, err := libjson.Pretty(apiSpec)
		if err != nil {
			exit.Error(err)
		}
		files.WriteFile([]byte(jsonStr), path.Join(projectRoot, debugFileName))

		fmt.Println(fmt.Sprintf(
			`
docker run --rm -p 9000:8888 \
-e "CORTEX_VERSION=%s" \
-e "CORTEX_API_SPEC=/mnt/project/%s" \
-v %s:/mnt/project \
%s`, consts.CortexVersion, debugFileName, path.Clean(projectRoot), apiToPrepare.Handler.Image))
	},
}
