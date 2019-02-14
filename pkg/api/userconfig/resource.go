package userconfig

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/api/resource"
)

type Resource interface {
	GetName() string
	GetResourceType() resource.Type
	GetFilePath() string
}

func Identify(r Resource) string {
	return fmt.Sprintf("%s: %s: %s", r.GetFilePath(), r.GetResourceType().String(), r.GetName())
}

func FindDuplicateResourceName(in ...Resource) []Resource {
	names := map[string][]Resource{}
	for _, elem := range in {
		names[elem.GetName()] = append(names[elem.GetName()], elem)
	}

	for key := range names {
		if len(names[key]) > 1 {
			return names[key]
		}
	}

	return nil
}
