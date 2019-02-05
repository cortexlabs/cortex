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
	"os"
	"path/filepath"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func appRootOrBlank() string {
	dir, err := os.Getwd()
	if err != nil {
		errors.Exit(err)
	}
	for true {
		if util.IsFile(filepath.Join(dir, "app.yaml")) {
			return dir
		}
		if dir == "/" {
			return ""
		}
		dir = util.ParentDir(dir)
	}
	return "" // unreachable
}

func mustAppRoot() string {
	appRoot := appRootOrBlank()
	if appRoot == "" {
		errors.Exit(s.ErrCliNotInAppDir)
	}
	return appRoot
}

func yamlPaths(dir string) []string {
	yamlPaths, err := util.ListDirRecursive(dir, false, util.IgnoreNonYAML)
	if err != nil {
		errors.Exit(err)
	}
	return yamlPaths
}

func pythonPaths(dir string) []string {
	pyPaths, err := util.ListDirRecursive(dir, false, util.IgnoreNonPython)
	if err != nil {
		errors.Exit(err)
	}
	return pyPaths
}

func allConfigPaths(root string) []string {
	var exportPaths []string
	requirementsPath := filepath.Join(root, consts.RequirementsTxt)
	if util.IsFile(requirementsPath) {
		exportPaths = append(exportPaths, requirementsPath)
	}

	customPackagesRoot := filepath.Join(root, consts.PackageDir)
	if util.IsDir(customPackagesRoot) {
		customPackagesPaths, err := util.ListDirRecursive(customPackagesRoot, false, util.IgnoreDotFiles, util.IgnoreHiddenFolders, util.IgnorePYC)
		if err != nil {
			errors.Exit(err)
		}
		exportPaths = append(exportPaths, customPackagesPaths...)
	}
	exportPaths = append(exportPaths, yamlPaths(root)...)
	exportPaths = append(exportPaths, pythonPaths(root)...)
	return util.UniqueStrs(exportPaths)
}

func appNameFromConfig() (string, error) {
	appRoot := mustAppRoot()
	return userconfig.ReadAppName(filepath.Join(appRoot, "app.yaml"))
}

func AppNameFromFlagOrConfig() (string, error) {
	if flagAppName != "" {
		return flagAppName, nil
	}

	appName, err := appNameFromConfig()
	if err != nil {
		return "", err
	}

	return appName, nil
}
