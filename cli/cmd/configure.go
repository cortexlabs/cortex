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
	"github.com/spf13/cobra"
)

var (
	cortexURL          string
	awsAccessKeyID     string
	awsSecretAccessKey string
)

func init() {
	configureCmd.PersistentFlags().StringVar(&cortexURL, "cortexURL", "", "set Cortex URL")
	configureCmd.PersistentFlags().StringVar(&awsAccessKeyID, "awsAccessKeyID", "", "set AWS_ACCESS_KEY_ID")
	configureCmd.PersistentFlags().StringVar(&awsSecretAccessKey, "awsSecretAccessKey", "", "set SECRET_ACCESS_KEY")
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "configure the CLI",
	Long:  "Configure the CLI.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var override bool
		conf := &CLIConfig{}
		if cortexURL != "" {
			conf.CortexURL = cortexURL
			override = true
		}
		if awsAccessKeyID != "" {
			conf.AWSAccessKeyID = awsAccessKeyID
			override = true
		}
		if awsSecretAccessKey != "" {
			conf.AWSSecretAccessKey = awsSecretAccessKey
			override = true
		}

		if !override {
			conf = nil
		}
		configure(conf)
	},
}
