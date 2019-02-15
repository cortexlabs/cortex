# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import sys
import os
import boto3
import botocore
import pickle
import json
import msgpack

from lib import util
from lib.exceptions import CortexException


def set_client_config_defaults(client_config={}):
    default_config = {
        "use_ssl": True,
        "verify": True,
        "region_name": None,
        "aws_access_key_id": None,
        "aws_secret_access_key": None,
        "aws_session_token": None,
    }

    if client_config is None:
        client_config = {}

    return util.merge_dicts_in_place_no_overwrite(client_config, default_config)


def s3_client(client_config={}):
    set_client_config_defaults(client_config)
    return boto3.client("s3", **client_config)


def s3a_path(*keys):
    keys = util.flatten(keys)
    return os.path.join("s3a://", *keys)


def s3_path(*keys):
    keys = util.flatten(keys)
    return os.path.join("s3://", *keys)


def deconstruct_s3_path(s3_path):
    path = util.remove_prefix_if_present(s3_path, "s3://")
    bucket = path.split("/")[0]
    key = os.path.join(*path.split("/")[1:])
    return (bucket, key)


def is_s3_file(key, bucket, client_config={}):
    s3 = s3_client(client_config)
    try:
        s3.head_object(Bucket=bucket, Key=key)
        return True
    except botocore.exceptions.ClientError as e:
        if e.response["Error"]["Code"] == "404":
            return False
        else:
            raise


def is_s3_prefix(prefix, bucket, client_config={}):
    s3 = s3_client(client_config)
    response = s3.list_objects_v2(Bucket=bucket, Prefix=prefix)
    return response["KeyCount"] > 0


def is_s3_dir(dir_path, bucket, client_config={}):
    prefix = util.add_suffix_unless_present(dir_path, "/")
    return is_s3_prefix(prefix, bucket, client_config=client_config)


# prefix/suffix can be a string or a list/tuple
def get_matching_s3_objects_generator(bucket, prefix="", suffix="", client_config={}):
    s3 = s3_client(client_config)
    kwargs = {"Bucket": bucket}

    if isinstance(prefix, str):
        kwargs["Prefix"] = prefix

    while True:
        resp = s3.list_objects_v2(**kwargs)
        try:
            contents = resp["Contents"]
        except KeyError:
            return

        for obj in contents:
            key = obj["Key"]
            if key.startswith(prefix) and key.endswith(suffix):
                yield obj

        try:
            kwargs["ContinuationToken"] = resp["NextContinuationToken"]
        except KeyError:
            break


# prefix/suffix can be a string or a list/tuple
def get_matching_s3_keys_generator(bucket, prefix="", suffix="", client_config={}):
    for obj in get_matching_s3_objects_generator(bucket, prefix, suffix, client_config):
        yield obj["Key"]


def get_matching_s3_keys(bucket, prefix="", suffix="", client_config={}):
    return list(get_matching_s3_keys_generator(bucket, prefix, suffix, client_config))


def upload_string_to_s3(string, key, bucket, client_config={}):
    s3 = s3_client(client_config)
    s3.put_object(Bucket=bucket, Key=key, Body=string)


def read_bytes_from_s3(key, bucket, allow_missing=True, client_config={}):
    s3 = s3_client(client_config)
    try:
        byte_array = s3.get_object(Bucket=bucket, Key=key)["Body"].read()
    except s3.exceptions.NoSuchKey as e:
        if allow_missing:
            return None
        raise e

    return byte_array.strip()


def read_string_from_s3(key, bucket, allow_missing=True, decoding="utf-8", client_config={}):
    return read_bytes_from_s3(key, bucket, allow_missing, client_config).decode(decoding)


def upload_empty_file_to_s3(key, bucket, client_config={}):
    return upload_string_to_s3("", key, bucket, client_config)


def upload_json_to_s3(obj, key, bucket, client_config={}):
    upload_string_to_s3(json.dumps(obj), key, bucket, client_config)


def read_json_from_s3(key, bucket, allow_missing=True, client_config={}):
    obj = read_bytes_from_s3(key, bucket, allow_missing, client_config).decode("utf-8")
    if obj is None:
        return None
    return json.loads(obj)


def upload_msgpack_to_s3(obj, key, bucket, client_config={}):
    upload_string_to_s3(msgpack.dumps(obj), key, bucket, client_config)


def read_msgpack_from_s3(key, bucket, allow_missing=True, client_config={}):
    obj = read_bytes_from_s3(key, bucket, allow_missing, client_config)
    if obj == None:
        return None
    return msgpack.loads(obj)


def upload_obj_to_s3(obj, key, bucket, client_config={}):
    upload_string_to_s3(pickle.dumps(obj), key, bucket, client_config)


def read_obj_from_s3(key, bucket, allow_missing=True, client_config={}):
    obj = read_bytes_from_s3(key, bucket, allow_missing, client_config)
    if obj is None:
        return None
    return pickle.loads(obj)


def upload_file_to_s3(local_path, key, bucket, client_config={}):
    s3 = s3_client(client_config)
    s3.upload_file(local_path, bucket, key)


def download_file_from_s3(key, local_path, bucket, client_config={}):
    try:
        util.mkdir_p(os.path.dirname(local_path))
        s3 = s3_client(client_config)
        s3.download_file(bucket, key, local_path)
        return local_path
    except Exception as e:
        raise CortexException("bucket " + bucket, "key " + key) from e


def download_dir_from_s3(prefix, local_dir, bucket, client_config={}):
    prefix = util.add_suffix_unless_present(prefix, "/")
    util.mkdir_p(local_dir)
    for key in get_matching_s3_keys_generator(bucket, prefix, client_config=client_config):
        rel_path = util.remove_prefix_if_present(key, prefix)
        local_dest_path = os.path.join(local_dir, rel_path)
        download_file_from_s3(key, local_dest_path, bucket, client_config=client_config)


def compress_zip_and_upload(local_path, key, bucket, client_config={}):
    s3 = s3_client(client_config)
    util.zip_dir(local_path, "temp.zip")
    s3.upload_file("temp.zip", bucket, key)
    util.rm_file("temp.zip")


def download_and_extract_zip(key, local_dir, bucket, client_config={}):
    util.mkdir_p(local_dir)
    local_zip = os.path.join(local_dir, "zip.zip")
    download_file_from_s3(key, local_zip, bucket, client_config=client_config)
    util.extract_zip(local_zip, delete_zip_file=True)
