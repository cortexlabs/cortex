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

package archive

import (
	"archive/tar"
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

type tarArchiver struct {
	writer *tar.Writer
}

func newTarArchiver(writer io.Writer) *tarArchiver {
	return &tarArchiver{
		writer: tar.NewWriter(writer),
	}
}

func (arc *tarArchiver) add(reader io.Reader, dest string, size int64) error {
	header := &tar.Header{
		Name: dest,
		Size: size,
	}

	err := arc.writer.WriteHeader(header)
	if err != nil {
		return errors.Wrap(err, _errStrCreateTar)
	}

	_, err = io.Copy(arc.writer, reader)
	if err != nil {
		return errors.Wrap(err, _errStrCreateTar)
	}

	return nil
}

func (arc *tarArchiver) close() error {
	return arc.writer.Close()
}

func TarToWriter(input *Input, writer io.Writer) (strset.Set, error) {
	return archiveToWriter(input, writer, tarArchiveType)
}

func TarToFile(input *Input, destDir string) (strset.Set, error) {
	return archiveToFile(input, destDir, tarArchiveType)
}

func TarToMem(input *Input) ([]byte, strset.Set, error) {
	return archiveToMem(input, tarArchiveType)
}

// Will create destDir if missing
func UntarReaderToDir(reader io.Reader, destDir string) (strset.Set, error) {
	destDir, err := files.Clean(destDir)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(reader)

	filenames := strset.New()

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return filenames, nil

		case err != nil:
			return nil, errors.WithStack(err)

		case header == nil:
			continue
		}

		name := strings.TrimPrefix(header.Name, "/")
		target := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := files.CreateDir(target)
			if err != nil {
				return nil, err
			}

		case tar.TypeReg:
			filenames.Add(target)

			err := files.CreateDir(filepath.Dir(target))
			if err != nil {
				return nil, err
			}

			outFile, err := files.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				return nil, err
			}

			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				outFile.Close()
				return nil, errors.Wrap(err, _errStrUntar)
			}

			outFile.Close()
		}
	}
}

// Will create destDir if missing
func UntarFileToDir(src string, destDir string) (strset.Set, error) {
	file, err := files.Open(src)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return UntarReaderToDir(file, destDir)
}

func UntarReaderToMem(reader io.Reader) (map[string][]byte, error) {
	fileMap := map[string][]byte{}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return fileMap, nil

		case err != nil:
			return nil, err

		case header == nil:
			continue
		}

		if header.Typeflag == tar.TypeReg {
			contents, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, _errStrUntar)
			}

			path := strings.TrimPrefix(header.Name, "/")
			fileMap[path] = contents
		}
	}
}

func UntarMemToMem(tarBytes []byte) (map[string][]byte, error) {
	return UntarReaderToMem(bytes.NewReader(tarBytes))
}

func UntarFileToMem(src string) (map[string][]byte, error) {
	file, err := files.Open(src)
	if err != nil {
		return nil, errors.Wrap(err, _errStrUntar)
	}
	defer file.Close()

	return UntarReaderToMem(file)
}
