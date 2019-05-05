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
	GetEmbed() *Embed
	SetEmbed(*Embed)
}

type ResourceFields struct {
	Name     string `json:"name" yaml:"name"`
	Index    int    `json:"index" yaml:"-"`
	FilePath string `json:"file_path" yaml:"-"`
	Embed    *Embed `json:"embed" yaml:"-"`
}

func (ResourceFields *ResourceFields) GetName() string {
	return ResourceFields.Name
}

func (ResourceFields *ResourceFields) GetIndex() int {
	return ResourceFields.Index
}

func (ResourceFields *ResourceFields) SetIndex(index int) {
	ResourceFields.Index = index
}

func (ResourceFields *ResourceFields) GetFilePath() string {
	return ResourceFields.FilePath
}

func (ResourceFields *ResourceFields) SetFilePath(filePath string) {
	ResourceFields.FilePath = filePath
}

func (ResourceFields *ResourceFields) GetEmbed() *Embed {
	return ResourceFields.Embed
}

func (ResourceFields *ResourceFields) SetEmbed(embed *Embed) {
	ResourceFields.Embed = embed
}

func Identify(r Resource) string {
	return identify(r.GetFilePath(), r.GetResourceType(), r.GetName(), r.GetIndex(), r.GetEmbed())
}

func identify(filePath string, resourceType resource.Type, name string, index int, embed *Embed) string {
	resourceTypeStr := resourceType.String()
	if resourceType == resource.UnknownType {
		resourceTypeStr = "resource"
	}

	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if embed != nil {
		if embed.Index >= 0 {
			str += fmt.Sprintf("%s at %s (%s \"%s\"): ", resource.EmbedType.String(), s.Index(embed.Index), resource.TemplateType.String(), embed.Template)
		} else {
			str += fmt.Sprintf("%s (%s \"%s\"): ", resource.EmbedType.String(), resource.TemplateType.String(), embed.Template)
		}
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
