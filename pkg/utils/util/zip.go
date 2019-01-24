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

package util

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

type ZipFileInput struct {
	Source string
	Dest   string
}

type ZipBytesInput struct {
	Content []byte
	Dest    string
}

type ZipDirInput struct {
	Source             string
	Dest               string
	IgnoreFns          []IgnoreFn
	Flatten            bool
	RemovePrefix       string
	RemoveCommonPrefix bool
}

type ZipFileListInput struct {
	Sources            []string
	Dest               string
	Flatten            bool
	RemovePrefix       string
	RemoveCommonPrefix bool
}

type ZipInput struct {
	Files          []ZipFileInput
	Bytes          []ZipBytesInput
	Dirs           []ZipDirInput
	FileLists      []ZipFileListInput
	AddPrefix      string   // Gets added to every item
	EmptyFiles     []string // Empty files to be created
	AllowMissing   bool     // Don't error if a file/dir doesn't exist
	AllowOverwrite bool     // Don't error if a file in the zip is overwritten
}

func ZipToWriter(zipInput *ZipInput, writer io.Writer) error {
	archive := zip.NewWriter(writer)
	addedPaths := &map[string]bool{}
	var err error

	for _, byteInput := range zipInput.Bytes {
		err = addBytesToZip(&byteInput, zipInput, archive, addedPaths)
		if err != nil {
			archive.Close()
			return err
		}
	}

	for _, fileInput := range zipInput.Files {
		err = addFileToZip(&fileInput, zipInput, archive, addedPaths)
		if err != nil {
			archive.Close()
			return err
		}
	}

	for _, dirInput := range zipInput.Dirs {
		err = addDirToZip(&dirInput, zipInput, archive, addedPaths)
		if err != nil {
			archive.Close()
			return err
		}
	}

	for _, fileListInput := range zipInput.FileLists {
		err = addFileListToZip(&fileListInput, zipInput, archive, addedPaths)
		if err != nil {
			archive.Close()
			return err
		}
	}

	for _, emptyFilePath := range zipInput.EmptyFiles {
		err = addEmptyFileToZip(emptyFilePath, zipInput, archive, addedPaths)
		if err != nil {
			archive.Close()
			return err
		}
	}

	err = archive.Close()
	if err != nil {
		return errors.Wrap(err, s.ErrCreateZip)
	}
	return nil
}

func Zip(zipInput *ZipInput, destPath string) error {
	zipfile, err := os.Create(destPath)
	if err != nil {
		return errors.Wrap(err, s.ErrCreateFile(destPath))
	}

	err = ZipToWriter(zipInput, zipfile)
	if err != nil {
		zipfile.Close()
		return err
	}

	err = zipfile.Close()
	if err != nil {
		return errors.Wrap(err, destPath, s.ErrCreateZip)
	}
	return nil
}

