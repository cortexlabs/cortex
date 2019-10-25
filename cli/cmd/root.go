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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

var cmdStr string

var configFileExts = []string{"yaml", "yml"}

var localDir string
var cachedClusterConfigPath string

func init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		errors.Exit(err)
	}

	localDir = filepath.Join(homeDir, ".cortex")
	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		errors.Exit(err)
	}

	cachedClusterConfigPath = filepath.Join(localDir, "cluster.yaml")

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
	Short:   "deploy machine learning models in production",
	Long:    `Deploy machine learning models in production`,
}

// Copied from https://github.com/spf13/cobra/blob/master/command.go
var helpCmd = &cobra.Command{
	Use:   "help [command]",
	Short: "help about any command",
	Long: `help provides help for any command in the CLI.
Type ` + rootCmd.Name() + ` help [path to command] for full details.`,
	Run: func(c *cobra.Command, args []string) {
		cmd, _, e := c.Root().Find(args)
		if cmd == nil || e != nil {
			c.Printf("Unknown help topic %#q\n", args)
			c.Root().Usage()
		} else {
			cmd.InitDefaultHelpFlag()
			cmd.Help()
		}
	},
}

func Execute() {
	defer errors.RecoverAndExit()
	rootCmd.SetHelpCommand(helpCmd)

	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(deleteCmd)

	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(clusterCmd)

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(supportCmd)
	rootCmd.AddCommand(versionCmd)

	printLeadingNewLine()
	rootCmd.Execute()
}

func printLeadingNewLine() {
	for _, arg := range os.Args[1:] {
		if arg == "completion" {
			return
		}
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

		var prevStrSlice []string

		for true {
			nextStr, err := f()
			if err != nil {
				fmt.Println()
				errors.Exit(err)
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
			errors.Exit(err)
		}
		fmt.Println(str)
	}
}
