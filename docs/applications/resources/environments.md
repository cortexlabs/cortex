# Environments

Transfer data at scale from data warehouses like S3 into the Cortex cluster. Once data is ingested, it’s lifecycle is fully managed by Cortex.

## Config

```yaml
- kind: environment  # (required)
  name: <string>  # environment name (required)
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
    - <string>  # raw feature names listed in the CSV columns' order (required)
      ...
```

### Parquet Data Config

```yaml
data:
  type: parquet  # file type (required)
  path: s3a://<bucket_name>/<file_name>  # S3 is currently supported (required)
  drop_null: <bool>  # drop any rows that contain at least 1 null value (default: false)
  schema:
    - column_name: <string>  # name of the column in the parquet file (required)
      feature_name: <string>  # raw feature name (required)
      ...
```

## Example

```yaml
- kind: environment
  name: dev
  data:
    type: csv
    path: s3a://my-bucket/data.csv
    drop_null: true
    schema:
      - feature1
      - feature2
      - feature3
      - label

- kind: environment
  name: prod
  data:
    type: parquet
    path: s3a://my-bucket/data.parquet
    schema:
      - column_name: column1
        feature_name: feature1
      - column_name: column2
        feature_name: feature2
      - column_name: column3
        feature_name: feature3
      - column_name: column4
        feature_name: label
```
