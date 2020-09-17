/*
Copyright 2020 Cortex Labs, Inc.

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

package tar

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

// Will create destDir if missing
func UntarReaderToDir(reader io.Reader, destDir string, isGzip bool) ([]string, error) {
	destDir, err := files.Clean(destDir)
	if err != nil {
		return nil, err
	}

	var tr *tar.Reader

	if isGzip {
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)
	} else {
		tr = tar.NewReader(reader)
	}

	var filenames []string

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return filenames, nil

		case err != nil:
			return nil, err

		case header == nil:
			continue
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := files.CreateDir(target)
			if err != nil {
				return nil, err
			}

		case tar.TypeReg:
			filenames = append(filenames, target)

			err := files.CreateDir(filepath.Dir(target))
			if err != nil {
				return nil, err
			}

			outFile, err := files.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return nil, errors.Wrap(err, _errStrUntar)
			}
			outFile.Close()
		}
	}
}
