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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	tmpDir, err := files.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	filesList := []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "3/2/3/.tmp"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/2.pyc"),
		filepath.Join(tmpDir, "5/4/3/2/1.txt"),
		filepath.Join(tmpDir, "5/4/3/2/1.py"),
		filepath.Join(tmpDir, "5/4/3/2/2/1.py"),
	}

	err = files.MakeEmptyFiles(filesList[0], filesList[1:]...)
	require.NoError(t, err)

	var input *Input
	var expected []string

	input = &Input{
		Bytes: []BytesInput{
			{
				Content: []byte(""),
				Dest:    "text.txt",
			},
			{
				Content: []byte(""),
				Dest:    "test2/text2.txt",
			},
			{
				Content: []byte(""),
				Dest:    "/test3/text3.txt",
			},
		},
	}
	expected = []string{
		"text.txt",
		"test2/text2.txt",
		"test3/text3.txt",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Files: []FileInput{
			{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1.txt",
			},
			{
				Source: filepath.Join(tmpDir, "3/1.py"),
				Dest:   "3/1.py",
			},
			{
				Source: filepath.Join(tmpDir, "3/2/1.py"),
				Dest:   "3/2/1.py",
			},
			{
				Source: filepath.Join(tmpDir, "3/2/3/.tmp"),
				Dest:   "3/2/3/.tmp",
			},
			{
				Source: filepath.Join(tmpDir, "4/2.pyc"),
				Dest:   "4/4/2.pyc",
			},
		},
	}
	expected = []string{
		"1.txt",
		"3/1.py",
		"3/2/1.py",
		"3/2/3/.tmp",
		"4/4/2.pyc",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Bytes: []BytesInput{
			{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []FileInput{
			{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1/2/3.txt",
			},
		},
		AddPrefix: "test",
	}
	expected = []string{
		"test/text.txt",
		"test/1/2/3.txt",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Bytes: []BytesInput{
			{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []FileInput{
			{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1/2/3.txt",
			},
		},
		AddPrefix: "/test",
	}
	expected = []string{
		"test/text.txt",
		"test/1/2/3.txt",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		EmptyFiles: []string{
			"text.txt",
			"1/2/3.txt",
		},
	}
	expected = []string{
		"text.txt",
		"1/2/3.txt",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source: tmpDir,
			},
		},
	}
	expected = []string{
		"1.txt",
		"2.py",
		"3/1.py",
		"3/2/1.py",
		"3/2/2.txt",
		"3/2/3/.tmp",
		"4/1.yaml",
		"4/2.pyc",
		"5/4/3/2/1.txt",
		"5/4/3/2/1.py",
		"5/4/3/2/2/1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source: filepath.Join(tmpDir, "3"),
				Dest:   ".",
			},
		},
	}
	expected = []string{
		"1.py",
		"2/1.py",
		"2/2.txt",
		"2/3/.tmp",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:    tmpDir,
				Dest:      "/",
				IgnoreFns: []files.IgnoreFn{files.IgnoreHiddenFiles},
			},
		},
	}
	expected = []string{
		"1.txt",
		"2.py",
		"3/1.py",
		"3/2/1.py",
		"3/2/2.txt",
		"4/1.yaml",
		"4/2.pyc",
		"5/4/3/2/1.txt",
		"5/4/3/2/1.py",
		"5/4/3/2/2/1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:    filepath.Join(tmpDir, "3"),
				IgnoreFns: []files.IgnoreFn{files.IgnoreHiddenFiles},
				Dest:      "test3",
				Flatten:   true,
			},
			{
				Source: filepath.Join(tmpDir, "4"),
				Dest:   "test4/",
			},
		},
		AllowOverwrite: true,
	}
	expected = []string{
		"test3/1.py",
		"test3/2.txt",
		"test4/1.yaml",
		"test4/2.pyc",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:       filepath.Join(tmpDir, "5"),
				RemovePrefix: "4/3",
			},
		},
	}
	expected = []string{
		"2/1.txt",
		"2/1.py",
		"2/2/1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:       filepath.Join(tmpDir, "5"),
				RemovePrefix: "/4/3",
			},
		},
	}
	expected = []string{
		"2/1.txt",
		"2/1.py",
		"2/2/1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:  filepath.Join(tmpDir, "5"),
				Flatten: true,
			},
		},
		AllowOverwrite: true,
	}
	expected = []string{
		"1.txt",
		"1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:             filepath.Join(tmpDir, "5"),
				RemoveCommonPrefix: true,
			},
		},
	}
	expected = []string{
		"1.txt",
		"1.py",
		"2/1.py",
	}
	CheckArchive(input, expected, false, t)

	input = &Input{
		Bytes: []BytesInput{
			{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
			{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
		},
	}
	CheckArchive(input, nil, true, t)

	input = &Input{
		Bytes: []BytesInput{
			{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
		},
		EmptyFiles: []string{"1/text.txt"},
	}
	CheckArchive(input, nil, true, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:  filepath.Join(tmpDir, "3"),
				Flatten: true,
			},
		},
	}
	CheckArchive(input, nil, true, t)

	input = &Input{
		Dirs: []DirInput{
			{
				Source:  filepath.Join(tmpDir, "5"),
				Flatten: true,
			},
		},
	}
	CheckArchive(input, nil, true, t)
}

