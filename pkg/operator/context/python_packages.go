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

var (
	PackageDir = "packages"
)

func loadPythonPackages(files map[string][]byte, env *context.Environment) (context.PythonPackages, error) {
	pythonPackages := make(map[string]*context.PythonPackage)

	if reqFileBytes, ok := files["requirements.txt"]; ok {
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
			Name:       "requirements.txt",
			RawKey:     filepath.Join(consts.PythonPackagesDir, "raw", id+".txt"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, "package", id+".zip"),
		}

		if err := aws.UploadBytesToS3(reqFileBytes, pythonPackage.RawKey); err != nil {
			return nil, errors.Wrap(err, "upload", "requirements")
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}

	var customPackages []string

	for filePath := range files {
		if strings.HasSuffix(filePath, "setup.py") {
			packageFolder, packageName := filepath.Split(filepath.Dir(filePath))
			baseDir := filepath.Dir(packageFolder)
			// TODO: throw warning if setup.py detected but not in expected path
			if strings.TrimPrefix(baseDir, "/") == PackageDir {
				customPackages = append(customPackages, packageName)
			}
		}
	}

	for _, packageName := range customPackages {
		zipBytesInputs := []util.ZipBytesInput{}
		var buf bytes.Buffer
		buf.WriteString(env.ID)
		for filePath, fileBytes := range files {
			if strings.HasPrefix(filePath, filepath.Join(PackageDir, packageName)) {
				buf.Write(fileBytes)
				zipBytesInputs = append(zipBytesInputs, util.ZipBytesInput{
					Content: fileBytes,
					Dest:    filePath[len(PackageDir):],
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
			RawKey:     filepath.Join(consts.PythonPackagesDir, "raw", id+".zip"),
			PackageKey: filepath.Join(consts.PythonPackagesDir, "package", id+".zip"),
		}

		zipInput := util.ZipInput{
			Bytes: zipBytesInputs,
		}

		zipBytes, err := util.ZipToMem(&zipInput)
		if err != nil {
			return nil, errors.Wrap(err, "zip", packageName)
		}

		if err := aws.UploadBytesToS3(zipBytes, pythonPackage.RawKey); err != nil {
			return nil, errors.Wrap(err, "upload", packageName)
		}

		pythonPackages[pythonPackage.Name] = &pythonPackage
	}
	return pythonPackages, nil
}
