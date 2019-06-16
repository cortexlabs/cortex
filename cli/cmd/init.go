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
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

var initCmd = &cobra.Command{
	Use:   "init APP_NAME",
	Short: "initialize an application",
	Long:  "Initialize an application.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]

		if err := urls.CheckDNS1123(appName); err != nil {
			errors.Exit(err)
		}

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
#       - @column1
#       - @column2
#       - @column3
#       - @label
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
#   input:
#     col: @column1
#     num_buckets: 3
`,

		"resources/transformed_columns.yaml": `## Sample transformed columns:
#
# - kind: transformed_column
#   name: column1_bucketized
#   transformer: cortex.bucketize  # Cortex provided transformer in pkg/transformers
#   input:
#     num: @column1
#     bucket_boundaries: @column2_bucket_boundaries
#
# - kind: transformed_column
#   name: column2_transformed
#   transformer: my_transformer  # Your own custom transformer
#   inputs:
#     col: @column2
#     arg1: 10
#     arg2: 100
`,

		"resources/models.yaml": `## Sample model:
#
# - kind: model
#   name: dnn
#   estimator: cortex.dnn_classifier
#   target_column: @class
#   input:
#     numeric_columns: [@column1, @column2, @column3]
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
#   model: @my_model
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

		"resources/constants.yaml": `## Sample constant:
#
# - kind: constant
#   name: my_constant
#   value: [0, 50, 100]
`,

		"resources/aggregators.yaml": `## Sample aggregator:
#
# - kind: aggregator
#   name: my_aggregator
#   output_type: [FLOAT]
#   input:
#     column1: FLOAT_COLUMN|INT_COLUMN
#     arg1: INT
`,

		"implementations/aggregators/my_aggregator.py": `def aggregate_spark(data, input):
    """Aggregate a column in a PySpark context.

    This function is required.

    Args:
        data: A dataframe including all of the raw columns.

        input: The aggregate's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants) are replaced by their
            runtime values.

    Returns:
        Any serializable object that matches the output type of the aggregator.
    """

    ## Sample aggregate_spark implementation:
    #
    # from pyspark.ml.feature import QuantileDiscretizer
    #
    # discretizer = QuantileDiscretizer(
    #     numBuckets=input["num_buckets"], inputCol=input["col"], outputCol="_"
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
#   input:
#     column1: INT_COLUMN|FLOAT_COLUMN
#     arg1: FLOAT
#     arg2: FLOAT
`,

		"implementations/transformers/my_transformer.py": `def transform_spark(data, input, transformed_column_name):
    """Transform a column in a PySpark context.

    This function is optional (recommended for large-scale data processing).

    Args:
        data: A dataframe including all of the raw columns.

        input: The transformed column's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants and aggregates) are replaced
            by their runtime values.

        transformed_column_name: The name of the column containing the transformed
            data that is to be appended to the dataframe.

    Returns:
        The original 'data' dataframe with an added column named <transformed_column_name>
        which contains the transformed data.
    """

    ## Sample transform_spark implementation:
    #
    # return data.withColumn(
    #     transformed_column_name, ((data[input["col"]] - input["mean"]) / input["stddev"])
    # )

    pass


def transform_python(input):
    """Transform a single data sample outside of a PySpark context.

    This function is required for any columns that are used during inference.

    Args:
        input: The transformed column's input object. Column references in the input are
            replaced by their values in the sample (e.g. "@column1" will be replaced with
            the value for column1), and all other resource references (e.g. constants and
            aggregates) are replaced by their runtime values.

    Returns:
        The transformed value.
    """

    ## Sample transform_python implementation:
    #
    # return (input["col"] - input["mean"]) / input["stddev"]

    pass


def reverse_transform_python(transformed_value, input):
    """Reverse transform a single data sample outside of a PySpark context.

    This function is optional, and only relevant for certain one-to-one
    transformers.

    Args:
        transformed_value: The transformed data value.

        input: The transformed column's input object. Column references in the input are
            replaced by their names (e.g. "@column1" will be replaced with "column1"),
            and all other resource references (e.g. constants and aggregates) are replaced
            by their runtime values.

    Returns:
        The raw data value that corresponds to the transformed value.
    """

    ## Sample reverse_transform_python implementation:
    #
    # return input["mean"] + (transformed_value * input["stddev"])

    pass
`,

		"implementations/estimators/my_estimator.py": `def create_estimator(run_config, model_config):
    """Create an estimator to train the model.

    Args:
        run_config: An instance of tf.estimator.RunConfig to be used when creating
            the estimator.

        model_config: The Cortex configuration for the model. Column references in all
            inputs (i.e. model_config["target_column"], model_config["input"], and
            model_config["training_input"]) are replaced by their names (e.g. "@column1"
            will be replaced with "column1"). All other resource references (e.g. constants
            and aggregates) are replaced by their runtime values.

    Returns:
        An instance of tf.estimator.Estimator to train the model.
    """

    ## Sample create_estimator implementation:
    #
    # feature_columns = []
    # for col_name in model_config["input"]["numeric_columns"]:
    #     feature_columns.append(tf.feature_column.numeric_column(col_name))
    #
    # return tf.estimator.DNNClassifier(
    #     feature_columns=feature_columns,
    #     n_classes=model_config["input"]["num_classes"],
    #     hidden_units=model_config["hparams"]["hidden_units"],
    #     config=run_config,
    # )

    pass
`,
	}
}
