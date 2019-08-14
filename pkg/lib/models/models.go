package models

import (
	"fmt"
	"strconv"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
)

// IsValidS3Directory checks that the path contains a valid S3 directory for Tensorflow models
// Must contain the following structure:
// - 1523423423/ (timestamped prefix)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001
func IsValidS3Directory(path string) bool {
	listOut, err := aws.ListObjectsExternal(path)
	if err != nil {
		return false
	}

	if listOut.Prefix == nil {
		return false
	}

	prefix := *listOut.Prefix
	if _, err := strconv.ParseInt(prefix, 10, 64); err != nil {
		return false
	}

	objects := strset.New()
	for _, o := range listOut.Contents {
		objects.Add(*o.Key)
	}

	return objects.Has(
		fmt.Sprintf("%s/saved_model.pb", prefix),
		fmt.Sprintf("%s/variables/variables.index", prefix),
		fmt.Sprintf("%s/variables/variables.data-00000-of-00001", prefix),
	)
}
