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

package files

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/denormal/go-gitignore"
	"github.com/mitchellh/go-homedir"
	"github.com/shirou/gopsutil/mem"
	"github.com/xlab/treeprint"
)

var (
	_homeDir string
)

// the returned file should be closed by the caller
func Open(path string) (*os.File, error) {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadFile(path)))
	}

	return file, nil
}

// the returned file should be closed by the caller
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(cleanPath, flag, perm)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return file, err
}

// the returned file should be closed by the caller
func Create(path string) (*os.File, error) {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Create(cleanPath)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return file, nil
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
	path, err := EscapeTilde(path)
	if err != nil {
		return nil, err
	}

	if err := CheckFileErrPath(path, errMsgPath); err != nil {
		return nil, err
	}

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadFile(errMsgPath)))
	}

	return fileBytes, nil
}

func CreateFile(path string) error {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	file, err := os.Create(cleanPath)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}
	defer file.Close()

	return nil
}

func WriteFileFromReader(reader io.Reader, path string) error {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	file, err := os.Create(cleanPath)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return nil
}

func WriteFile(data []byte, path string) error {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(cleanPath, data, 0664); err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateFile(path)))
	}

	return nil
}

func IsAbsOrTildePrefixed(path string) bool {
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~/")
}

// e.g. ~/path -> /home/ubuntu/path
// returns original path if there was an error
func EscapeTilde(path string) (string, error) {
	if !(path == "~" || strings.HasPrefix(path, "~/")) {
		return path, nil
	}

	if _homeDir == "" {
		homeDir, err := homedir.Dir()
		if err != nil {
			return path, err
		}
		if homeDir == "" || homeDir == "/" {
			return path, nil
		}
		_homeDir = homeDir
	}

	if path == "~" {
		return _homeDir, nil
	}

	// path starts with "~/"
	return filepath.Join(_homeDir, path[2:]), nil
}

// e.g. ~/path/../path2 -> /home/ubuntu/path2
// returns without escaping tilde if there was an error
func Clean(path string) (string, error) {
	path, err := EscapeTilde(path)
	path = filepath.Clean(path)
	if err != nil {
		return path, err
	}
	return path, nil
}

// e.g. /home/ubuntu/path -> ~/path
func ReplacePathWithTilde(absPath string) string {
	if !strings.HasPrefix(absPath, "/") {
		return absPath
	}

	if _homeDir == "" {
		homeDir, err := homedir.Dir()
		if err != nil || homeDir == "" || homeDir == "/" {
			return absPath
		}
		_homeDir = homeDir
	}

	trimmedHomeDir := strings.TrimSuffix(s.EnsurePrefix(_homeDir, "/"), "/")

	if strings.Index(absPath, trimmedHomeDir) == 0 {
		return "~" + absPath[len(trimmedHomeDir):]
	}

	return absPath
}

func TrimDirPrefix(fullPath string, dirPath string) string {
	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}
	return strings.TrimPrefix(fullPath, dirPath)
}

func RelToAbsPath(relativePath string, baseDir string) string {
	if !IsAbsOrTildePrefixed(relativePath) {
		relativePath = filepath.Join(baseDir, relativePath)
	}
	cleanPath, _ := Clean(relativePath)
	return cleanPath
}

func UserRelToAbsPath(relativePath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return relativePath
	}
	return RelToAbsPath(relativePath, cwd)
}

func PathRelativeToCWD(absPath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return absPath
	}
	return PathRelativeToDir(absPath, cwd)
}

func PathRelativeToDir(absPath string, dir string) string {
	if !IsAbsOrTildePrefixed(absPath) {
		return absPath
	}
	absPath, _ = EscapeTilde(absPath)
	dir, _ = EscapeTilde(dir)
	dir = s.EnsureSuffix(dir, "/")
	return strings.TrimPrefix(absPath, dir)
}

func DirPathRelativeToCWD(absPath string) string {
	return s.EnsureSuffix(PathRelativeToCWD(absPath), "/")
}

func DirPathRelativeToDir(absPath string, dir string) string {
	return s.EnsureSuffix(PathRelativeToDir(absPath, dir), "/")
}

func IsFileOrDir(path string) bool {
	path, _ = EscapeTilde(path)
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
	dirPath, err := EscapeTilde(dirPath)
	if err != nil {
		return err
	}

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
	path, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return ErrorFileDoesNotExist(errMsgPath)
	}
	if fileInfo.IsDir() {
		return ErrorNotAFile(errMsgPath)
	}

	return nil
}

func CreateDir(path string) error {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cleanPath, os.ModePerm); err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateDir(path)))
	}

	return nil
}

func CreateDirIfMissing(path string) (bool, error) {
	if IsDir(path) {
		return false, nil
	}

	if IsFile(path) {
		return false, ErrorFileAlreadyExists(path)
	}

	err := CreateDir(path)
	if err != nil {
		return false, err
	}

	return true, nil
}

func DeleteDir(path string) error {
	cleanPath, err := EscapeTilde(path)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(cleanPath); err != nil {
		return errors.Wrap(err, errors.Message(ErrorDeleteDir(path)))
	}

	return nil
}