func CheckArchive(input *Input, expected []string, shouldErr bool, t *testing.T) {
	CheckZip(input, expected, shouldErr, t)
	CheckTar(input, expected, shouldErr, t)
	CheckTgz(input, expected, shouldErr, t)
}

func CheckZip(input *Input, expected []string, shouldErr bool, t *testing.T) {
	tmpDir, err := files.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	_, err = ZipToFile(input, filepath.Join(tmpDir, "archive.zip"))
	if shouldErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)

	_, err = UnzipFileToDir(filepath.Join(tmpDir, "archive.zip"), filepath.Join(tmpDir, "archive"))
	require.NoError(t, err)

	unzippedFiles, err := files.ListDirRecursive(filepath.Join(tmpDir, "archive"), true)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, unzippedFiles)

	contents, err := UnzipFileToMem(filepath.Join(tmpDir, "archive.zip"))
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))

	zipBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "archive.zip"))
	require.NoError(t, err)
	contents, err = UnzipMemToMem(zipBytes)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))
}

func CheckTar(input *Input, expected []string, shouldErr bool, t *testing.T) {
	tmpDir, err := files.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	_, err = TarToFile(input, filepath.Join(tmpDir, "archive.tar"))
	if shouldErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)

	_, err = UntarFileToDir(filepath.Join(tmpDir, "archive.tar"), filepath.Join(tmpDir, "archive"))
	require.NoError(t, err)

	untaredFiles, err := files.ListDirRecursive(filepath.Join(tmpDir, "archive"), true)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, untaredFiles)

	contents, err := UntarFileToMem(filepath.Join(tmpDir, "archive.tar"))
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))

	tarBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "archive.tar"))
	require.NoError(t, err)
	contents, err = UntarMemToMem(tarBytes)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))
}

func CheckTgz(input *Input, expected []string, shouldErr bool, t *testing.T) {
	tmpDir, err := files.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	_, err = TgzToFile(input, filepath.Join(tmpDir, "archive.tgz"))
	if shouldErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)

	_, err = UntgzFileToDir(filepath.Join(tmpDir, "archive.tgz"), filepath.Join(tmpDir, "archive"))
	require.NoError(t, err)

	untgzedFiles, err := files.ListDirRecursive(filepath.Join(tmpDir, "archive"), true)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, untgzedFiles)

	contents, err := UntgzFileToMem(filepath.Join(tmpDir, "archive.tgz"))
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))

	tgzBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "archive.tgz"))
	require.NoError(t, err)
	contents, err = UntgzMemToMem(tgzBytes)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))
}
