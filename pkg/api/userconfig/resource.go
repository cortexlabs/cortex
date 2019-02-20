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

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type Resource interface {
	GetName() string
	GetResourceType() resource.Type
	GetFilePath() string
	GetEmbed() *Embed
}

func Identify(r Resource) string {
	return identifyHelper(r.GetFilePath(), r.GetResourceType(), r.GetName(), -1, r.GetEmbed())
}

func identifyHelper(filePath string, resourceType resource.Type, name string, index int, embed *Embed) string {
	resourceTypeStr := resourceType.String()
	if resourceType == resource.UnknownType {
		resourceTypeStr = "resource"
	}

	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if embed != nil {
		str += fmt.Sprintf("%s at %s (%s \"%s\"): ", resource.EmbedType.String(), s.Index(embed.ConfigIndex), resource.TemplateType.String(), embed.Template)
	}

	if name != "" {
		return str + resourceTypeStr + ": " + name
	} else if index >= 0 {
		return str + resourceTypeStr + " at " + s.Index(index)
	} else {
		return str + resourceTypeStr
	}
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
