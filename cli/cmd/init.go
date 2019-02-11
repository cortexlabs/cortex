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

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

var initCmd = &cobra.Command{
	Use:   "init APP_NAME",
	Short: "initialize an application",
	Long:  "Initialize an application.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			errors.Exit(err)
		}

		if currentRoot := appRootOrBlank(); currentRoot != "" {
			errors.Exit(s.ErrCliAlreadyInAppDir(currentRoot))
		}

		appRoot := filepath.Join(cwd, appName)
		var createdFiles []string

		if !createDirIfMissing(appRoot) {
			errors.Exit(s.ErrCwdDirExists(appName))
		}

		for path, content := range appInitFiles(appName) {
			createdFiles = writeFile(path, content, appRoot, createdFiles)
		}

		fmt.Println("Created files:")
		fmt.Println(util.FileTree(createdFiles, cwd, util.DirsOnBottom))
	},
}

func createDirIfMissing(path string) bool {
	created, err := util.CreateDirIfMissing(path)
	if err != nil {
		errors.Exit(err)
	}
	return created
}

func writeFile(subPath string, content string, root string, createdFiles []string) []string {
	path := filepath.Join(root, subPath)
	createDirIfMissing(filepath.Dir(path))
	err := ioutil.WriteFile(path, []byte(content), 0664)
	if err != nil {
		errors.Exit(err, s.ErrCreateFile(path))
	}
	return append(createdFiles, path)
}

func appInitFiles(appName string) map[string]string {
	return map[string]string{
		"app.yaml": fmt.Sprintf("- kind: app\n  name: %s\n", appName),

		"resources/environments.yaml": `## Sample environment:
#
# - kind: environment
#   name: dev
#   data:
#     type: csv
#     path: s3a://my-bucket/data.csv
#     csv_config:
#       header: true
#     schema:
#       - feature1
#       - feature2
#       - feature3
#       - label
`,

		"resources/raw_features.yaml": `## Sample raw features:
#
# - kind: raw_feature
#   name: feature1
#   type: INT_FEATURE
#   required: true
#   min: 0
#   max: 10
#
# - kind: raw_feature
#   name: feature2
#   type: FLOAT_FEATURE
#   required: true
#   min: 1.1
#   max: 2.2
#
# - kind: raw_feature
#   name: feature3
#   type: STRING_FEATURE
#   required: false
#   values: [a, b, c]
`,

		"resources/aggregates.yaml": `## Sample aggregates:
#
# - kind: aggregate
#   name: feature1_bucket_boundaries
#   aggregator: cortex.bucket_boundaries
#   inputs:
#     features:
#       col: feature1
#     args:
#       num_buckets: 3
`,

		"resources/transformed_features.yaml": `## Sample transformed features:
#
# - kind: transformed_feature
#   name: feature1_bucketized
#   transformer: cortex.bucketize  # Cortex provided transformer in pkg/transformers
#   inputs:
#     features:
#       num: feature1
#     args:
#       bucket_boundaries: feature2_bucket_boundaries
#
# - kind: transformed_feature
#   name: feature2_transformed
#   transformer: my_transformer  # Your own custom transformer from the transformers folder
#   inputs:
#     features:
#       num: feature2
#     args:
#       arg1: 10
#       arg2: 100
`,

		"resources/models.yaml": `## Sample model:
#
# - kind: model
#   name: my_model
#   type: classification
#   target: label
#   features:
#     - feature1
#     - feature2
#     - feature3
#   hparams:
#     hidden_units: [4, 2]
#   data_partition_ratio:
#     training: 0.9
#     evaluation: 0.1
#   training:
#     batch_size: 10
#     num_steps: 1000
`,

		"resources/apis.yaml": `## Sample API:
#
# - kind: api
#   name: my_api
#   model_name: my_model
#   compute:
#     replicas: 1
`,

		"samples.json": `{
  "samples": [
    {
      "key1": "value1",
      "key2": "value2",
      "key3": "value3"
    }
  ]
}
`,

		"implementations/models/model.py": `import tensorflow as tf


def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model.
            Note: nested resources are expanded (e.g. model_config["target"])
            will be the configuration for the target feature, rather than the
            name of the target feature).

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """

    ## Sample create_estimator implementation:
    #
    # columns = [
    #     tf.feature_column.numeric_column("feature1"),
    #     tf.feature_column.indicator_column(
    #         tf.feature_column.categorical_column_with_identity("feature2", num_buckets=3)
    #     ),
    # ]
    #
    # return tf.estimator.DNNRegressor(
    #     feature_columns=columns,
    #     hidden_units=model_config["hparams"]["hidden_units"],
    #     config=run_config,
    # )

    pass
`,

		"resources/constants.yaml": `## Sample constant:
#
# - kind: constant
#   name: my_constant
#   type: [INT]
#   value: [0, 50, 100]
`,

		"resources/aggregators.yaml": `## Sample aggregator:
#
# - kind: aggregator
#   name: my_aggregator
#   output_type: [FLOAT]
#   inputs:
#     features:
#       feature1: FLOAT_FEATURE|INT_FEATURE
#     args:
#       arg1: INT
`,

		"implementations/aggregators/aggregator.py": `def aggregate_spark(data, features, args):
    """Aggregate a feature in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw features.

        features: A dict with the same structure as the aggregator's input
            features specifying the names of the dataframe's columns that
            contain the input features.

        args: A dict with the same structure as the aggregator's input args
            containing the runtime values of the args.

    Returns:
        Any json-serializable object that matches the data type of the aggregator.
    """

    ## Sample aggregate_spark implementation:
    #
    # from pyspark.ml.feature import QuantileDiscretizer
    #
    # discretizer = QuantileDiscretizer(
    #     numBuckets=args["num_buckets"], inputCol=features["col"], outputCol="_"
    # ).fit(data)
    #
    # return discretizer.getSplits()

    pass
`,

		"resources/transformers.yaml": `## Sample transformer:
#
# - kind: transformer
#   name: my_transformer
#   output_type: INT_FEATURE
#   inputs:
#     features:
#       feature1: INT_FEATURE|FLOAT_FEATURE
#     args:
#       arg1: FLOAT
#       arg2: FLOAT
`,

		"implementations/transformers/transformer.py": `def transform_spark(data, features, args, transformed_feature):
    """Transform a feature in a PySpark context.

    This function is optional (recommended for large-scale feature processing).

    Args:
        data: A dataframe including all of the raw features.

        features: A dict with the same structure as the transformer's input
            features specifying the names of the dataframe's columns that
            contain the input features.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

        transformed_feature: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column with the name of the
        transformed_feature arg containing the transformed data.
    """

    ## Sample transform_spark implementation:
    #
    # return data.withColumn(
    #     transformed_feature, ((data[features["num"]] - args["mean"]) / args["stddev"])
    # )

    pass


def transform_python(sample, args):
    """Transform a single data sample outside of a PySpark context.

    This function is required.

    Args:
        sample: A dict with the same structure as the transformer's input
            features containing a data sample to transform.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

    Returns:
        The transformed value.
    """

    ## Sample transform_python implementation:
    #
    # return (sample["num"] - args["mean"]) / args["stddev"]

    pass


def reverse_transform_python(transformed_value, args):
    """Reverse transform a single data sample outside of a PySpark context.

    This function is optional, and only relevant for certain one-to-one
    transformers.

    Args:
        transformed_value: The transformed data value.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

    Returns:
        The raw data value that corresponds to the transformed value.
    """

    ## Sample reverse_transform_python implementation:
    #
    # return args["mean"] + (transformed_value * args["stddev"])

    pass
`,
	}
}
