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
	"path/filepath"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
)

var cmdStr string

var configFileExts = []string{"yaml", "yml"}

var localDir string
var cachedClusterConfigPath string
var _cliConfigPath string
var _clientIDPath string
var _emailPath string

func init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		exit.Error(err)
	}

	localDir = filepath.Join(homeDir, ".cortex")
	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		exit.Error(err)
	}

	cachedClusterConfigPath = filepath.Join(localDir, "cluster.yaml")
	_cliConfigPath = filepath.Join(localDir, "cli.yaml")
	_clientIDPath = filepath.Join(localDir, "client-id.txt")
	_emailPath = filepath.Join(localDir, "email.txt")

	cobra.EnablePrefixMatching = true

	cmdStr = "cortex"
	for _, arg := range os.Args[1:] {
		if arg == "-w" || arg == "--watch" {
			continue
		}
		cmdStr += " " + arg
	}

	enableTelemetry, err := readTelemetryConfig()
	if err != nil {
		exit.Error(err)
	}
	if enableTelemetry {
		initTelemetry()
	}
}

func initTelemetry() {
	telemetry.Init(telemetry.Config{
		Enabled:              true,
		UserID:               clientID(),
		Environment:          "cli",
		LogErrors:            false,
		BlockDuplicateErrors: false,
	})
}

var rootCmd = &cobra.Command{
	Use:     "cortex",
	Aliases: []string{"cx"},
	Short:   "deploy machine learning models in production",
}

func Execute() {
	defer exit.RecoverAndExit()

	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(deleteCmd)

	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(supportCmd)
	rootCmd.AddCommand(completionCmd)

	updateRootUsage()

	printLeadingNewLine()
	rootCmd.Execute()

	exit.Ok()
}

func updateRootUsage() {
	defaultUsageFunc := rootCmd.UsageFunc()
	usage := rootCmd.UsageString()

	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		if cmd != rootCmd {
			return defaultUsageFunc(cmd)
		}

		usage = strings.Replace(usage, "Usage:\n  cortex [command]\n\nAliases:\n  cortex, cx\n\n", "", 1)
		usage = strings.Replace(usage, "Available Commands:", "deployment commands:", 1)
		usage = strings.Replace(usage, "\n  cluster", "\n\ncluster commands:\n  cluster", 1)
		usage = strings.Replace(usage, "\n  configure", "\n\nother commands:\n  configure", 1)
		usage = strings.Replace(usage, "\n\nUse \"cortex [command] --help\" for more information about a command.", "", 1)

		cmd.Print(usage)

		return nil
	})
}

func printLeadingNewLine() {
	if len(os.Args) == 2 && os.Args[1] == "completion" {
		return
	}
	fmt.Println("")
}

var flagEnv string
var flagAppName string

func addEnvFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "default", "environment")
}

func addAppNameFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagAppName, "deployment", "d", "", "deployment name")
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

		var prevStrSlice []string

		for true {
			nextStr, err := f()
			if err != nil {
				fmt.Println()
				exit.Error(err)
			}

			nextStr = watchHeader() + "\n\n" + nextStr
			nextStr = strings.TrimRight(nextStr, "\n") + "\n" // ensure a single new line at the end
			nextStrSlice := strings.Split(nextStr, "\n")

			terminalWidth := getTerminalWidth()

			nextNumLines := 0
			for _, strLine := range nextStrSlice {
				nextNumLines += (len(strLine)-1)/terminalWidth + 1
			}
			prevNumLines := 0
			for _, strLine := range prevStrSlice {
				prevNumLines += (len(strLine)-1)/terminalWidth + 1
			}

			for i := prevNumLines; i > nextNumLines; i-- {
				fmt.Printf("\033[%dA\033[2K", 1) // move the cursor up and clear the line
			}

			for i := 0; i < prevNumLines; i++ {
				fmt.Printf("\033[%dA", 1) // move the cursor up
			}

			for _, strLine := range nextStrSlice {
				fmt.Printf("\033[2K%s\n", strLine) // clear the line and print the new line
			}

			prevStrSlice = nextStrSlice

			time.Sleep(time.Second)
		}
	} else {
		str, err := f()
		if err != nil {
			exit.Error(err)
		}
		fmt.Println(str)
	}
}
