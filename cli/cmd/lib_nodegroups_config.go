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
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

func getNodeGroupsConfig(awsClient *aws.Client, nodeGroupsConfigFile string) ([]*clusterconfig.NodeGroup, error) {
	nodegroupsBytes, err := files.ReadFileBytes(nodeGroupsConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, nodeGroupsConfigFile)
	}

	nodegroupData, err := cr.ReadYAMLBytes(nodegroupsBytes)
	if err != nil {
		return nil, errors.Wrap(err, nodeGroupsConfigFile)
	}

	nodegroupDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(nodegroupData)
	if !ok {
		return nil, errors.Wrap(ErrorMalformedConfig(), nodeGroupsConfigFile)
	}

	nodegroups := make([]*clusterconfig.NodeGroup, len(nodegroupDataSlice))

	for i, data := range nodegroupDataSlice {
		nodegroup := clusterconfig.NodeGroup{}
		errs := cr.Struct(&nodegroup, data, clusterconfig.NodeGroupsValidation)
		if errors.HasError(errs) {
			err = errors.Wrap(errors.FirstError(errs...), nodeGroupsConfigFile, s.Index(i))
			return nil, errors.Append(err, fmt.Sprintf("\n\nnodegroups configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		}
		nodegroups[i] = &nodegroup
	}

	return nodegroups, nil
}