func DeleteDirIfPresent(path string) (bool, error) {
	if IsFile(path) {
		return false, ErrorNotADir(path)
	}

	if !IsDir(path) {
		return false, nil
	}

	err := DeleteDir(path)
	if err != nil {
		return false, err
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
	dir, err := Clean(dir)
	if err != nil {
		return "", err
	}

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
	cleanPath, err := Clean(path)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(cleanPath), os.ModePerm)
	if err != nil {
		return errors.Wrap(err, errors.Message(ErrorCreateDir(filepath.Dir(path))))
	}
	f, err := os.OpenFile(cleanPath, os.O_RDONLY|os.O_CREATE, 0666)
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

// IgnoreFn if passed a dir, returning true will ignore all subdirs of dir
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
	if fi.IsDir() && fi.Name() == "__pycache__" {
		return true, nil
	}
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

func GitIgnoreFn(gitIgnorePath string) (IgnoreFn, error) {
	gitIgnoreDir := filepath.Dir(gitIgnorePath)

	ignore, err := gitignore.NewFromFile(gitIgnorePath)
	if err != nil {
		return nil, errors.Wrap(err, gitIgnorePath)
	}

	return func(path string, fi os.FileInfo) (bool, error) {
		if path == gitIgnoreDir {
			// This is to avoid a bug in ignore.Ignore()
			return false, nil
		}
		return ignore.Ignore(path), nil
	}, nil
}

// promptMsgTemplate should have two placeholders: the first is for the file path and the second is for the file size
func PromptForFilesAboveSize(size int, promptMsgTemplate string) IgnoreFn {
	if promptMsgTemplate == "" {
		promptMsgTemplate = "do you want to zip %s (%s)?"
	}

	return func(path string, fi os.FileInfo) (bool, error) {
		if !fi.IsDir() && fi.Size() > int64(size) {
			promptMsg := fmt.Sprintf(promptMsgTemplate, PathRelativeToCWD(path), s.Int64ToBase2Byte(fi.Size()))
			return !prompt.YesOrNo(promptMsg, "", ""), nil
		}
		return false, nil
	}
}

func ErrorOnBigFilesFn(maxFileSizeBytes int64) IgnoreFn {
	return func(path string, fi os.FileInfo) (bool, error) {
		if !fi.IsDir() {
			fileSizeBytes := fi.Size()
			virtual, _ := mem.VirtualMemory()
			if fileSizeBytes > int64(virtual.Available) {
				return false, errors.Wrap(
					ErrorInsufficientMemoryToReadFile(fileSizeBytes, int64(virtual.Available)),
					path,
				)
			}
			if fileSizeBytes > maxFileSizeBytes {
				return false, errors.Wrap(ErrorFileSizeLimit(maxFileSizeBytes), path)
			}
		}

		return false, nil
	}
}

func ErrorOnProjectSizeLimit(maxProjectSizeBytes int64) IgnoreFn {
	filesSizeSum := int64(0)
	return func(path string, fi os.FileInfo) (bool, error) {
		if !fi.IsDir() {
			filesSizeSum += fi.Size()
			if filesSizeSum > maxProjectSizeBytes {
				return false, errors.Wrap(ErrorProjectSizeLimit(maxProjectSizeBytes), path)
			}
		}
		return false, nil
	}
}

// Retrieves the longest common path given a list of paths.
func LongestCommonPath(paths ...string) string {

	// Handle special cases.
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return path.Clean(paths[0])
	}

	startsWithSlash := false
	allStartWithSlash := true

	var splitPaths [][]string
	shortestPathLength := -1
	for _, path := range paths {
		if strings.HasPrefix(path, "/") {
			startsWithSlash = true
		} else {
			allStartWithSlash = false
		}

		splitPath := slices.RemoveEmpties(strings.Split(path, "/"))
		splitPaths = append(splitPaths, splitPath)

		if len(splitPath) < shortestPathLength || shortestPathLength == -1 {
			shortestPathLength = len(splitPath)
		}
	}

	commonPath := ""
	numPaths := len(splitPaths)

	for level := 0; level < shortestPathLength; level++ {
		element := splitPaths[0][level]
		counter := 1
		for _, splitPath := range splitPaths[1:] {
			if splitPath[level] != element {
				break
			}
			counter++
		}

		if counter != numPaths {
			break
		}

		commonPath = filepath.Join(commonPath, element)
	}
	if commonPath != "" && startsWithSlash {
		commonPath = s.EnsurePrefix(commonPath, "/")
		commonPath = s.EnsureSuffix(commonPath, "/")
	}
	if commonPath == "" && allStartWithSlash {
		return "/"
	}

	return commonPath
}

