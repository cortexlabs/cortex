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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func TestZip(t *testing.T) {
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
		filepath.Join(tmpDir, "5/4/3/2/1.txt"),
		filepath.Join(tmpDir, "5/4/3/2/1.py"),
		filepath.Join(tmpDir, "5/4/3/2/2/1.py"),
	}

	err = util.MakeEmptyFiles(files...)
	require.NoError(t, err)

	var zipInput *util.ZipInput
	var expected []string

	zipInput = &util.ZipInput{
		Bytes: []util.ZipBytesInput{
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "text.txt",
			},
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "test2/text2.txt",
			},
			util.ZipBytesInput{
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

	zipInput = &util.ZipInput{
		Files: []util.ZipFileInput{
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1.txt",
			},
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "3/1.py"),
				Dest:   "3/1.py",
			},
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "3/2/1.py"),
				Dest:   "3/2/1.py",
			},
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "3/2/3/.tmp"),
				Dest:   "3/2/3/.tmp",
			},
			util.ZipFileInput{
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

	zipInput = &util.ZipInput{
		Files: []util.ZipFileInput{
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "1.txt"),
				Dest:   "1.txt",
			},
			util.ZipFileInput{
				Source: filepath.Join(tmpDir, "test.txt"),
				Dest:   "test.txt",
			},
		},
		AllowMissing: true,
	}
	CheckZip(zipInput, []string{"1.txt"}, false, t)
	zipInput.AllowMissing = false
	CheckZip(zipInput, nil, true, t)

	zipInput = &util.ZipInput{
		Bytes: []util.ZipBytesInput{
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []util.ZipFileInput{
			util.ZipFileInput{
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

	zipInput = &util.ZipInput{
		Bytes: []util.ZipBytesInput{
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "text.txt",
			},
		},
		Files: []util.ZipFileInput{
			util.ZipFileInput{
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

	zipInput = &util.ZipInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
				Source:    tmpDir,
				Dest:      "/",
				IgnoreFns: []util.IgnoreFn{util.IgnoreHiddenFiles},
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
				Source:    filepath.Join(tmpDir, "3"),
				IgnoreFns: []util.IgnoreFn{util.IgnoreHiddenFiles},
				Dest:      "test3",
				Flatten:   true,
			},
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
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

	zipInput = &util.ZipInput{
		Bytes: []util.ZipBytesInput{
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
		},
	}
	CheckZip(zipInput, nil, true, t)

	zipInput = &util.ZipInput{
		Bytes: []util.ZipBytesInput{
			util.ZipBytesInput{
				Content: []byte(""),
				Dest:    "1/text.txt",
			},
		},
		EmptyFiles: []string{"1/text.txt"},
	}
	CheckZip(zipInput, nil, true, t)

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
				Source:  filepath.Join(tmpDir, "3"),
				Flatten: true,
			},
		},
	}
	CheckZip(zipInput, nil, true, t)

	zipInput = &util.ZipInput{
		Dirs: []util.ZipDirInput{
			util.ZipDirInput{
				Source:  filepath.Join(tmpDir, "5"),
				Flatten: true,
			},
		},
	}
	CheckZip(zipInput, nil, true, t)
}

func CheckZip(zipInput *util.ZipInput, expected []string, shouldErr bool, t *testing.T) {
	tmpDir, err := util.TmpDir()
	defer os.RemoveAll(tmpDir)
	require.NoError(t, err)

	err = util.Zip(zipInput, filepath.Join(tmpDir, "zip.zip"))
	if shouldErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)

	_, err = util.Unzip(filepath.Join(tmpDir, "zip.zip"), filepath.Join(tmpDir, "zip"))
	require.NoError(t, err)

	unzippedFiles, err := util.ListDirRecursive(filepath.Join(tmpDir, "zip"), true)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, unzippedFiles)

	contents, err := util.UnzipFileToMem(filepath.Join(tmpDir, "zip.zip"))
	require.NoError(t, err)
	require.ElementsMatch(t, expected, util.InterfaceMapKeysUnsafe(contents))

	zipBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "zip.zip"))
	require.NoError(t, err)
	contents, err = util.UnzipMemToMem(zipBytes)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, util.InterfaceMapKeysUnsafe(contents))
}