func ZipToMem(zipInput *ZipInput) ([]byte, error) {
	buf := new(bytes.Buffer)

	err := ZipToWriter(zipInput, buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func addBytesToZip(byteInput *ZipBytesInput, zipInput *ZipInput, archive *zip.Writer, addedPaths *map[string]bool) error {
	path := filepath.Join(zipInput.AddPrefix, byteInput.Dest)

	if !zipInput.AllowOverwrite {
		if _, ok := (*addedPaths)[path]; ok {
			return errors.New(s.ErrDuplicateZipPath(path))
		}
		(*addedPaths)[path] = true
	}

	f, err := archive.Create(path)
	if err != nil {
		return errors.Wrap(err, s.ErrCreateZip)
	}
	_, err = f.Write(byteInput.Content)
	if err != nil {
		return errors.Wrap(err, s.ErrCreateZip)
	}
	return nil
}

func addEmptyFileToZip(path string, zipInput *ZipInput, archive *zip.Writer, addedPaths *map[string]bool) error {
	byteInput := &ZipBytesInput{
		Content: []byte{},
		Dest:    path,
	}
	return addBytesToZip(byteInput, zipInput, archive, addedPaths)
}

func addFileToZip(fileInput *ZipFileInput, zipInput *ZipInput, archive *zip.Writer, addedPaths *map[string]bool) error {
	if !IsFile(fileInput.Source) {
		if !zipInput.AllowMissing {
			return errors.New(fileInput.Source, s.ErrFileDoesNotExist(fileInput.Source))
		} else {
			return nil
		}
	}

	content, err := ioutil.ReadFile(fileInput.Source)
	if err != nil {
		return errors.Wrap(err, s.ErrReadFile(fileInput.Source))
	}

	byteInput := &ZipBytesInput{
		Content: content,
		Dest:    fileInput.Dest,
	}
	return addBytesToZip(byteInput, zipInput, archive, addedPaths)
}

func addDirToZip(dirInput *ZipDirInput, zipInput *ZipInput, archive *zip.Writer, addedPaths *map[string]bool) error {
	if !IsDir(dirInput.Source) {
		if !zipInput.AllowMissing {
			return errors.New(s.ErrDirDoesNotExist(dirInput.Source))
		} else {
			return nil
		}
	}

	paths, err := ListDirRecursive(dirInput.Source, true, dirInput.IgnoreFns...)
	if err != nil {
		return err
	}

	commonPrefix := ""
	if dirInput.RemoveCommonPrefix {
		commonPrefix = s.LongestCommonPrefix(paths...)
	}

	for _, path := range paths {
		file := filepath.Join(dirInput.Source, path)

		if dirInput.Flatten {
			path = filepath.Base(path)
		} else {
			removePrefix := strings.TrimPrefix(dirInput.RemovePrefix, "/")
			path = strings.TrimPrefix(path, removePrefix)
			path = strings.TrimPrefix(path, commonPrefix)
		}

		fileInput := &ZipFileInput{
			Source: file,
			Dest:   filepath.Join(dirInput.Dest, path),
		}
		err = addFileToZip(fileInput, zipInput, archive, addedPaths)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileListToZip(fileListInput *ZipFileListInput, zipInput *ZipInput, archive *zip.Writer, addedPaths *map[string]bool) error {
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

		fileInput := &ZipFileInput{
			Source: fullPath,
			Dest:   filepath.Join(fileListInput.Dest, path),
		}
		err := addFileToZip(fileInput, zipInput, archive, addedPaths)
		if err != nil {
			return err
		}
	}

	return nil
}

func Unzip(src string, destPath string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, errors.Wrap(err, s.ErrUnzip)
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, errors.Wrap(err, s.ErrUnzip)
		}
		defer rc.Close()

		fpath := filepath.Join(destPath, f.Name)
		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return nil, errors.Wrap(err, s.ErrCreateDir(fpath))
			}
		} else {
			err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
			if err != nil {
				return nil, errors.Wrap(err, s.ErrCreateDir(filepath.Dir(fpath)))
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return nil, errors.Wrap(err, s.ErrCreateFile(fpath))
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				return nil, errors.Wrap(err, s.ErrCreateFile(fpath))
			}
		}
	}
	return filenames, nil
}

func UnzipMemToMem(zipBytes []byte) (map[string][]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, errors.Wrap(err, s.ErrUnzip)
	}

	return UnzipToMem(r)
}

func UnzipFileToMem(src string) (map[string][]byte, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return nil, errors.Wrap(err, s.ErrUnzip)
	}
	defer r.Close()

	return UnzipToMem(&r.Reader)
}

func UnzipToMem(r *zip.Reader) (map[string][]byte, error) {
	contents := map[string][]byte{}

	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, errors.Wrap(err, s.ErrUnzip)
			}
			defer rc.Close()

			bytes, err := ioutil.ReadAll(rc)
			if err != nil {
				return nil, errors.Wrap(err, s.ErrUnzip)
			}

			path := strings.TrimPrefix(f.Name, "/")
			contents[path] = bytes
		}
	}
	return contents, nil
}
