# Copyright 2021 Cortex Labs, Inc.
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

import datetime
import json
import os
import time
from typing import List, Tuple

import boto3
import botocore
import msgpack

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import CortexException


class S3:
    def __init__(self, bucket=None, region=None, client_config={}):
        self.bucket = bucket
        self.region = region

        if client_config is None:
            client_config = {}

        if region is not None:
            client_config["region_name"] = region

        session = boto3.Session()
        credentials = session.get_credentials()

        # use anonymous client if credentials haven't been detected
        if credentials is None:
            client_config["config"] = botocore.client.Config(signature_version=botocore.UNSIGNED)

        self.s3 = boto3.client("s3", **client_config)

    @staticmethod
    def construct_s3_path(bucket_name: str, prefix: str) -> str:
        return f"s3://{bucket_name}/{prefix}"

    @staticmethod
    def deconstruct_s3_path(s3_path: str) -> Tuple[str, str]:
        path = util.trim_prefix(s3_path, "s3://")
        bucket = path.split("/")[0]
        key = os.path.join(*path.split("/")[1:])
        return bucket, key

    @staticmethod
    def is_valid_s3_path(path: str) -> bool:
        if not path.startswith("s3://"):
            return False
        parts = path[5:].split("/")
        if len(parts) < 2:
            return False
        if parts[0] == "" or parts[1] == "":
            return False
        return True

    def blob_path(self, key):
        return os.path.join("s3://", self.bucket, key)

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
        prefix = util.ensure_suffix(dir_path, "/")
        return self._is_s3_prefix(prefix)

    def _get_matching_s3_objects_generator(self, prefix="", suffix="", include_dir_objects=False):
        kwargs = {"Bucket": self.bucket, "Prefix": prefix}

        while True:
            resp = self.s3.list_objects_v2(**kwargs)
            try:
                contents = resp["Contents"]
            except KeyError:
                return

            for obj in contents:
                key = obj["Key"]
                if (
                    key.startswith(prefix)
                    and key.endswith(suffix)
                    and (include_dir_objects or not key.endswith("/"))
                ):
                    yield obj

            try:
                kwargs["ContinuationToken"] = resp["NextContinuationToken"]
            except KeyError:
                break

    def _get_matching_s3_keys_generator(self, prefix="", suffix="", include_dir_objects=False):
        for obj in self._get_matching_s3_objects_generator(prefix, suffix, include_dir_objects):
            yield obj["Key"], obj["LastModified"]

    def put_object(self, body, key):
        self.s3.put_object(Bucket=self.bucket, Key=key, Body=body)

    def get_object(self, key):
        return self.s3.get_object(Bucket=self.bucket, Key=key)

    def _read_bytes_from_s3(
        self, key, allow_missing=False, ext_bucket=None, num_retries=0, retry_delay_sec=2
    ):
        while True:
            try:
                return self._read_bytes_from_s3_single(
                    key, allow_missing=allow_missing, ext_bucket=ext_bucket
                )
            except:
                if num_retries <= 0:
                    raise
                num_retries -= 1
                time.sleep(retry_delay_sec)

    def _read_bytes_from_s3_single(self, key, allow_missing=False, ext_bucket=None):
        bucket = self.bucket
        if ext_bucket is not None:
            bucket = ext_bucket

        try:
            try:
                byte_array = self.s3.get_object(Bucket=bucket, Key=key)["Body"].read()
            except self.s3.exceptions.NoSuchKey:
                if allow_missing:
                    return None
                raise
        except Exception as e:
            raise CortexException(
                'key "{}" in bucket "{}" could not be accessed; '.format(key, bucket)
                + "it may not exist, or you may not have sufficient permissions"
            ) from e

        return byte_array.strip()

    def search(
        self, prefix="", suffix="", include_dir_objects=False
    ) -> Tuple[List[str], List[datetime.datetime]]:
        paths = []
        timestamps = []

        for key, ts in self._get_matching_s3_keys_generator(prefix, suffix, include_dir_objects):
            paths.append(key)
            timestamps.append(ts)

        return paths, timestamps

    def put_str(self, str_val, key):
        self.put_object(str_val, key)

    def put_json(self, obj, key):
        self.put_object(json.dumps(obj), key)

    def get_json(self, key, allow_missing=False, num_retries=0, retry_delay_sec=2):
        obj = self._read_bytes_from_s3(
            key,
            allow_missing=allow_missing,
            num_retries=num_retries,
            retry_delay_sec=retry_delay_sec,
        )
        if obj is None:
            return None
        return json.loads(obj.decode("utf-8"))

    def put_msgpack(self, obj, key):
        self.put_object(msgpack.dumps(obj), key)

    def get_msgpack(self, key, allow_missing=False, num_retries=0, retry_delay_sec=2):
        obj = self._read_bytes_from_s3(
            key,
            allow_missing=allow_missing,
            num_retries=num_retries,
            retry_delay_sec=retry_delay_sec,
        )
        if obj == None:
            return None
        return msgpack.loads(obj, raw=False)

    def upload_file(self, local_path, key):
        self.s3.upload_file(local_path, self.bucket, key)

    def download_file_to_dir(self, key, local_dir_path):
        filename = os.path.basename(key)
        return self.download_file(key, os.path.join(local_dir_path, filename))

    def download_file(self, key, local_path):
        util.mkdir_p(os.path.dirname(local_path))
        try:
            self.s3.download_file(self.bucket, key, local_path)
            return local_path
        except Exception as e:
            raise CortexException(
                'key "{}" in bucket "{}" could not be accessed; '.format(key, self.bucket)
                + "it may not exist, or you may not have sufficient permissions"
            ) from e

    def download_dir(self, prefix, local_dir):
        dir_name = util.trim_suffix(prefix, "/").split("/")[-1]
        return self.download_dir_contents(prefix, os.path.join(local_dir, dir_name))

    def download_dir_contents(self, prefix, local_dir):
        util.mkdir_p(local_dir)
        prefix = util.ensure_suffix(prefix, "/")
        for key, _ in self._get_matching_s3_keys_generator(prefix, include_dir_objects=True):
            rel_path = util.trim_prefix(key, prefix)
            local_dest_path = os.path.join(local_dir, rel_path)

            if not local_dest_path.endswith("/"):
                self.download_file(key, local_dest_path)
            else:
                util.mkdir_p(os.path.dirname(local_dest_path))

    def download_and_unzip(self, key, local_dir):
        util.mkdir_p(local_dir)
        local_zip = os.path.join(local_dir, "zip.zip")
        self.download_file(key, local_zip)
        util.extract_zip(local_zip, delete_zip_file=True)

    def download(self, prefix, local_dir):
        if self._is_s3_dir(prefix):
            self.download_dir(prefix, local_dir)
        else:
            self.download_file_to_dir(prefix, local_dir)

    def delete(self, key):
        self.s3.delete_object(Bucket=self.bucket, Key=key)
