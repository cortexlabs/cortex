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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
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

func loadPythonPackages(files map[string][]byte, datasetVersion string) (context.PythonPackages, error) {
	pythonPackages := make(map[string]*context.PythonPackage)

	if reqFileBytes, ok := files[consts.RequirementsTxt]; ok {
		var buf bytes.Buffer
		buf.Write(reqFileBytes)
		// Invalidate cached packages when refreshed without depending on environment ID
		buf.WriteString(datasetVersion)
		id := hash.Bytes(buf.Bytes())
		pythonPackage := context.PythonPackage{
			ResourceFields: userconfig.ResourceFields{
				Name: consts.RequirementsTxt,
			},
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.PythonPackageType,
					MetadataKey:  filepath.Join(consts.PythonPackagesDir, id, "metadata.json"),
				},
			},
			SrcKey:     filepath.Join(consts.PythonPackagesDir, id, "src.txt"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, id, "package.zip"),
		}

		if err := config.AWS.UploadBytesToS3(reqFileBytes, pythonPackage.SrcKey); err != nil {
			return nil, errors.Wrap(err, "upload", "requirements")
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}

	customPackages := findCustomPackages(files)

	for _, packageName := range customPackages {
		zipBytesInputs := []zip.BytesInput{}
		var buf bytes.Buffer
		// Invalidate cached packages when refreshed without depending on environment ID
		buf.WriteString(datasetVersion)
		for filePath, fileBytes := range files {
			if strings.HasPrefix(filePath, filepath.Join(consts.PackageDir, packageName)) {
				buf.Write(fileBytes)
				zipBytesInputs = append(zipBytesInputs, zip.BytesInput{
					Content: fileBytes,
					Dest:    filePath[len(consts.PackageDir):],
				})
			}
		}
		id := hash.Bytes(buf.Bytes())
		pythonPackage := context.PythonPackage{
			ResourceFields: userconfig.ResourceFields{
				Name: packageName,
			},
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.PythonPackageType,
					MetadataKey:  filepath.Join(consts.PythonPackagesDir, id, "metadata.json"),
				},
			},
			SrcKey:     filepath.Join(consts.PythonPackagesDir, id, "src.zip"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, id, "package.zip"),
		}

		zipInput := zip.Input{
			Bytes: zipBytesInputs,
		}

		zipBytes, err := zip.ToMem(&zipInput)
		if err != nil {
			return nil, errors.Wrap(err, "zip", packageName)
		}

		if err := config.AWS.UploadBytesToS3(zipBytes, pythonPackage.SrcKey); err != nil {
			return nil, errors.Wrap(err, "upload", packageName)
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}
	return pythonPackages, nil
}
