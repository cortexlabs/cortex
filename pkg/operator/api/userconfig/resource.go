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

package userconfig

import (
	"fmt"
	"strings"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Resource interface {
	GetName() string
	GetResourceType() resource.Type
	GetIndex() int
	SetIndex(int)
	GetFilePath() string
	SetFilePath(string)
}

type ResourceFields struct {
	Name     string `json:"name" yaml:"name"`
	Index    int    `json:"index" yaml:"-"`
	FilePath string `json:"file_path" yaml:"-"`
}

func (resourceFields *ResourceFields) GetName() string {
	return resourceFields.Name
}

func (resourceFields *ResourceFields) GetIndex() int {
	return resourceFields.Index
}

func (resourceFields *ResourceFields) SetIndex(index int) {
	resourceFields.Index = index
}

func (resourceFields *ResourceFields) GetFilePath() string {
	return resourceFields.FilePath
}

func (resourceFields *ResourceFields) SetFilePath(filePath string) {
	resourceFields.FilePath = filePath
}

func (resourceFields *ResourceFields) UserConfigStr() string {
	var sb strings.Builder
	if resourceFields.FilePath == "" {
		sb.WriteString("file: <none>\n")
	} else {
		sb.WriteString(fmt.Sprintf("file: %s\n", resourceFields.FilePath))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, resourceFields.Name))
	return sb.String()
}

func Identify(r Resource) string {
	return identify(r.GetFilePath(), r.GetResourceType(), r.GetName(), r.GetIndex())
}

func identify(filePath string, resourceType resource.Type, name string, index int) string {
	resourceTypeStr := resourceType.String()
	if resourceType == resource.UnknownType {
		resourceTypeStr = "resource"
	}

	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if name != "" {
		return str + resourceTypeStr + ": " + name
	} else if index >= 0 {
		return str + resourceTypeStr + " at " + s.Index(index)
	}
	return str + resourceTypeStr
}

func FindDuplicateResourceName(resources ...Resource) []Resource {
	names := make(map[string][]Resource)
	for _, r := range resources {
		names[r.GetName()] = append(names[r.GetName()], r)
	}

	for name := range names {
		if len(names[name]) > 1 {
			return names[name]
		}
	}

	return nil
}
