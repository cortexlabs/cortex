package models

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
)

// IsValidS3Directory checks that the path contains a valid S3 directory for Tensorflow models
// Must contain the following structure:
// - 1523423423/ (timestamped prefix)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func IsValidS3Directory(path string) bool {
	listOut, err := aws.ListObjectsExternal(path)
	if err != nil {
		return false
	}

	if listOut.Prefix == nil {
		return false
	}

	prefix := *listOut.Prefix
	prefixParts := strings.Split(prefix, "/")
	timestamp := prefixParts[len(prefixParts)-1]
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		return false
	}

	var containsVariableDataFile bool
	objects := strset.New()
	for _, o := range listOut.Contents {
		if strings.Contains(*o.Key, "variables/variables.data-00000-of") {
			containsVariableDataFile = true
		}
		objects.Add(*o.Key)
	}

	return objects.Has(
		fmt.Sprintf("%s/saved_model.pb", prefix),
		fmt.Sprintf("%s/variables/variables.index", prefix),
	) && containsVariableDataFile
}
