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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/xlab/treeprint"
)

func Open(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadFile(path)))
	}

	return file, nil
}

func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return file, err
}

func ReadFile(path string) (string, error) {
	fileBytes, err := ReadFileBytes(path)
	if err != nil {
		return "", err
	}

	return string(fileBytes), nil
}

func ReadFileBytes(path string) ([]byte, error) {
	return ReadFileBytesErrPath(path, path)
}

func ReadFileBytesErrPath(path string, errMsgPath string) ([]byte, error) {
	if err := CheckFileErrPath(path, errMsgPath); err != nil {
		return nil, err
	}

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadFile(errMsgPath)))
	}

	return fileBytes, nil
}

func CreateFile(path string) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return file, nil
}

func WriteFile(data []byte, path string) error {
	if err := ioutil.WriteFile(path, data, 0664); err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return nil
}

func MkdirAll(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateDir(path)))
	}

	return nil
}

func TrimDirPrefix(fullPath string, dirPath string) string {
	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}
	return strings.TrimPrefix(fullPath, dirPath)
}

func RelToAbsPath(relativePath string, baseDir string) string {
	if !filepath.IsAbs(relativePath) {
		relativePath = filepath.Join(baseDir, relativePath)
	}
	return filepath.Clean(relativePath)
}

func UserRelToAbsPath(relativePath string) string {
	cwd, _ := os.Getwd()
	return RelToAbsPath(relativePath, cwd)
}

func PathRelativeToCWD(absPath string) string {
	cwd, _ := os.Getwd()
	cwd = s.EnsureSuffix(cwd, "/")
	return strings.TrimPrefix(absPath, cwd)
}

func DirPathRelativeToCWD(absPath string) string {
	return s.EnsureSuffix(PathRelativeToCWD(absPath), "/")
}

func IsFileOrDir(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func IsDir(path string) bool {
	if err := CheckDir(path); err != nil {
		return false
	}
	return true
}

// CheckDir returns nil if the path is a directory
func CheckDir(dirPath string) error {
	return CheckDirErrPath(dirPath, dirPath)
}

// CheckDir returns nil if the path is a directory
func CheckDirErrPath(dirPath string, errMsgPath string) error {
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorDirDoesNotExist(errMsgPath)))
	}
	if !fileInfo.IsDir() {
		return ErrorNotADir(errMsgPath)
	}

	return nil
}

func IsFile(path string) bool {
	if err := CheckFile(path); err != nil {
		return false
	}
	return true
}

// CheckFile returns nil if the path is a file
func CheckFile(path string) error {
	return CheckFileErrPath(path, path)
}

// CheckFile returns nil if the path is a file
func CheckFileErrPath(path string, errMsgPath string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return ErrorFileDoesNotExist(errMsgPath)
	}
	if fileInfo.IsDir() {
		return ErrorNotAFile(errMsgPath)
	}

	return nil
}

func CreateDirIfMissing(path string) (bool, error) {
	if err := CheckDir(path); err == nil {
		return false, nil
	}

	if err := CheckFile(path); err == nil {
		return false, ErrorFileAlreadyExists(path)
	}

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return false, errors.Wrap(err, path)
	}

	return true, nil
}

func TmpDir() (string, error) {
	filePath, err := ioutil.TempDir("", "")
	if err != nil {
		return "", errors.Wrap(err)
	}
	return filePath, nil
}

func ParentDir(dir string) string {
	return filepath.Clean(filepath.Join(dir, ".."))
}

func SearchForFile(filename string, dir string) (string, error) {
	dir = filepath.Clean(dir)
	for true {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return "", errors.Wrap(err, errors.Message(ErrorReadDir(dir)))
		}

		for _, file := range files {
			if file.Name() == filename {
				return filepath.Join(dir, filename), nil
			}
		}

		if dir == "/" {
			return "", nil
		}

		dir = ParentDir(dir)
	}

	return "", ErrorUnexpected()
}

func MakeEmptyFile(path string) error {
	path = filepath.Clean(path)
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateDir(filepath.Dir(path))))
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}
	defer f.Close()
	return nil
}

func MakeEmptyFiles(path string, paths ...string) error {
	allPaths := append(paths, path)
	for _, path := range allPaths {
		if err := MakeEmptyFile(path); err != nil {
			return err
		}
	}
	return nil
}

func MakeEmptyFilesInDir(dir string, path string, paths ...string) error {
	allPaths := append(paths, path)
	for _, path := range allPaths {
		fullPath := filepath.Join(dir, path)
		if err := MakeEmptyFile(fullPath); err != nil {
			return err
		}
	}
	return nil
}

func IsFilePathYAML(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}

func IsFilePathPython(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".py"
}

// IgnoreFn if passed a dir, returning false will ignore all subdirs of dir
type IgnoreFn func(string, os.FileInfo) (bool, error)

func IgnoreHiddenFiles(path string, fi os.FileInfo) (bool, error) {
	if !fi.IsDir() && strings.HasPrefix(fi.Name(), ".") {
		return true, nil
	}
	return false, nil
}

