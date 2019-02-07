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

package context

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func findCustomPackages(files map[string][]byte) []string {
	var customPackages []string
	for filePath := range files {
		if strings.HasSuffix(filePath, "setup.py") {
			packageFolder, packageName := filepath.Split(filepath.Dir(filePath))
			baseDir := filepath.Dir(packageFolder)

			if strings.TrimPrefix(baseDir, "/") == consts.PackageDir {
				customPackages = append(customPackages, packageName)
			}
		}
	}

	return customPackages
}

func loadPythonPackages(files map[string][]byte, env *context.Environment) (context.PythonPackages, error) {
	pythonPackages := make(map[string]*context.PythonPackage)

	if reqFileBytes, ok := files[consts.RequirementsTxt]; ok {
		var buf bytes.Buffer
		buf.WriteString(env.ID)
		buf.Write(reqFileBytes)
		id := util.HashBytes(buf.Bytes())
		pythonPackage := context.PythonPackage{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.PythonPackageType,
				},
			},
			Name:       consts.RequirementsTxt,
			SrcKey:     filepath.Join(consts.PythonPackagesDir, id, "src.txt"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, id, "package.zip"),
		}

		if err := aws.UploadBytesToS3(reqFileBytes, pythonPackage.SrcKey); err != nil {
			return nil, errors.Wrap(err, "upload", "requirements")
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}

	customPackages := findCustomPackages(files)

	for _, packageName := range customPackages {
		zipBytesInputs := []util.ZipBytesInput{}
		var buf bytes.Buffer
		buf.WriteString(env.ID)
		for filePath, fileBytes := range files {
			if strings.HasPrefix(filePath, filepath.Join(consts.PackageDir, packageName)) {
				buf.Write(fileBytes)
				zipBytesInputs = append(zipBytesInputs, util.ZipBytesInput{
					Content: fileBytes,
					Dest:    filePath[len(consts.PackageDir):],
				})
			}
		}
		id := util.HashBytes(buf.Bytes())
		pythonPackage := context.PythonPackage{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.PythonPackageType,
				},
			},
			Name:       packageName,
			SrcKey:     filepath.Join(consts.PythonPackagesDir, id, "src.zip"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, id, "package.zip"),
		}

		zipInput := util.ZipInput{
			Bytes: zipBytesInputs,
		}

		zipBytes, err := util.ZipToMem(&zipInput)
		if err != nil {
			return nil, errors.Wrap(err, "zip", packageName)
		}

		if err := aws.UploadBytesToS3(zipBytes, pythonPackage.SrcKey); err != nil {
			return nil, errors.Wrap(err, "upload", packageName)
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}
	return pythonPackages, nil
}
