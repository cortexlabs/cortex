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

	"github.com/spf13/cobra"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

var flagPrint bool

func init() {
	addEnvFlag(configureCmd)
	configureCmd.PersistentFlags().BoolVarP(&flagPrint, "print", "p", false, "print the configuration")
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "configure the CLI",
	Long: `
This command configures the Cortex URL and AWS credentials
in order to authenticate and send requests to Cortex.
The configuration is stored in ~/.cortex.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if flagPrint {
			cliConfig := getDefaults()
			fmt.Printf("Operator URL:           %s\n", cliConfig.CortexURL)
			fmt.Printf("AWS Access Key ID:      %s\n", cliConfig.AWSAccessKeyID)
			fmt.Printf("AWS Secret Access Key:  %s\n", s.MaskString(cliConfig.AWSSecretAccessKey, 4))
			return
		}

		configure()
	},
}
