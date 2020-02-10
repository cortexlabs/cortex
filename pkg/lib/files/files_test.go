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

package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.Equal(t, expectedHeader+expectedTree, FileTree(filesList, cwd, DirsSorted))

	cwd = "/missing"
	expectedHeader = "/1/2/"
	require.Equal(t, expectedHeader+expectedTree, FileTree(filesList, cwd, DirsSorted))

	cwd = "/1/2"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, FileTree(filesList, cwd, DirsSorted))

	cwd = "/1/2/"
	expectedHeader = "."
	require.Equal(t, expectedHeader+expectedTree, FileTree(filesList, cwd, DirsSorted))

	cwd = "/1"
	expectedHeader = "./2/"
	require.Equal(t, expectedHeader+expectedTree, FileTree(filesList, cwd, DirsSorted))

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
	require.Equal(t, "/"+expectedTree, FileTree(filesList, cwd, DirsSorted))

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
	require.Equal(t, "/"+expectedTree, FileTree(filesList, cwd, DirsOnBottom))

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
	require.Equal(t, "/"+expectedTree, FileTree(filesList, cwd, DirsOnTop))
}

func TestListDirRecursive(t *testing.T) {
	var emptyExcludes []string

	_, err := ListDirRecursive("/home/path/to/fake/dir", false, emptyExcludes)
	require.Error(t, err)

	tmpDir, err := TmpDir()
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
		filepath.Join(tmpDir, "README.md"),
	}

	excludes := []string{
		filepath.Join(tmpDir, "*.txt"),
		filepath.Join(tmpDir, "*/1.py"),
		filepath.Join(tmpDir, "3/*/1.py"),
		filepath.Join(tmpDir, "3/2/*/.tmp"),
		filepath.Join(tmpDir, "4/*.yaml"),
		filepath.Join(tmpDir, "*.md"),
	}

	excludesWithRelativePaths := []string{
		filepath.Join("*.txt"),
		filepath.Join("*/1.py"),
		filepath.Join("3/*/1.py"),
		filepath.Join("3/2/*/.tmp"),
		filepath.Join("4/*.yaml"),
		filepath.Join(tmpDir, "*.md"),
	}

	excludesWithBadPatterns := []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "[a-b-c]"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "[]a]"),
		filepath.Join(tmpDir, "*.md"),
	}

	excludesWithBadPatternsAndRelativePaths := []string{
		filepath.Join("1.txt"),
		filepath.Join("2.py"),
		filepath.Join("[a-b-c]"),
		filepath.Join("3/1.py"),
		filepath.Join("[]a]"),
	}

	err = MakeEmptyFiles(filesList[0], filesList[1:]...)
	require.NoError(t, err)

	var filesListRecursive []string
	var expected []string

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes)
	require.NoError(t, err)
	require.ElementsMatch(t, filesList, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes, IgnoreHiddenFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/2.pyc"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
		filepath.Join(tmpDir, "README.md"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes, IgnoreHiddenFiles, IgnoreHiddenFolders)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/2.pyc"),
		filepath.Join(tmpDir, "README.md"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes, IgnoreHiddenFiles, IgnoreDir3, IgnorePythonGeneratedFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
		filepath.Join(tmpDir, "README.md"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes, IgnoreNonPython)
	expected = []string{
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/1.py"),
		filepath.Join(tmpDir, "3/2/1.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, true, emptyExcludes, IgnoreNonPython)
	expected = []string{
		filepath.Join("2.py"),
		filepath.Join("3/1.py"),
		filepath.Join("3/2/1.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, excludes)
	expected = []string{
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "3/2/2.txt"),
		filepath.Join(tmpDir, "4/2.pyc"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, true, excludes)
	expected = []string{
		"2.py",
		"3/2/2.txt",
		"4/2.pyc",
		"4/.git/HEAD",
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, excludesWithRelativePaths, IgnoreNonPython)
	expected = []string{
		filepath.Join(tmpDir, "2.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, true, excludesWithRelativePaths, IgnoreNonPython)
	expected = []string{
		filepath.Join("2.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	_, err = ListDirRecursive(tmpDir, false, excludesWithBadPatterns)
	require.Error(t, err)

	_, err = ListDirRecursive(tmpDir, true, excludesWithBadPatternsAndRelativePaths)
	require.Error(t, err)
}

func TestReadAllIgnorePatterns(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "cortexignore-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	di, err := ReadAllIgnorePatterns(nil)
	require.NoError(t, err)
	require.Equalf(t, len(di), 0, "Expected to have zero cortexignore entry, got %d", len(di))

	diName := filepath.Join(tmpDir, ".cortexignore")
	content := fmt.Sprintf("test1\n/test2\n/a/file/here\n\nlastfile\n# this is a comment\n! /inverted/abs/path\n!\n! \n")
	err = ioutil.WriteFile(diName, []byte(content), 0777)
	require.NoError(t, err)

	diFd, err := os.Open(diName)
	require.NoError(t, err)
	defer diFd.Close()

	di, err = ReadAllIgnorePatterns(diFd)
	require.NoError(t, err)

	require.Equalf(t, len(di), 7, "Expected 7 entries, got %v", len(di))
	require.Equalf(t, di[0], "test1", "Expected value: test1, got %s", di[0])
	require.Equalf(t, di[1], "test2", "Expected value: test2, got %s", di[1])
	require.Equalf(t, di[2], "a/file/here", "Expected value: a/file/here, got %s", di[2])
	require.Equalf(t, di[3], "lastfile", "Expected value: lastfile, got %s", di[3])
	require.Equalf(t, di[4], "!inverted/abs/path", "Expected value: !inverted/abs/path, got %s", di[4])
	require.Equalf(t, di[5], "!", "Expected value: !, got %s", di[5])
	require.Equalf(t, di[6], "!", "Expected value: !, got %s", di[6])
}
