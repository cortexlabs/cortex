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
	emptyExcludes := make([]string, 0)

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
	}

	excludes := []string{
		filepath.Join(tmpDir, "*.txt"),
		filepath.Join(tmpDir, "*/1.py"),
		filepath.Join(tmpDir, "3/*/1.py"),
		filepath.Join(tmpDir, "3/2/*/.tmp"),
		filepath.Join(tmpDir, "4/*.yaml"),
	}

	excludesWithRelativePaths := []string{
		filepath.Join("*.txt"),
		filepath.Join("*/1.py"),
		filepath.Join("3/*/1.py"),
		filepath.Join("3/2/*/.tmp"),
		filepath.Join("4/*.yaml"),
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
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)

	filesListRecursive, err = ListDirRecursive(tmpDir, false, emptyExcludes, IgnoreHiddenFiles, IgnoreDir3, IgnorePythonGeneratedFiles)
	expected = []string{
		filepath.Join(tmpDir, "1.txt"),
		filepath.Join(tmpDir, "2.py"),
		filepath.Join(tmpDir, "4/1.yaml"),
		filepath.Join(tmpDir, "4/.git/HEAD"),
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

	filesListRecursive, err = ListDirRecursive(tmpDir, true, excludesWithRelativePaths, IgnoreNonPython)
	expected = []string{
		filepath.Join("2.py"),
	}
	require.NoError(t, err)
	require.ElementsMatch(t, expected, filesListRecursive)
}

func TestReadAllIgnorePatterns(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "cortexignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	di, err := ReadAllIgnorePatterns(nil)
	if err != nil {
		t.Fatalf("Expected not to have error, got %v", err)
	}

	if diLen := len(di); diLen != 0 {
		t.Fatalf("Expected to have zero dockerignore entry, got %d", diLen)
	}

	diName := filepath.Join(tmpDir, ".dockerignore")
	content := fmt.Sprintf("test1\n/test2\n/a/file/here\n\nlastfile\n# this is a comment\n! /inverted/abs/path\n!\n! \n")
	err = ioutil.WriteFile(diName, []byte(content), 0777)
	if err != nil {
		t.Fatal(err)
	}

	diFd, err := os.Open(diName)
	if err != nil {
		t.Fatal(err)
	}
	defer diFd.Close()

	di, err = ReadAllIgnorePatterns(diFd)
	if err != nil {
		t.Fatal(err)
	}

	if len(di) != 7 {
		t.Fatalf("Expected 5 entries, got %v", len(di))
	}
	if di[0] != "test1" {
		t.Fatal("First element is not test1")
	}
	if di[1] != "test2" {
		t.Fatal("Second element is not test2")
	}
	if di[2] != "a/file/here" {
		t.Fatal("Third element is not a/file/here")
	}
	if di[3] != "lastfile" {
		t.Fatal("Fourth element is not lastfile")
	}
	if di[4] != "!inverted/abs/path" {
		t.Fatal("Fifth element is not !inverted/abs/path")
	}
	if di[5] != "!" {
		t.Fatalf("Sixth element is not !, but %s", di[5])
	}
	if di[6] != "!" {
		t.Fatalf("Sixth element is not !, but %s", di[6])
	}
}
