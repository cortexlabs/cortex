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

package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func IgnoreDir3(path string, fi os.FileInfo) (bool, error) {
	if fi.IsDir() && fi.Name() == "3" {
		return true, nil
	}
	return false, nil
}

func TestPrintFileTree(t *testing.T) {
	var files []string
	var cwd string
	var expectedTree string
	var expectedHeader string

	files = []string{
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
	require.Equal(t, expectedHeader+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

	cwd = "/missing"
	expectedHeader = "/1/2/"
	require.Equal(t, expectedHeader+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

	cwd = "/1/2"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

	cwd = "/1/2/"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

	cwd = "/1"
	expectedHeader = "./2/"
	require.Equal(t, expectedHeader+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

	files = []string{
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
	require.Equal(t, "/"+expectedTree, util.FileTree(files, cwd, util.DirsSorted))

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
	require.Equal(t, "/"+expectedTree, util.FileTree(files, cwd, util.DirsOnBottom))

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
	require.Equal(t, "/"+expectedTree, util.FileTree(files, cwd, util.DirsOnTop))
}

func TestListDirRecursive(t *testing.T) {
	_, err := util.ListDirRecursive("/home/path/to/fake/dir", false)
	require.Error(t, err)

	tmpDir, err := util.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	files := []string{
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

	err = util.MakeEmptyFiles(files...)
	require.NoError(t, err)

	var fileList []string
	var expected []string

	fileList, err = util.ListDirRecursive(tmpDir, false)
	require.NoError(t, err)
	require.ElementsMatch(t, files, fileList)

	fileList, err = util.ListDirRecursive(tmpDir, false, util.IgnoreHiddenFiles)
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
	require.ElementsMatch(t, expected, fileList)

	fileList, err = util.ListDirRecursive(tmpDir, false, util.IgnoreHiddenFiles, util.IgnoreHiddenFolders)
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
	require.ElementsMatch(t, expected, fileList)

	fileList, err = util.ListDirRecursive(tmpDir, false, util.IgnoreHiddenFiles, IgnoreDir3, util.IgnorePythonGeneratedFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, fileList)

	fileList, err = util.ListDirRecursive(tmpDir, false, util.IgnoreNonPython)
	expected = []string{
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
	}
	require.NoError(t, err)
	util.Ppj(fileList)
	require.ElementsMatch(t, expected, fileList)

	fileList, err = util.ListDirRecursive(tmpDir, true, util.IgnoreNonPython)
	expected = []string{
		filepath.Join("2.py"),
		filepath.Join("3/1.py"),
		filepath.Join("3/2/1.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, fileList)
}
