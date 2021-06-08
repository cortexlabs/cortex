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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

func getNodeGroupConfig(awsClient *aws.Client, nodeGroupConfigFile string) (clusterconfig.NodeGroup, error) {
	var nodegroup clusterconfig.NodeGroup

	errs := cr.ParseYAMLFile(&nodegroup, clusterconfig.NodeGroupsValidation, nodeGroupConfigFile)
	if errors.HasError(errs) {
		err := errors.Wrap(errors.FirstError(errs...), nodeGroupConfigFile)
		return clusterconfig.NodeGroup{}, errors.Append(err, fmt.Sprintf("\n\nnodegroups configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	return nodegroup, nil
}
