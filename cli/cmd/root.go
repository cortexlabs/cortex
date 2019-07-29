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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

var cmdStr string

var flagEnv string
var flagWatch bool
var flagAppName string
var flagVerbose bool
var flagSummary bool
var flagActiveDeployments bool

var configFileExts = []string{"yaml", "yml"}

func init() {
	cobra.EnablePrefixMatching = true

	cmdStr = "cortex"
	for _, arg := range os.Args[1:] {
		if arg == "-w" || arg == "--watch" {
			continue
		}
		cmdStr += " " + arg
	}
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

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(deleteCmd)

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

func addWatchFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&flagWatch, "watch", "w", false, "re-run the command every 2 seconds")
}

func addAppNameFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagAppName, "deployment", "d", "", "deployment name")
}

func addVerboseFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "show verbose output")
}

func addSummaryFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&flagSummary, "summary", "s", false, "show summarized output")
}

func addActiveDeploymentsFlag(cmd *cobra.Command) {
	getCmd.PersistentFlags().BoolVarP(&flagActiveDeployments, "active-deployments", "a", false, "list active Cortex deployments")
}

var resourceTypesHelp = fmt.Sprintf("\nResource Types:\n  %s\n", strings.Join(resource.VisibleTypes.StringList(), "\n  "))

func addResourceTypesToHelp(cmd *cobra.Command) {
	usage := cmd.UsageTemplate()
	usage = strings.Replace(usage, "\nFlags:\n", resourceTypesHelp+"\nFlags:\n", 1)
	cmd.SetUsageTemplate(usage)
}

func getTerminalWidth() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	dimensions := strings.Split(strings.TrimSpace(string(out)), " ")
	if len(dimensions) != 2 {
		return 0
	}
	widthStr := dimensions[1]
	width, ok := s.ParseInt(widthStr)
	if !ok {
		return 0
	}
	return width
}

func watchHeader() string {
	timeStr := libtime.LocalHourNow()
	width := getTerminalWidth()
	numExtraChars := 4
	padding := strings.Repeat(" ", slices.MaxInt(width-len(cmdStr)-len(timeStr)-numExtraChars, 0))
	return fmt.Sprintf("$ %s  %s%s", cmdStr, padding, libtime.LocalHourNow())
}

func rerun(f func() (string, error)) {
	if flagWatch {
		print("\033[H\033[2J") // clear the screen

		prevLines := 0
		nextLines := 0

		for true {
			str, err := f()
			if err != nil {
				fmt.Println()
				errors.Exit(err)
			}

			str = watchHeader() + "\n" + str
			str = strings.TrimRight(str, "\n") + "\n" // ensure a single new line at the end
			strSlice := strings.Split(str, "\n")
			nextLines = len(strSlice)

			for prevLines > nextLines {
				fmt.Printf("\033[%dA\033[2K", 1) // move the cursor up and clear the line
				prevLines--
			}

			for i := 0; i < prevLines; i++ {
				fmt.Printf("\033[%dA", 1) // move the cursor up
			}

			prevLines = nextLines

			for _, strLine := range strSlice {
				fmt.Printf("\033[2K%s\n", strLine) // clear the line and print the new line
			}

			time.Sleep(time.Second)
		}
	} else {
		str, err := f()
		if err != nil {
			errors.Exit(err)
		}
		fmt.Println(str)
	}
}
