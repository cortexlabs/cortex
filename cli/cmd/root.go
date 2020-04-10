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
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var _cmdStr string

var _configFileExts = []string{"yaml", "yml"}

var _localDir string
var _localWorkSpace string
var _cliConfigPath string
var _clientIDPath string
var _emailPath string
var _debugPath string
var _cwd string

var _flagEnv string

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		err := errors.Wrap(err, "unable to determine current working directory")
		exit.Error(err)
	}
	_cwd = s.EnsureSuffix(cwd, "/")

	homeDir, err := homedir.Dir()
	if err != nil {
		err := errors.Wrap(err, "unable to determine home directory")
		exit.Error(err)
	}

	_localDir = filepath.Join(homeDir, ".cortex")
	err = os.MkdirAll(_localDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _localDir)
		exit.Error(err)
	}

	_localWorkSpace = filepath.Join(_localDir, "local_workspace")
	fmt.Println(_localWorkSpace)
	err = os.MkdirAll(_localWorkSpace, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _localDir)
		exit.Error(err)
	}

	_cliConfigPath = filepath.Join(_localDir, "cli.yaml")
	_clientIDPath = filepath.Join(_localDir, "client-id.txt")
	_emailPath = filepath.Join(_localDir, "email.txt")
	_debugPath = filepath.Join(_localDir, "cortex-debug.tgz")

	cobra.EnablePrefixMatching = true

	_cmdStr = "cortex"
	for _, arg := range os.Args[1:] {
		if arg == "-w" || arg == "--watch" {
			continue
		}
		_cmdStr += " " + arg
	}

	enableTelemetry, err := readTelemetryConfig()
	if err != nil {
		exit.Error(err)
	}
	if enableTelemetry {
		initTelemetry()
	}

	clusterInit()
	completionInit()
	deleteInit()
	deployInit()
	envInit()
	getInit()
	logsInit()
	predictInit()
	refreshInit()
	versionInit()
}

func initTelemetry() {
	cID := clientID()
	telemetry.Init(telemetry.Config{
		Enabled: true,
		UserID:  cID,
		Properties: map[string]string{
			"client_id": cID,
		},
		Environment: "cli",
		LogErrors:   false,
		BackoffMode: telemetry.NoBackoff,
	})
}

var _rootCmd = &cobra.Command{
	Use:     "cortex",
	Aliases: []string{"cx"},
	Short:   "deploy machine learning models in production",
}

func Execute() {
	defer exit.RecoverAndExit()

	cobra.EnableCommandSorting = false

	_rootCmd.AddCommand(_deployCmd)
	_rootCmd.AddCommand(_refreshCmd)
	_rootCmd.AddCommand(_getCmd)
	_rootCmd.AddCommand(_logsCmd)
	_rootCmd.AddCommand(_predictCmd)
	_rootCmd.AddCommand(_deleteCmd)

	_rootCmd.AddCommand(_clusterCmd)
	_rootCmd.AddCommand(_versionCmd)

	_rootCmd.AddCommand(_envCmd)
	_rootCmd.AddCommand(_completionCmd)

	updateRootUsage()

	printLeadingNewLine()
	_rootCmd.Execute()

	exit.Ok()
}

func updateRootUsage() {
	defaultUsageFunc := _rootCmd.UsageFunc()
	usage := _rootCmd.UsageString()

	_rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		if cmd != _rootCmd {
			return defaultUsageFunc(cmd)
		}

		usage = strings.Replace(usage, "Usage:\n  cortex [command]\n\nAliases:\n  cortex, cx\n\n", "", 1)
		usage = strings.Replace(usage, "Available Commands:", "api commands:", 1)
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

type commandType int

const (
	_generalCommandType commandType = iota
	_clusterCommandType
)

const _envToUseUsage = "environment to use"
const _envToConfigureUsage = "environment to configure"

func addEnvFlag(cmd *cobra.Command, cmdType commandType, usage string) {
	defaultEnv := getDefaultEnv(cmdType)
	cmd.Flags().StringVarP(&_flagEnv, "env", "e", defaultEnv, usage)
}
