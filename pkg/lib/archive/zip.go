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

package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

type zipArchiver struct {
	writer *zip.Writer
}

func newZipArchiver(writer io.Writer) *zipArchiver {
	return &zipArchiver{
		writer: zip.NewWriter(writer),
	}
}

func (arc *zipArchiver) add(reader io.Reader, dest string, size int64) error {
	writer, err := arc.writer.Create(dest)
	if err != nil {
		return errors.Wrap(err, _errStrCreateArchive)
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.Wrap(err, _errStrCreateArchive)
	}

	return nil
}

func (arc *zipArchiver) close() error {
	return arc.writer.Close()
}

func ZipToWriter(input *Input, writer io.Writer) (strset.Set, error) {
	return archiveToWriter(input, writer, zipArchiveType)
}

func ZipToFile(input *Input, destDir string) (strset.Set, error) {
	return archiveToFile(input, destDir, zipArchiveType)
}

func ZipToMem(input *Input) ([]byte, strset.Set, error) {
	return archiveToMem(input, zipArchiveType)
}

// Will create destDir if missing
func UnzipFileToDir(src string, destDir string) (strset.Set, error) {
	destDir, err := files.Clean(destDir)
	if err != nil {
		return nil, err
	}

	cleanSrc, err := files.EscapeTilde(src)
	if err != nil {
		return nil, err
	}

	filenames := strset.New()

	r, err := zip.OpenReader(cleanSrc)
	if err != nil {
		return nil, errors.Wrap(err, _errStrUnzip)
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, errors.Wrap(err, _errStrUnzip)
		}
		defer rc.Close()

		target := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			err := files.CreateDir(target)
			if err != nil {
				return nil, err
			}
		} else {
			filenames.Add(target)

			err := files.CreateDir(filepath.Dir(target))
			if err != nil {
				return nil, err
			}

			outFile, err := files.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				return nil, err
			}

			_, err = io.Copy(outFile, rc)
			if err != nil {
				outFile.Close()
				return nil, errors.Wrap(err, _errStrUnzip)
			}

			outFile.Close()
		}
	}
	return filenames, nil
}

func UnzipMemToMem(zipBytes []byte) (map[string][]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, errors.Wrap(err, _errStrUnzip)
	}

	return unzipReaderToMem(zipReader)
}

func UnzipFileToMem(src string) (map[string][]byte, error) {
	cleanSrc, err := files.Clean(src)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.OpenReader(cleanSrc)
	if err != nil {
		return nil, errors.Wrap(err, _errStrUnzip)
	}
	defer zipReader.Close()

	return unzipReaderToMem(&zipReader.Reader)
}

func unzipReaderToMem(zipReader *zip.Reader) (map[string][]byte, error) {
	fileMap := map[string][]byte{}

	for _, f := range zipReader.File {
		if !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, errors.Wrap(err, _errStrUnzip)
			}
			defer rc.Close()

			contents, err := ioutil.ReadAll(rc)
			if err != nil {
				return nil, errors.Wrap(err, _errStrUnzip)
			}

			path := strings.TrimPrefix(f.Name, "/")
			fileMap[path] = contents
		}
	}

	return fileMap, nil
}
