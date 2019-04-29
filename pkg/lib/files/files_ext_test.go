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

package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

func IgnoreDir3(path string, fi os.FileInfo) (bool, error) {
	if fi.IsDir() && fi.Name() == "3" {
		return true, nil
	}
	return false, nil
}

func TestPrintFileTree(t *testing.T) {
	var filesList []string
	var cwd string
	var expectedTree string
	var expectedHeader string

	filesList = []string{
		"/1/2/3.txt",
		"/1/2/3/5.txt",
		"/1/2/4/5/6.txt",
		"/1/2/3/4.txt",
	}
	expectedTree = `
├── 3.txt
├── 3
│   ├── 4.txt
│   └── 5.txt
└── 4
    └── 5
        └── 6.txt
`

	cwd = ""
	expectedHeader = "/1/2/"
	require.Equal(t, expectedHeader+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	cwd = "/missing"
	expectedHeader = "/1/2/"
	require.Equal(t, expectedHeader+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	cwd = "/1/2"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	cwd = "/1/2/"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	cwd = "/1"
	expectedHeader = "./2/"
	require.Equal(t, expectedHeader+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	filesList = []string{
		"/1",
		"/2",
		"/1/1",
		"/1/2",
		"/2/1",
		"/2/2",
		"/1/1/1",
		"/1/1/2",
		"/1/2/1",
		"/1/2/2",
		"/2/1/1",
		"/2/1/2",
		"/2/2/1",
		"/2/2/2",
	}

	expectedTree = `
├── 1
├── 1
│   ├── 1
│   ├── 1
│   │   ├── 1
│   │   └── 2
│   ├── 2
│   └── 2
│       ├── 1
│       └── 2
├── 2
└── 2
    ├── 1
    ├── 1
    │   ├── 1
    │   └── 2
    ├── 2
    └── 2
        ├── 1
        └── 2
`
	require.Equal(t, "/"+expectedTree, files.FileTree(filesList, cwd, files.DirsSorted))

	expectedTree = `
├── 1
├── 2
├── 1
│   ├── 1
│   ├── 2
│   ├── 1
│   │   ├── 1
│   │   └── 2
│   └── 2
│       ├── 1
│       └── 2
└── 2
    ├── 1
    ├── 2
    ├── 1
    │   ├── 1
    │   └── 2
    └── 2
        ├── 1
        └── 2
`
	require.Equal(t, "/"+expectedTree, files.FileTree(filesList, cwd, files.DirsOnBottom))

	expectedTree = `
├── 1
│   ├── 1
│   │   ├── 1
│   │   └── 2
│   ├── 2
│   │   ├── 1
│   │   └── 2
│   ├── 1
│   └── 2
├── 2
│   ├── 1
│   │   ├── 1
│   │   └── 2
│   ├── 2
│   │   ├── 1
│   │   └── 2
│   ├── 1
│   └── 2
├── 1
└── 2
`
	require.Equal(t, "/"+expectedTree, files.FileTree(filesList, cwd, files.DirsOnTop))
}

func TestListDirRecursive(t *testing.T) {
	_, err := files.ListDirRecursive("/home/path/to/fake/dir", false)
	require.Error(t, err)

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
		filepath.Join(tmpDir, "4/.git/HEAD"),
	}

	err = files.MakeEmptyFiles(filesList...)
	require.NoError(t, err)

	var filesListRecursive []string
	var expected []string

	filesListRecursive, err = files.ListDirRecursive(tmpDir, false)
	require.NoError(t, err)
	require.ElementsMatch(t, filesList, filesListRecursive)

	filesListRecursive, err = files.ListDirRecursive(tmpDir, false, files.IgnoreHiddenFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/2.pyc"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = files.ListDirRecursive(tmpDir, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/2.pyc"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = files.ListDirRecursive(tmpDir, false, files.IgnoreHiddenFiles, IgnoreDir3, files.IgnorePythonGeneratedFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = files.ListDirRecursive(tmpDir, false, files.IgnoreNonPython)
	expected = []string{
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
	}
	require.NoError(t, err)
	debug.Ppj(filesListRecursive)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = files.ListDirRecursive(tmpDir, true, files.IgnoreNonPython)
	expected = []string{
		filepath.Join("2.py"),
		filepath.Join("3/1.py"),
		filepath.Join("3/2/1.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)
}
