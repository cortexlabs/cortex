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

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

var flagEnv string
var flagAppName string

var configFileExts []string = []string{"json", "yaml", "yml"}

func init() {
	cobra.EnablePrefixMatching = true
}

var rootCmd = &cobra.Command{
	Use:     "cortex",
	Aliases: []string{"cx"},
	Short:   "machine learning infrastructure for developers",
	Long:    "Machine learning infrastructure for developers",
	Version: consts.CortexVersion,
}

func Execute() {
	defer errors.RecoverAndExit()
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(deleteCmd)

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)

	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(completionCmd)

	rootCmd.Execute()
}

func setConfigFlag(flagStr string, description string, flag *string, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(flag, flagStr+"-config", "", "", description)
	cmd.PersistentFlags().SetAnnotation(flagStr+"-config", cobra.BashCompFilenameExt, configFileExts)
}

func addEnvFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "dev", "environment")
}

func addAppNameFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagAppName, "app", "a", "", "app name")
}

var resourceTypesHelp = fmt.Sprintf("\nResource Types:\n  %s\n", strings.Join(resource.VisibleTypes.StringList(), "\n  "))

func addResourceTypesToHelp(cmd *cobra.Command) {
	usage := cmd.UsageTemplate()
	usage = strings.Replace(usage, "\nFlags:\n", resourceTypesHelp+"\nFlags:\n", 1)
	cmd.SetUsageTemplate(usage)
}
