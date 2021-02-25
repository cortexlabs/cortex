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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	_cmdStr string

	_configFileExts = []string{"yaml", "yml"}
	_flagVerbose    bool
	_flagOutput     = flags.PrettyOutputType

	_credentialsCacheDir string
	_localDir            string
	_cliConfigPath       string
	_clientIDPath        string
	_emailPath           string
	_debugPath           string
	_cwd                 string
	_homeDir             string
)

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
	_homeDir = s.EnsureSuffix(homeDir, "/")

	_localDir = os.Getenv("CORTEX_CLI_CONFIG_DIR")
	if _localDir != "" {
		_localDir = files.UserRelToAbsPath(_localDir)
	} else {
		_localDir = filepath.Join(homeDir, ".cortex")
	}

	err = os.MkdirAll(_localDir, os.ModePerm)
	if err != nil {
		err := errors.Wrap(err, "unable to write to home directory", _localDir)
		exit.Error(err)
	}

	// ~/.cortex/credentials/
	_credentialsCacheDir = filepath.Join(_localDir, "credentials")
	err = os.MkdirAll(_credentialsCacheDir, os.ModePerm)
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
	clusterGCPInit()
	completionInit()
	deleteInit()
	deployInit()
	envInit()
	getInit()
	logsInit()
	patchInit()
	refreshInit()
	versionInit()
}

func initTelemetry() {
	cID := clientID()

	invoker := os.Getenv("CORTEX_CLI_INVOKER")
	if invoker == "" {
		invoker = "direct"
	}

	telemetry.Init(telemetry.Config{
		Enabled: true,
		UserID:  cID,
		Properties: map[string]string{
			"client_id": cID,
			"invoker":   invoker,
		},
		Environment: "cli",
		LogErrors:   false,
		BackoffMode: telemetry.NoBackoff,
	})
}

var _rootCmd = &cobra.Command{
	Use:     "cortex",
	Aliases: []string{"cx"},
	Short:   "model serving at scale",
}

func Execute() {
	defer exit.RecoverAndExit()

	cobra.EnableCommandSorting = false

	_rootCmd.AddCommand(_deployCmd)
	_rootCmd.AddCommand(_getCmd)
	_rootCmd.AddCommand(_patchCmd)
	_rootCmd.AddCommand(_logsCmd)
	_rootCmd.AddCommand(_refreshCmd)
	_rootCmd.AddCommand(_deleteCmd)

	_rootCmd.AddCommand(_clusterCmd)
	_rootCmd.AddCommand(_clusterGCPCmd)

	_rootCmd.AddCommand(_envCmd)
	_rootCmd.AddCommand(_versionCmd)
	_rootCmd.AddCommand(_completionCmd)

	updateRootUsage()

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
		usage = strings.Replace(usage, "\n  env ", "\n\nother commands:\n  env ", 1)
		usage = strings.Replace(usage, "\n\nUse \"cortex [command] --help\" for more information about a command.", "", 1)

		cmd.Print(usage)

		return nil
	})
}

func addVerboseFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&_flagVerbose, "verbose", "v", false, "show additional information (only applies to pretty output format)")
}

func wasEnvFlagProvided(cmd *cobra.Command) bool {
	envFlagProvided := false
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Shorthand == "e" && flag.Changed && flag.Value.String() != "" {
			envFlagProvided = true
		}
	})

	return envFlagProvided
}

func printEnvIfNotSpecified(envName string, cmd *cobra.Command) error {
	out, err := envStringIfNotSpecified(envName, cmd)
	if err != nil {
		return err
	}

	if out != "" {
		fmt.Print(out)
	}

	return nil
}

func envStringIfNotSpecified(envName string, cmd *cobra.Command) (string, error) {
	envNames, err := listConfiguredEnvNames()
	if err != nil {
		return "", err
	}

	if _flagOutput == flags.PrettyOutputType && !wasEnvFlagProvided(cmd) && len(envNames) > 1 {
		return fmt.Sprintf("using %s environment\n\n", envName), nil
	}

	return "", nil
}

func mixedPrint(a interface{}) error {
	jsonBytes, err := libjson.Marshal(a)
	if err != nil {
		return err
	}
	fmt.Print(fmt.Sprintf("~~cortex~~%s~~cortex~~", base64.StdEncoding.EncodeToString(jsonBytes)))
	return nil
}
