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


class S3(object):
    def __init__(self, bucket=None, region=None, client_config={}):
        self.bucket = bucket
        self.region = region

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

        if region is not None:
            client_config["region_name"] = region

        merged_client_config = util.merge_dicts_in_place_no_overwrite(client_config, default_config)

        self.s3 = boto3.client("s3", **client_config)

    @staticmethod
    def deconstruct_s3_path(s3_path):
        path = util.remove_prefix_if_present(s3_path, "s3://")
        bucket = path.split("/")[0]
        key = os.path.join(*path.split("/")[1:])
        return (bucket, key)

    def hadoop_path(self, key):
        return os.path.join("s3a://", self.bucket, key)

    def blob_path(self, key):
        return os.path.join("s3://", self.bucket, key)

    def _get_dir(self, prefix, local_dir):
        prefix = util.add_suffix_unless_present(prefix, "/")
        util.mkdir_p(local_dir)
        for key in self._get_matching_s3_keys_generator(prefix):
            rel_path = util.remove_prefix_if_present(key, prefix)
            local_dest_path = os.path.join(local_dir, rel_path)
            self.download_file(key, local_dest_path)

    def _file_exists(self, key):
        try:
            self.s3.head_object(Bucket=self.bucket, Key=key)
            return True
        except botocore.exceptions.ClientError as e:
            if e.response["Error"]["Code"] == "404":
                return False
            else:
                raise

    def _is_s3_prefix(self, prefix):
        response = self.s3.list_objects_v2(Bucket=self.bucket, Prefix=prefix)
        return response["KeyCount"] > 0

    def _is_s3_dir(self, dir_path):
        prefix = util.add_suffix_unless_present(dir_path, "/")
        return self._is_s3_prefix(prefix)

    def _get_matching_s3_objects_generator(self, prefix="", suffix=""):
        kwargs = {"Bucket": self.bucket, "Prefix": prefix}

        while True:
            resp = self.s3.list_objects_v2(**kwargs)
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

    def _get_matching_s3_keys_generator(self, prefix="", suffix=""):
        for obj in self._get_matching_s3_objects_generator(prefix, suffix):
            yield obj["Key"]

    def _upload_string_to_s3(self, string, key):
        self.s3.put_object(Bucket=self.bucket, Key=key, Body=string)

    def _read_bytes_from_s3(self, key, allow_missing=False):
        try:
            byte_array = self.s3.get_object(Bucket=self.bucket, Key=key)["Body"].read()
        except self.s3.exceptions.NoSuchKey as e:
            if allow_missing:
                return None
            raise CortexException("bucket " + self.bucket, "key " + key) from e

        return byte_array.strip()

    def search(self, prefix="", suffix=""):
        return list(self._get_matching_s3_keys_generator(prefix, suffix))

    def put_json(self, obj, key):
        self._upload_string_to_s3(json.dumps(obj), key)

    def get_json(self, key, allow_missing=False):
        obj = self._read_bytes_from_s3(key, allow_missing)
        if obj is None:
            return None
        return json.loads(obj.decode("utf-8"))

    def put_msgpack(self, obj, key):
        self._upload_string_to_s3(msgpack.dumps(obj), key)

    def get_msgpack(self, key, allow_missing=False):
        obj = self._read_bytes_from_s3(key, allow_missing)
        if obj == None:
            return None
        return msgpack.load(obj, raw=False)

    def put_pyobj(self, obj, key):
        self._upload_string_to_s3(pickle.dumps(obj), key)

    def get_pyobj(self, key, allow_missing=False):
        obj = self._read_bytes_from_s3(key, allow_missing)
        if obj is None:
            return None
        return pickle.loads(obj)

    def upload_file(self, local_path, key):
        self.s3.upload_file(local_path, self.bucket, key)

    def download_file(self, key, local_path):
        try:
            util.mkdir_p(os.path.dirname(local_path))
            self.s3.download_file(self.bucket, key, local_path)
            return local_path
        except Exception as e:
            raise CortexException("bucket " + self.bucket, "key " + key) from e

    def zip_and_upload(self, local_path, key):
        util.zip_dir(local_path, "temp.zip")
        self.s3.upload_file("temp.zip", self.bucket, key)
        util.rm_file("temp.zip")

    def download_and_unzip(self, key, local_dir):
        util.mkdir_p(local_dir)
        local_zip = os.path.join(local_dir, "zip.zip")
        self.download_file(key, local_zip)
        util.extract_zip(local_zip, delete_zip_file=True)