func FilterPathsWithDirPrefix(paths []string, prefix string) []string {
	prefix = s.EnsureSuffix(prefix, "/")

	var filteredPaths []string
	for _, path := range paths {
		if strings.HasPrefix(path, prefix) {
			filteredPaths = append(filteredPaths, path)
		}
	}

	return filteredPaths
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

	commonPrefix := LongestCommonPath(dirPaths...)
	paths, _ = s.TrimPrefixIfPresentInAll(paths, commonPrefix)

	var header string

	if didTrimCwd && commonPrefix == "" {
		header = ".\n"
	} else if !didTrimCwd && commonPrefix == "" {
		header = ""
	} else if didTrimCwd && commonPrefix != "" {
		header = "./" + commonPrefix
		header = s.EnsureSingleOccurrenceCharSuffix(header, "/") + "\n"
	} else if !didTrimCwd && commonPrefix != "" {
		header = commonPrefix + "/"
		header = s.EnsureSingleOccurrenceCharSuffix(header, "/") + "\n"
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

// Return the path to the directory containing the provided path (with a trailing slash)
func Dir(path string) string {
	return s.EnsureSuffix(filepath.Dir(path), "/")
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

func ListDirRecursive(dir string, relative bool, ignoreFns ...IgnoreFn) ([]string, error) {
	cleanDir, err := Clean(dir)
	if err != nil {
		return nil, err
	}
	cleanDir = strings.TrimSuffix(cleanDir, "/")

	if err := CheckDir(cleanDir); err != nil {
		return nil, err
	}

	var fileList []string
	walkErr := filepath.Walk(cleanDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, path)
		}

		for _, ignoreFn := range ignoreFns {
			ignore, err := ignoreFn(path, fi)
			if err != nil {
				return err
			}
			if ignore {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !fi.IsDir() {
			if relative && dir != "." {
				path = path[len(cleanDir)+1:]
			}
			fileList = append(fileList, path)
		}
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return fileList, nil
}

func ListDir(dir string, relative bool) ([]string, error) {
	cleanDir, err := Clean(dir)
	if err != nil {
		return nil, err
	}
	cleanDir = strings.TrimSuffix(cleanDir, "/")

	if err := CheckDir(cleanDir); err != nil {
		return nil, err
	}

	var filenames []string
	fileInfo, err := ioutil.ReadDir(cleanDir)
	if err != nil {
		return nil, errors.Wrap(err, errors.Message(ErrorReadDir(dir)))
	}
	for _, file := range fileInfo {
		filename := file.Name()
		if !relative {
			filename = filepath.Join(cleanDir, filename)
		}
		filenames = append(filenames, filename)
	}
	return filenames, nil
}

func CopyFileOverwrite(src string, dest string) error {
	srcFile, err := Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, srcFile); err != nil {
		return err
	}

	return nil
}

func CopyDirOverwrite(src string, dest string, ignoreFns ...IgnoreFn) error {
	srcRelFilePaths, err := ListDirRecursive(src, true, ignoreFns...)
	if err != nil {
		return err
	}

	if _, err := CreateDirIfMissing(dest); err != nil {
		return err
	}
	createdDirs := strset.New(dest)

	for _, srcRelFilePath := range srcRelFilePaths {
		srcFilePath := filepath.Join(src, srcRelFilePath)
		destFilePath := filepath.Join(dest, srcRelFilePath)

		destFileDir := filepath.Dir(destFilePath)
		if !createdDirs.Has(destFileDir) {
			if _, err := CreateDirIfMissing(destFileDir); err != nil {
				return err
			}
			createdDirs.Add(destFileDir)
		}

		if err := CopyFileOverwrite(srcFilePath, destFilePath); err != nil {
			return err
		}
	}

	return nil
}

func CopyRecursiveShell(src string, dest string) error {
	cleanSrc, err := EscapeTilde(src)
	if err != nil {
		return err
	}
	cleanDest, err := EscapeTilde(dest)
	if err != nil {
		return err
	}

	cmd := exec.Command("cp", "-r", cleanSrc, cleanDest)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, strings.TrimSpace(out.String()))
	}

	return nil
}

func HashFile(path string, paths ...string) (string, error) {
	md5Hash := md5.New()

	allPaths := append(paths, path)
	for _, path := range allPaths {
		f, err := Open(path)
		if err != nil {
			return "", err
		}

		// Skip directories
		fileInfo, err := f.Stat()
		if err != nil {
			f.Close()
			return "", errors.WithStack(err)
		}
		if fileInfo.IsDir() {
			f.Close()
			continue
		}

		if _, err := io.Copy(md5Hash, f); err != nil {
			f.Close()
			return "", errors.Wrap(err, path)
		}

		io.WriteString(md5Hash, path)
		f.Close()
	}

	return hex.EncodeToString((md5Hash.Sum(nil))), nil
}

func HashDirectory(dir string, ignoreFns ...IgnoreFn) (string, error) {
	md5Hash := md5.New()

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
		if fi.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, path)
		}
		defer f.Close()

		if _, err := io.Copy(md5Hash, f); err != nil {
			return errors.Wrap(err, path)
		}

		io.WriteString(md5Hash, path)
		return nil
	})

	if walkErr != nil {
		return "", walkErr
	}

	return hex.EncodeToString((md5Hash.Sum(nil))), nil
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
