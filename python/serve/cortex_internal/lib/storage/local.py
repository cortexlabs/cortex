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

import json
import os
import shutil
import time
from pathlib import Path

import msgpack

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import CortexException


class LocalStorage(object):
    def __init__(self, base_dir):
        self.base_dir = base_dir

    def _get_path(self, key):
        return Path(os.path.join(self.base_dir, key))

    def _get_or_create_path(self, key):
        p = Path(os.path.join(self.base_dir, key))
        p.parent.mkdir(parents=True, exist_ok=True)
        return p

    def _get_path_if_exists(self, key, allow_missing=False, num_retries=0, retry_delay_sec=2):
        while True:
            try:
                return self._get_path_if_exists_single(key, allow_missing=allow_missing)
            except:
                if num_retries <= 0:
                    raise
                num_retries -= 1
                time.sleep(retry_delay_sec)

    def _get_path_if_exists_single(self, key, allow_missing=False):
        p = Path(os.path.join(self.base_dir, key))
        if not p.exists() and allow_missing:
            return None
        elif not p.exists() and not allow_missing:
            raise KeyError(p + " not found in local storage")
        return p

    def blob_path(self, key):
        return os.path.join(self.base_dir, key)

    def search(self, prefix="", suffix=""):
        files = []
        for root, dirs, files in os.walk(self.base_dir, topdown=False):
            common_prefix_len = min(len(prefix), len(root))
            if root[:common_prefix_len] != prefix[:common_prefix_len]:
                continue

            for name in files:
                filename = os.path.join(root, name)
                if filename.startswith(prefix) and filename.endswith(suffix):
                    files.append(filename)
        return files

    def _put_str(self, str_val, key):
        f = self._get_or_create_path(key)
        f.write_text(str_val)

    def put_str(self, str_val, key):
        self._put_str(str_val, key)

    def put_json(self, obj, key):
        self._put_str(json.dumps(obj), key)

    def get_json(self, key, allow_missing=False, num_retries=0, retry_delay_sec=2):
        f = self._get_path_if_exists(
            key,
            allow_missing=allow_missing,
            num_retries=num_retries,
            retry_delay_sec=retry_delay_sec,
        )
        if f is None:
            return None
        return json.loads(f.read_text())

    def put_object(self, body: bytes, key):
        f = self._get_or_create_path(key)
        f.write_bytes(body)

    def put_msgpack(self, obj, key):
        f = self._get_or_create_path(key)
        f.write_bytes(msgpack.dumps(obj))

    def get_msgpack(self, key, allow_missing=False, num_retries=0, retry_delay_sec=2):
        f = self._get_path_if_exists(
            key,
            allow_missing=allow_missing,
            num_retries=num_retries,
            retry_delay_sec=retry_delay_sec,
        )
        if f is None:
            return None
        return msgpack.loads(f.read_bytes())

    def upload_file(self, local_path, key):
        shutil.copy(local_path, str(self._get_or_create_path(key)))

    def download_file(self, key, local_path):
        try:
            Path(local_path).parent.mkdir(parents=True, exist_ok=True)
            shutil.copy(str(self._get_path(key)), local_path)
        except Exception as e:
            raise CortexException("file not found", key) from e

    def download_and_unzip(self, key, local_dir):
        util.mkdir_p(local_dir)
        local_zip = os.path.join(local_dir, "zip.zip")
        self.download_file(key, local_zip)
        util.extract_zip(local_zip, delete_zip_file=True)
