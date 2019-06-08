import tensorflow as tf


def create_estimator(run_config, model_config):
    dnn_feature_columns = []

    for col_name in model_config["input"]["dnn_columns"]["numeric_columns"]:
        feature_columns.append(tf.feature_column.numeric_column(col_name))

    for col_info in model_config["input"]["dnn_columns"]["categorical_columns_with_vocab"]:
        col = tf.feature_column.categorical_column_with_vocabulary_list(
            col_info["col"], col_info["vocab"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        if "embedding_size" in col_info:
            col = tf.feature_column.embedding_column(col, col_info["embedding_size"])
        else:
            col = tf.feature_column.indicator_column(col)

        dnn_feature_columns.append(col)

    for col_info in model_config["input"]["dnn_columns"]["categorical_columns_with_identity"]:
        col = tf.feature_column.categorical_columns_with_identity(
            col_info["col"], col_info["num_classes"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        if "embedding_size" in col_info:
            col = tf.feature_column.embedding_column(col, col_info["embedding_size"])
        else:
            col = tf.feature_column.indicator_column(col)

        dnn_feature_columns.append(col)

    for col_info in model_config["input"]["dnn_columns"]["categorical_columns_with_hash_bucket"]:
        col = tf.feature_column.categorical_columns_with_hash_bucket(
            col_info["col"], col_info["hash_bucket_size"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        if "embedding_size" in col_info:
            col = tf.feature_column.embedding_column(col, col_info["embedding_size"])
        else:
            col = tf.feature_column.indicator_column(col)

        dnn_feature_columns.append(col)

    for col_info in model_config["input"]["dnn_columns"]["bucketized_columns"]:
        dnn_feature_columns.append(
            tf.feature_column.bucketized_column(
                tf.feature_column.numeric_column(col_info["col"]), col_info["boundaries"]
            )
        )

    linear_feature_columns = []

    for col_name in model_config["input"]["linear_columns"]["numeric_columns"]:
        linear_feature_columns.append(tf.feature_column.numeric_column(col_name))

    for col_info in model_config["input"]["linear_columns"]["categorical_columns_with_vocab"]:
        col = tf.feature_column.categorical_column_with_vocabulary_list(
            col_info["col"], col_info["vocab"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        linear_feature_columns.append(col)

    for col_info in model_config["input"]["linear_columns"]["categorical_columns_with_identity"]:
        col = tf.feature_column.categorical_columns_with_identity(
            col_info["col"], col_info["num_classes"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        linear_feature_columns.append(col)

    for col_info in model_config["input"]["linear_columns"]["categorical_columns_with_hash_bucket"]:
        col = tf.feature_column.categorical_columns_with_hash_bucket(
            col_info["col"], col_info["hash_bucket_size"]
        )

        if "weight_column" in col_info:
            col = tf.feature_column.weighted_categorical_column(col, col_info["weight_column"])

        linear_feature_columns.append(col)

    for col_info in model_config["input"]["linear_columns"]["bucketized_columns"]:
        linear_feature_columns.append(
            tf.feature_column.bucketized_column(
                tf.feature_column.numeric_column(col_info["col"]), col_info["boundaries"]
            )
        )

    return tf.estimator.DNNClassifier(
        linear_feature_columns=linear_feature_columns,
        dnn_feature_columns=dnn_feature_columns,
        dnn_hidden_units=model_config["hparams"]["dnn_hidden_units"],
        weight_column=model_config["input"].get("weight_column", None),
        config=run_config,
    )