func IgnoreCortexYAML(path string, fi os.FileInfo) (bool, error) {
	if !fi.IsDir() && fi.Name() == "cortex.yaml" {
		return true, nil
	}
	return false, nil
}

func IgnoreCortexDebug(path string, fi os.FileInfo) (bool, error) {
	if strings.HasPrefix(fi.Name(), "cortex-debug-") {
		return true, nil
	}
	return false, nil
}

func IgnoreHiddenFolders(path string, fi os.FileInfo) (bool, error) {
	if fi.IsDir() && strings.HasPrefix(fi.Name(), ".") {
		return true, nil
	}
	return false, nil
}

func IgnorePythonGeneratedFiles(path string, fi os.FileInfo) (bool, error) {
	if !fi.IsDir() {
		ext := filepath.Ext(path)
		return ext == ".pyc" || ext == ".pyo" || ext == ".pyd", nil
	}
	return false, nil
}

func IgnoreNonPython(path string, fi os.FileInfo) (bool, error) {
	if !fi.IsDir() && !IsFilePathPython(path) {
		return true, nil
	}
	return false, nil
}

func IgnoreNonYAML(path string, fi os.FileInfo) (bool, error) {
	if !fi.IsDir() && !IsFilePathYAML(path) {
		return true, nil
	}
	return false, nil
}

func IgnoreSpecificFiles(absPaths ...string) IgnoreFn {
	absPathsSet := strset.New(absPaths...)
	return func(path string, fi os.FileInfo) (bool, error) {
		return absPathsSet.Has(path), nil
	}
}

// promptMsgTemplate should have two placeholders: the first is for the file path and the second is for the file size
func PromptForFilesAboveSize(size int, promptMsgTemplate string) IgnoreFn {
	if promptMsgTemplate == "" {
		promptMsgTemplate = "do you want to zip %s (%s)?"
	}

	return func(path string, fi os.FileInfo) (bool, error) {
		if !fi.IsDir() && fi.Size() > int64(size) {
			promptMsg := fmt.Sprintf(promptMsgTemplate, PathRelativeToCWD(path), s.IntToBase2Byte(int(fi.Size())))
			return !prompt.YesOrNo(promptMsg, "", ""), nil
		}
		return false, nil
	}
}

type DirsOrder string

var DirsSorted DirsOrder = "sorted"
var DirsOnTop DirsOrder = "top"
var DirsOnBottom DirsOrder = "bottom"

func SortFilePaths(paths []string, dirsOrder DirsOrder) []string {
	if dirsOrder == DirsSorted {
		sort.Strings(paths)
		return paths
	}

	dirsSortChar := ""
	if dirsOrder == DirsOnTop {
		dirsSortChar = " "
	}
	if dirsOrder == DirsOnBottom {
		dirsSortChar = "|"
	}

	for i, path := range paths {
		dirPath := filepath.Dir(path)
		if dirPath == "." || dirPath == "/" {
			continue
		}
		replacedDir := strings.Replace(dirPath, "/", "/"+dirsSortChar, -1)
		paths[i] = dirsSortChar + replacedDir + "/" + filepath.Base(path)
	}
	sort.Strings(paths)
	for i, path := range paths {
		paths[i] = strings.Replace(path, dirsSortChar, "", -1)
	}

	return paths
}

func FileTree(paths []string, cwd string, dirsOrder DirsOrder) string {
	if len(paths) == 0 {
		return "."
	}

	paths = SortFilePaths(paths, dirsOrder)
	dirPaths := DirPaths(paths, true)

	didTrimCwd := false
	if cwd != "" {
		cwd = s.EnsureSuffix(cwd, "/")
		paths, didTrimCwd = s.TrimPrefixIfPresentInAll(paths, cwd)
		dirPaths = DirPaths(paths, true)
	}

	commonPrefix := s.LongestCommonPrefix(dirPaths...)
	paths, _ = s.TrimPrefixIfPresentInAll(paths, commonPrefix)

	var header string

	if didTrimCwd && commonPrefix == "" {
		header = ".\n"
	} else if !didTrimCwd && commonPrefix == "" {
		header = ""
	} else if didTrimCwd && commonPrefix != "" {
		header = "./" + commonPrefix + "\n"
	} else if !didTrimCwd && commonPrefix != "" {
		header = commonPrefix + "\n"
	}

	tree := treeprint.New()

	cachedTrees := make(map[string]treeprint.Tree)
	cachedTrees["."] = tree
	cachedTrees["/"] = tree
	cachedTrees[""] = tree

	for _, path := range paths {
		dir := filepath.Dir(path)
		branch := getTreeBranch(dir, cachedTrees)
		branch.AddNode(filepath.Base(path))
	}

	treeStr := tree.String()
	return header + treeStr[2:]
}

