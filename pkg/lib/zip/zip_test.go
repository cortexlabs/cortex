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

package zip_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
)

func TestZip(t *testing.T) {
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

	err = files.MakeEmptyFiles(filesList...)
	require.NoError(t, err)

	var zipInput *zip.Input
	var expected []string

	zipInput = &zip.Input{
		Bytes: []zip.BytesInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Files: []zip.FileInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Files: []zip.FileInput{
			{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1.txt",
			},
			{
				Source: filepath.Join(tmpDir, "test.txt"),
				Dest:   "test.txt",
			},
		},
		AllowMissing: true,
	}
	CheckZip(zipInput, []string{"1.txt"}, false, t)
	zipInput.AllowMissing = false
	CheckZip(zipInput, nil, true, t)

	zipInput = &zip.Input{
		Bytes: []zip.BytesInput{
			{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []zip.FileInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Bytes: []zip.BytesInput{
			{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []zip.FileInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		EmptyFiles: []string{
			"text.txt",
			"1/2/3.txt",
		},
	}
	expected = []string{
		"text.txt",
		"1/2/3.txt",
	}
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
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
	CheckZip(zipInput, expected, false, t)

	zipInput = &zip.Input{
		Bytes: []zip.BytesInput{
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
	CheckZip(zipInput, nil, true, t)

	zipInput = &zip.Input{
		Bytes: []zip.BytesInput{
			{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
		},
		EmptyFiles: []string{"1/text.txt"},
	}
	CheckZip(zipInput, nil, true, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
			{
				Source:  filepath.Join(tmpDir, "3"),
				Flatten: true,
			},
		},
	}
	CheckZip(zipInput, nil, true, t)

	zipInput = &zip.Input{
		Dirs: []zip.DirInput{
			{
				Source:  filepath.Join(tmpDir, "5"),
				Flatten: true,
			},
		},
	}
	CheckZip(zipInput, nil, true, t)
}

func CheckZip(zipInput *zip.Input, expected []string, shouldErr bool, t *testing.T) {
	tmpDir, err := files.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	err = zip.ToFile(zipInput, filepath.Join(tmpDir, "zip.zip"))
	if shouldErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)

	_, err = zip.UnzipToFile(filepath.Join(tmpDir, "zip.zip"), filepath.Join(tmpDir, "zip"))
	require.NoError(t, err)

	unzippedFiles, err := files.ListDirRecursive(filepath.Join(tmpDir, "zip"), true)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, unzippedFiles)

	contents, err := zip.UnzipFileToMem(filepath.Join(tmpDir, "zip.zip"))
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))

	zipBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "zip.zip"))
	require.NoError(t, err)
	contents, err = zip.UnzipMemToMem(zipBytes)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, maps.InterfaceMapKeysUnsafe(contents))
}
