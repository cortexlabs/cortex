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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
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
			errors.Exit(ErrorCliAlreadyInAppDir(currentRoot))
		}

		appRoot := filepath.Join(cwd, appName)
		var createdFiles []string

		if _, err := files.CreateDirIfMissing(appRoot); err != nil {
			errors.Exit(err)
		}

		for path, content := range appInitFiles(appName) {
			createdFiles = writeFile(path, content, appRoot, createdFiles)
		}

		fmt.Println("Created files:")
		fmt.Println(files.FileTree(createdFiles, cwd, files.DirsOnBottom))
	},
}

func writeFile(subPath string, content string, root string, createdFiles []string) []string {
	path := filepath.Join(root, subPath)
	if _, err := files.CreateDirIfMissing(filepath.Dir(path)); err != nil {
		errors.Exit(err)
	}
	err := files.WriteFile(path, []byte(content), 0664)
	if err != nil {
		errors.Exit(err)
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
#       - column1
#       - column2
#       - column3
#       - label
`,

		"resources/raw_columns.yaml": `## Sample raw columns:
#
# - kind: raw_column
#   name: column1
#   type: INT_COLUMN
#   required: true
#   min: 0
#   max: 10
#
# - kind: raw_column
#   name: column2
#   type: FLOAT_COLUMN
#   required: true
#   min: 1.1
#   max: 2.2
#
# - kind: raw_column
#   name: column3
#   type: STRING_COLUMN
#   required: false
#   values: [a, b, c]
`,

		"resources/aggregates.yaml": `## Sample aggregates:
#
# - kind: aggregate
#   name: column1_bucket_boundaries
#   aggregator: cortex.bucket_boundaries
#   inputs:
#     columns:
#       col: column1
#     args:
#       num_buckets: 3
`,

		"resources/transformed_columns.yaml": `## Sample transformed columns:
#
# - kind: transformed_column
#   name: column1_bucketized
#   transformer: cortex.bucketize  # Cortex provided transformer in pkg/transformers
#   inputs:
#     columns:
#       num: column1
#     args:
#       bucket_boundaries: column2_bucket_boundaries
#
# - kind: transformed_column
#   name: column2_transformed
#   transformer: my_transformer  # Your own custom transformer from the transformers folder
#   inputs:
#     columns:
#       num: column2
#     args:
#       arg1: 10
#       arg2: 100
`,

		"resources/models.yaml": `## Sample model:
#
# - kind: model
#   name: my_model
#   type: classification
#   target_column: label
#   feature_columns:
#     - column1
#     - column2
#     - column3
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
#   name: my-api
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

		"implementations/models/my_model.py": `import tensorflow as tf


def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model.
            Note: nested resources are expanded (e.g. model_config["target_column"])
            will be the configuration for the target column, rather than the
            name of the target column).

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """

    ## Sample create_estimator implementation:
    #
    # feature_columns = [
    #     tf.feature_column.numeric_column("column1"),
    #     tf.feature_column.indicator_column(
    #         tf.feature_column.categorical_column_with_identity("column2", num_buckets=3)
    #     ),
    # ]
    #
    # return tf.estimator.DNNRegressor(
    #     feature_columns=feature_columns,
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
#     columns:
#       column1: FLOAT_COLUMN|INT_COLUMN
#     args:
#       arg1: INT
`,

		"implementations/aggregators/my_aggregator.py": `def aggregate_spark(data, columns, args):
    """Aggregate a column in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw columns.

        columns: A dict with the same structure as the aggregator's input
            columns specifying the names of the dataframe's columns that
            contain the input columns.

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
    #     numBuckets=args["num_buckets"], inputCol=columns["col"], outputCol="_"
    # ).fit(data)
    #
    # return discretizer.getSplits()

    pass
`,

		"resources/transformers.yaml": `## Sample transformer:
#
# - kind: transformer
#   name: my_transformer
#   output_type: INT_COLUMN
#   inputs:
#     columns:
#       column1: INT_COLUMN|FLOAT_COLUMN
#     args:
#       arg1: FLOAT
#       arg2: FLOAT
`,

		"implementations/transformers/my_transformer.py": `def transform_spark(data, columns, args, transformed_column_name):
    """Transform a column in a PySpark context.

    This function is optional (recommended for large-scale data processing).

    Args:
        data: A dataframe including all of the raw columns.

        columns: A dict with the same structure as the transformer's input
            columns specifying the names of the dataframe's columns that
            contain the input columns.

        args: A dict with the same structure as the transformer's input args
            containing the runtime values of the args.

        transformed_column_name: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column named <transformed_column_name>
        which contains the transformed data.
    """

    ## Sample transform_spark implementation:
    #
    # return data.withColumn(
    #     transformed_column_name, ((data[columns["num"]] - args["mean"]) / args["stddev"])
    # )

    pass


def transform_python(sample, args):
    """Transform a single data sample outside of a PySpark context.

    This function is required.

    Args:
        sample: A dict with the same structure as the transformer's input
            columns containing a data sample to transform.

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
