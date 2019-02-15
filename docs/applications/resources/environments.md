# Environments

Transfer data at scale from data warehouses like S3 into the Cortex environment. Once data is ingested, it’s lifecycle is fully managed by Cortex.

## Config

```yaml
- kind: environment  # (required)
  name: <string>  # environment name (required)
  log_level:
    tensorflow: <string>  # TensorFlow log level (DEBUG, INFO, WARN, ERROR, or FATAL) (default: INFO)
    spark: <string>  # Spark log level (ALL, TRACE, DEBUG, INFO, WARN, ERROR, or FATAL) (default: WARN)
  data:
    <data_config>
```

### CSV Data Config

```yaml
data:
  type: csv  # file type (required)
  path: s3a://<bucket_name>/<file_name>  # S3 is currently supported (required)
  skip_header: <bool>  # skips a single header line (default: false)
  drop_null: <bool>  # drop any rows that contain at least 1 null value (default: false)
  schema:
    - <string>  # raw column names listed in the CSV columns' order (required)
      ...
```

### Parquet Data Config

```yaml
data:
  type: parquet  # file type (required)
  path: s3a://<bucket_name>/<file_name>  # S3 is currently supported (required)
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