func getTreeBranch(dir string, cachedTrees map[string]treeprint.Tree) treeprint.Tree {
	dir = s.TrimPrefixAndSuffix(dir, "/")
	if cachedTree, ok := cachedTrees[dir]; ok {
		return cachedTree
	}

	var parentDir, lastDir string

	lastIndex := strings.LastIndex(dir, "/")
	if lastIndex == -1 {
		parentDir = "."
		lastDir = dir
	} else {
		parentDir = s.TrimPrefixAndSuffix(dir[:lastIndex], "/")
		lastDir = s.TrimPrefixAndSuffix(dir[lastIndex:], "/")
	}

	parentBranch := getTreeBranch(parentDir, cachedTrees)
	branch := parentBranch.AddBranch(lastDir)
	cachedTrees[dir] = branch
	return branch
}

func DirPaths(paths []string, addTrailingSlash bool) []string {
	suffix := ""
	if addTrailingSlash {
		suffix = "/"
	}
	dirPaths := make([]string, len(paths))
	for i, path := range paths {
		dirPaths[i] = s.EnsureSuffix(filepath.Dir(path), suffix)
	}
	return dirPaths
}

func ListDirRecursive(dir string, relative bool, excludes []string, ignoreFns ...IgnoreFn) ([]string, error) {
	dir = filepath.Clean(dir)

	var fileList []string
	walkErr := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, path)
		}

		for _, ignoreFn := range ignoreFns {
			ignore, err := ignoreFn(path, fi)
			if err != nil {
				return errors.Wrap(err, path)
			}
			if ignore {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !fi.IsDir() {
			if strings.Contains(path, ".cortexignore") {
				return nil
			}

			relPath := path[len(dir)+1:]

			if len(excludes) > 0 {
				for _, exclude := range excludes {
					matchAbs, err := filepath.Match(exclude, path)
					if err != nil {
						// The only possible returned error is ErrBadPattern, when pattern is malformed
						return errors.Wrap(err, relPath)
					}
					if matchAbs {
						return nil
					}

					matchRel, err := filepath.Match(exclude, relPath)
					if err != nil {
						// The only possible returned error is ErrBadPattern, when pattern is malformed
						return errors.Wrap(err, relPath)
					}
					if matchRel {
						return nil
					}
				}
			}

			if relative {
				fileList = append(fileList, relPath)
			} else {
				fileList = append(fileList, path)
			}
		}

		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return fileList, nil
}

func ListDir(dir string, relative bool) ([]string, error) {
	dir = filepath.Clean(dir)
	var filenames []string
	fileInfo, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadDir(dir)))
	}
	for _, file := range fileInfo {
		filename := file.Name()
		if !relative {
			filename = filepath.Join(dir, filename)
		}
		filenames = append(filenames, filename)
	}
	return filenames, nil
}

func CloseSilent(closer io.Closer, closers ...io.Closer) {
	allClosers := append(closers, closer)
	for _, closer := range allClosers {
		closer.Close()
	}
}

// ReadReqFile returns nil if no file
func ReadReqFile(r *http.Request, fileName string) ([]byte, error) {
	mpFile, _, err := r.FormFile(fileName)
	if err != nil {
		if strings.Contains(errors.Message(err), "no such file") {
			return nil, nil
		}
		return nil, errors.Wrap(err, errors.Message(ErrorReadFormFile(fileName)))
	}
	defer mpFile.Close()
	fileBytes, err := ioutil.ReadAll(mpFile)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadFormFile(fileName)))
	}
	return fileBytes, nil
}

// ReadAll reads a .cortexignore file and returns the list of file patterns
// to ignore. Note this will trim whitespace from each line as well
// as use GO's "clean" func to get the shortest/cleanest path for each.
func ReadAllIgnorePatterns(reader io.Reader) ([]string, error) {
	if reader == nil {
		return nil, nil
	}

	scanner := bufio.NewScanner(reader)
	var excludes []string
	currentLine := 0

	utf8bom := []byte{0xEF, 0xBB, 0xBF}
	for scanner.Scan() {
		scannedBytes := scanner.Bytes()
		// We trim UTF8 BOM
		if currentLine == 0 {
			scannedBytes = bytes.TrimPrefix(scannedBytes, utf8bom)
		}
		pattern := string(scannedBytes)
		currentLine++
		// Lines starting with # (comments) are ignored before processing
		if strings.HasPrefix(pattern, "#") {
			continue
		}
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		// normalize absolute paths to paths relative to the context
		// (taking care of '!' prefix)
		invert := pattern[0] == '!'
		if invert {
			pattern = strings.TrimSpace(pattern[1:])
		}
		if len(pattern) > 0 {
			pattern = filepath.Clean(pattern)
			pattern = filepath.ToSlash(pattern)
			if len(pattern) > 1 && pattern[0] == '/' {
				pattern = pattern[1:]
			}
		}
		if invert {
			pattern = "!" + pattern
		}

		excludes = append(excludes, pattern)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error reading .cortexignore")
	}
	return excludes, nil
}
