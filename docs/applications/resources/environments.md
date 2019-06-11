# Environments

Transfer data at scale from data warehouses like S3 into the Cortex environment. Once data is ingested, it’s lifecycle is fully managed by Cortex.

## Config

```yaml
- kind: environment  # (required)
  name: <string>  # environment name (required)
  limit:
      # specify `num_rows` or `fraction_of_rows` if using `limit`
      num_rows: <int>  # maximum number of rows to select from the dataset
      fraction_of_rows: <float>  # fraction of rows to select from the dataset
      randomize: <bool>  # flag to indicate random selection of data (exact dataset size will not be guaranteed when this flag is true)
      random_seed: <int>  # seed value for randomizing
  log_level:
    tensorflow: <string>  # TensorFlow log level (DEBUG, INFO, WARN, ERROR, or FATAL) (default: DEBUG)
    spark: <string>  # Spark log level (ALL, TRACE, DEBUG, INFO, WARN, ERROR, or FATAL) (default: WARN)
  data:
    <data_config>
```

### CSV Data Config

```yaml
data:
  type: csv  # file type (required)
  path: s3a://<bucket_name>/<file_name>  # S3 is currently supported (required)
  region: us-west-2 # S3 region (default: us-west-2)
  drop_null: <bool>  # drop any rows that contain at least 1 null value (default: false)
  csv_config: <csv_config>  # optional configuration that can be provided
  schema:
    - <string>  # raw column names listed in the CSV columns' order (required)
      ...
```

#### CSV Config

To help ingest different styles of CSV files, Cortex supports the parameters listed below. All of these parameters are optional. A description and default values for each parameter can be found in the [PySpark CSV Documentation](https://spark.apache.org/docs/2.4.2/api/python/pyspark.sql.html#pyspark.sql.DataFrameReader.csv).

```yaml
csv_config:
  sep: <string>
  encoding: <string>
  quote: <string>
  escape: <string>
  comment: <string>
  header: <bool>
  ignore_leading_white_space: <bool>
  ignore_trailing_white_space: <bool>
  null_value: <string>
  nan_value: <string>
  positive_inf: <bool>
  negative_inf: <bool>
  max_columns: <int>
  max_chars_per_column: <int>
  multiline: <bool>
  char_to_escape_quote_escaping: <string>
  empty_value: <string>
```

### Parquet Data Config

```yaml
data:
  type: parquet  # file type (required)
  path: s3a://<bucket_name>/<file_name>  # S3 is currently supported (required)
  region: us-west-2 # S3 region (default: us-west-2)
  drop_null: <bool>  # drop any rows that contain at least 1 null value (default: false)
  schema:
    - parquet_column_name: <string>  # name of the column in the parquet file (required)
      raw_column_name: <string>  # raw column name (required)
      ...
```

## Example

```yaml
- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://my-bucket/data.csv
    schema:
      - column1
      - column2
      - column3
      - label

- kind: environment
  name: prod
  data:
    type: parquet
    path: s3a://my-bucket/data.parquet
    schema:
      - parquet_column_name: column1
        raw_column_name: column1
      - parquet_column_name: column2
        raw_column_name: column2
      - parquet_column_name: column3
        raw_column_name: column3
      - parquet_column_name: column4
        raw_column_name: label
```
