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

package spec

type ProjectFiles interface {
	// Return all paths in the project, relative to the project root containing cortex.yaml
	AllPaths() []string
	// Return the contents of a file, given the path (relative to the project root)
	GetFile(string) ([]byte, error)
	// Return whether the project contains a file path (relative to the project root)
	HasFile(string) bool
	// Return whether the project contains a directory path (relative to the project root)
	HasDir(string) bool
}
