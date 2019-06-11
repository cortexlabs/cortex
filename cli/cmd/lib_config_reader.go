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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func appRootOrBlank() string {
	dir, err := os.Getwd()
	if err != nil {
		errors.Exit(err)
	}
	for true {
		if err := files.CheckFile(filepath.Join(dir, "app.yaml")); err == nil {
			return dir
		}
		if dir == "/" {
			return ""
		}
		dir = files.ParentDir(dir)
	}
	return "" // unreachable
}

func mustAppRoot() string {
	appRoot := appRootOrBlank()
	if appRoot == "" {
		errors.Exit(ErrorCliNotInAppDir())
	}
	return appRoot
}

func YamlPaths(dir string) []string {
	yamlPaths, err := files.ListDirRecursive(dir, false, files.IgnoreNonYAML)
	if err != nil {
		errors.Exit(err)
	}
	return yamlPaths
}

func pythonPaths(dir string) []string {
	pyPaths, err := files.ListDirRecursive(dir, false, files.IgnoreNonPython)
	if err != nil {
		errors.Exit(err)
	}
	return pyPaths
}

func allConfigPaths(root string) []string {
	exportPaths := strset.New()
	requirementsPath := filepath.Join(root, consts.RequirementsTxt)
	if err := files.CheckFile(requirementsPath); err == nil {
		exportPaths.Add(requirementsPath)
	}

	customPackagesRoot := filepath.Join(root, consts.PackageDir)
	if err := files.CheckDir(customPackagesRoot); err == nil {
		customPackagesPaths, err := files.ListDirRecursive(customPackagesRoot, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders, files.IgnorePythonGeneratedFiles)
		if err != nil {
			errors.Exit(err)
		}
		exportPaths.Add(customPackagesPaths...)
	}
	exportPaths.Add(YamlPaths(root)...)
	exportPaths.Add(pythonPaths(root)...)

	return exportPaths.Slice()
}

func appNameFromConfig() (string, error) {
	appRoot := mustAppRoot()
	return userconfig.ReadAppName(filepath.Join(appRoot, "app.yaml"), "app.yaml")
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
