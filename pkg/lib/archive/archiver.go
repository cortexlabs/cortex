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
	"io"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type archiveType int

const (
	unknownArchiveType archiveType = iota
	zipArchiveType
	tarArchiveType
	tgzArchiveType
)

type archiver interface {
	add(reader io.Reader, dest string, size int64) error
	close() error
}

func archive(input *Input, arc archiver) (strset.Set, error) {
	addedPaths := strset.New()
	var err error

	for i := range input.Bytes {
		err = addBytesToArchive(&input.Bytes[i], input, arc, addedPaths)
		if err != nil {
			return nil, err
		}
	}

	for i := range input.Files {
		err = addFileToArchive(&input.Files[i], input, arc, addedPaths)
		if err != nil {
			return nil, err
		}
	}

	for i := range input.Dirs {
		err = addDirToArchive(&input.Dirs[i], input, arc, addedPaths)
		if err != nil {
			return nil, err
		}
	}

	for i := range input.FileLists {
		err = addFileListToArchive(&input.FileLists[i], input, arc, addedPaths)
		if err != nil {
			return nil, err
		}
	}

	for _, emptyFilePath := range input.EmptyFiles {
		err = addEmptyFileToArchive(emptyFilePath, input, arc, addedPaths)
		if err != nil {
			return nil, err
		}
	}

	return addedPaths, nil
}

func addBytesToArchive(byteInput *BytesInput, input *Input, arc archiver, addedPaths strset.Set) error {
	path := filepath.Join(input.AddPrefix, byteInput.Dest)
	path = strings.TrimPrefix(path, "/")

	if !input.AllowOverwrite {
		if addedPaths.Has(path) {
			return ErrorDuplicatePath(path)
		}
		addedPaths.Add(path)
	}

	reader := bytes.NewReader(byteInput.Content)
	return arc.add(reader, path, reader.Size())
}

func addFileToArchive(fileInput *FileInput, input *Input, arc archiver, addedPaths strset.Set) error {
	content, err := files.ReadFileBytes(fileInput.Source)
	if err != nil {
		return err
	}

	byteInput := &BytesInput{
		Content: content,
		Dest:    fileInput.Dest,
	}

	return addBytesToArchive(byteInput, input, arc, addedPaths)
}

func addDirToArchive(dirInput *DirInput, input *Input, arc archiver, addedPaths strset.Set) error {
	paths, err := files.ListDirRecursive(dirInput.Source, true, dirInput.IgnoreFns...)
	if err != nil {
		return err
	}

	commonPrefix := ""
	if dirInput.RemoveCommonPrefix {
		commonPrefix = s.LongestCommonPrefix(paths...)
	}

	removePrefix := strings.TrimPrefix(dirInput.RemovePrefix, "/")

	for _, path := range paths {
		file := filepath.Join(dirInput.Source, path)

		if dirInput.Flatten {
			path = filepath.Base(path)
		} else {
			path = strings.TrimPrefix(path, removePrefix)
			path = strings.TrimPrefix(path, commonPrefix)
		}

		fileInput := &FileInput{
			Source: file,
			Dest:   filepath.Join(dirInput.Dest, path),
		}

		err = addFileToArchive(fileInput, input, arc, addedPaths)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileListToArchive(fileListInput *FileListInput, input *Input, arc archiver, addedPaths strset.Set) error {
	commonPrefix := ""
	if fileListInput.RemoveCommonPrefix {
		commonPrefix = s.LongestCommonPrefix(fileListInput.Sources...)
	}

	for _, path := range fileListInput.Sources {
		fullPath := path

		if fileListInput.Flatten {
			path = filepath.Base(path)
		} else {
			path = strings.TrimPrefix(path, fileListInput.RemovePrefix)
			path = strings.TrimPrefix(path, commonPrefix)
		}

		fileInput := &FileInput{
			Source: fullPath,
			Dest:   filepath.Join(fileListInput.Dest, path),
		}

		err := addFileToArchive(fileInput, input, arc, addedPaths)
		if err != nil {
			return err
		}
	}

	return nil
}

func addEmptyFileToArchive(path string, input *Input, arc archiver, addedPaths strset.Set) error {
	byteInput := &BytesInput{
		Content: []byte{},
		Dest:    path,
	}
	return addBytesToArchive(byteInput, input, arc, addedPaths)
}

func archiveToWriter(input *Input, writer io.Writer, arcType archiveType) (strset.Set, error) {
	var arc archiver
	switch arcType {
	case zipArchiveType:
		arc = newZipArchiver(writer)
	case tarArchiveType:
		arc = newTarArchiver(writer)
	case tgzArchiveType:
		arc = newTgzArchiver(writer)
	default:
		return nil, errors.ErrorUnexpected("unknown archive type:", arcType)
	}

	paths, err := archive(input, arc)
	if err != nil {
		arc.close()
		return nil, err
	}

	err = arc.close()
	if err != nil {
		return nil, errors.Wrap(err, _errStrCreateArchive)
	}

	return paths, nil
}

func archiveToMem(input *Input, arcType archiveType) ([]byte, strset.Set, error) {
	buf := new(bytes.Buffer)

	paths, err := archiveToWriter(input, buf, arcType)
	if err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), paths, nil
}

func archiveToFile(input *Input, destDir string, arcType archiveType) (strset.Set, error) {
	cleanDestDir, err := files.EscapeTilde(destDir)
	if err != nil {
		return nil, err
	}

	archiveFile, err := files.Create(cleanDestDir)
	if err != nil {
		return nil, err
	}

	paths, err := archiveToWriter(input, archiveFile, arcType)
	if err != nil {
		archiveFile.Close()
		return nil, err
	}

	err = archiveFile.Close()
	if err != nil {
		return nil, errors.Wrap(err, destDir, _errStrCreateArchive)
	}

	return paths, nil
}
