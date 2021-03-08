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
	"path"
	"path/filepath"
	"regexp"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

var _cachedGCPClusterConfigRegex = regexp.MustCompile(`^cluster-gcp_\S+\.yaml$`)

func cachedGCPClusterConfigPath(clusterName string, project string, zone string) string {
	return filepath.Join(_localDir, fmt.Sprintf("cluster-gcp_%s_%s_%s.yaml", clusterName, project, zone))
}

func existingCachedGCPClusterConfigPaths() []string {
	paths, err := files.ListDir(_localDir, false)
	if err != nil {
		return nil
	}

	var matches []string
	for _, p := range paths {
		if _cachedGCPClusterConfigRegex.MatchString(path.Base(p)) {
			matches = append(matches, p)
		}
	}

	return matches
}

func readUserGCPClusterConfigFile(clusterConfig *clusterconfig.GCPConfig, filePath string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.GCPFullManagedValidation, filePath)
	if errors.HasError(errs) {
		return errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	return nil
}

func getNewGCPClusterAccessConfig(clusterConfigFile string) (*clusterconfig.GCPAccessConfig, error) {
	accessConfig := &clusterconfig.GCPAccessConfig{}

	errs := cr.ParseYAMLFile(accessConfig, clusterconfig.GCPAccessValidation, clusterConfigFile)
	if errors.HasError(errs) {
		return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	return accessConfig, nil
}

func getGCPClusterAccessConfigWithCache(disallowPrompt bool) (*clusterconfig.GCPAccessConfig, error) {
	accessConfig := &clusterconfig.GCPAccessConfig{
		ImageManager: "quay.io/cortexlabs/manager:" + consts.CortexVersion,
	}

	cachedPaths := existingCachedGCPClusterConfigPaths()
	if len(cachedPaths) == 1 {
		cachedAccessConfig := &clusterconfig.GCPAccessConfig{}
		cr.ParseYAMLFile(cachedAccessConfig, clusterconfig.GCPAccessValidation, cachedPaths[0])
		accessConfig.ClusterName = cachedAccessConfig.ClusterName
		accessConfig.Project = cachedAccessConfig.Project
		accessConfig.Zone = cachedAccessConfig.Zone
	}

	if _flagClusterGCPConfig != "" {
		errs := cr.ParseYAMLFile(accessConfig, clusterconfig.GCPAccessValidation, _flagClusterGCPConfig)
		if errors.HasError(errs) {
			return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		}
	}

	if _flagClusterGCPName != "" {
		accessConfig.ClusterName = _flagClusterGCPName
	}
	if _flagClusterGCPZone != "" {
		accessConfig.Zone = _flagClusterGCPZone
	}
	if _flagClusterGCPProject != "" {
		accessConfig.Project = _flagClusterGCPProject
	}

	if accessConfig.ClusterName == "" || accessConfig.Project == "" || accessConfig.Zone == "" {
		return nil, ErrorGCPClusterAccessConfigRequired()
	}

	return accessConfig, nil
}

func getGCPInstallClusterConfig(gcpClient *gcp.Client, clusterConfigFile string, disallowPrompt bool) (*clusterconfig.GCPConfig, error) {
	clusterConfig := &clusterconfig.GCPConfig{}

	err := readUserGCPClusterConfigFile(clusterConfig, clusterConfigFile)
	if err != nil {
		return nil, err
	}

	clusterConfig.Telemetry, err = readTelemetryConfig()
	if err != nil {
		return nil, err
	}

	err = clusterConfig.Validate(gcpClient)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		return nil, errors.Wrap(err, clusterConfigFile)
	}

	confirmGCPInstallClusterConfig(clusterConfig, disallowPrompt)

	return clusterConfig, nil
}

func confirmGCPInstallClusterConfig(clusterConfig *clusterconfig.GCPConfig, disallowPrompt bool) {
	fmt.Printf("a cluster named \"%s\" will be created in %s (zone: %s)\n\n", clusterConfig.ClusterName, clusterConfig.Project, clusterConfig.Zone)

	if !disallowPrompt {
		exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://docs.cortex.dev/v/%s/ for more information", consts.CortexVersionMinor)
		prompt.YesOrExit("would you like to continue?", "", exitMessage)
	}
}
