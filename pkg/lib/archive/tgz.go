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
	"bytes"
	"compress/gzip"
	"io"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

type tgzArchiver struct {
	tarArc     *tarArchiver
	gzipWriter *gzip.Writer
}

func newTgzArchiver(writer io.Writer) *tgzArchiver {
	gzipWriter := gzip.NewWriter(writer)
	return &tgzArchiver{
		tarArc:     newTarArchiver(gzipWriter),
		gzipWriter: gzipWriter,
	}
}

func (arc *tgzArchiver) add(reader io.Reader, dest string, size int64) error {
	return arc.tarArc.add(reader, dest, size)
}

func (arc *tgzArchiver) close() error {
	err1 := arc.tarArc.close()
	err2 := arc.gzipWriter.Close()
	return errors.FirstError(err1, err2)
}

func TgzToWriter(input *Input, writer io.Writer) (strset.Set, error) {
	return archiveToWriter(input, writer, tgzArchiveType)
}

func TgzToFile(input *Input, destDir string) (strset.Set, error) {
	return archiveToFile(input, destDir, tgzArchiveType)
}

func TgzToMem(input *Input) ([]byte, strset.Set, error) {
	return archiveToMem(input, tgzArchiveType)
}

// Will create destDir if missing
func UntgzReaderToDir(reader io.Reader, destDir string) (strset.Set, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	return UntarReaderToDir(gzipReader, destDir)
}

// Will create destDir if missing
func UntgzFileToDir(src string, destDir string) (strset.Set, error) {
	file, err := files.Open(src)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return UntgzReaderToDir(file, destDir)
}

func UntgzReaderToMem(reader io.Reader) (map[string][]byte, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	return UntarReaderToMem(gzipReader)
}

func UntgzMemToMem(tgzBytes []byte) (map[string][]byte, error) {
	return UntgzReaderToMem(bytes.NewReader(tgzBytes))
}

func UntgzFileToMem(src string) (map[string][]byte, error) {
	file, err := files.Open(src)
	if err != nil {
		return nil, errors.Wrap(err, _errStrUntar)
	}
	defer file.Close()

	return UntgzReaderToMem(file)
}
