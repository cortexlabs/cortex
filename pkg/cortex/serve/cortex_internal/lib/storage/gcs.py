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

import os
import pathlib
import datetime
from typing import Optional, List, Tuple

from google.cloud import storage
from google.cloud import exceptions as gexp

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import CortexException


class GCS:
    def __init__(self, bucket: str, project: Optional[str] = None):
        if os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
            self.client = storage.Client(project=project)
        else:
            self.client = storage.Client.create_anonymous_client()
        self.gcs = self.client.bucket(bucket, project)

    @staticmethod
    def construct_gcs_path(bucket_name: str, prefix: str) -> str:
        return f"gs://{bucket_name}/{prefix}"

    @staticmethod
    def deconstruct_gcs_path(gcs_path: str) -> Tuple[str, str]:
        path = util.trim_prefix(gcs_path, "gs://")
        bucket = path.split("/")[0]
        key = os.path.join(*path.split("/")[1:])
        return bucket, key

    @staticmethod
    def is_valid_gcs_path(path: str) -> bool:
        if not path.startswith("gs://"):
            return False
        parts = path[5:].split("/")
        if len(parts) < 2:
            return False
        if parts[0] == "" or parts[1] == "":
            return False
        return True

    def _does_blob_exist(self, prefix: str) -> bool:
        return isinstance(self.gcs.get_blob(blob_name=prefix), storage.blob.Blob)

    def _gcs_matching_blobs_generator(self, max_results=None, prefix="", include_dir_objects=False):
        for blob in self.gcs.list_blobs(max_results, prefix=prefix):
            if include_dir_objects or not blob.name.endswith("/"):
                yield blob

    def _is_gcs_dir(self, dir_path: str) -> bool:
        prefix = util.ensure_suffix(dir_path, "/")
        matching_blobs = list(
            self._gcs_matching_blobs_generator(
                max_results=2, prefix=prefix, include_dir_objects=True
            )
        )

        return len(matching_blobs) > 0

    def search(
        self, prefix: str = "", include_dir_objects=False
    ) -> Tuple[List[str], List[datetime.datetime]]:
        paths = []
        timestamps = []

        for blob in self._gcs_matching_blobs_generator(
            prefix=prefix, include_dir_objects=include_dir_objects
        ):
            paths.append(blob.name)
            timestamps.append(blob.updated)

        return paths, timestamps

    def upload_file(self, local_path: str, key: str):
        """
        Upload file to bucket.
        """
        if not pathlib.Path(local_path).is_file():
            raise CortexException(f'file "{key}" doesn\'t exist')

        blob = self.gcs.get_blob(blob_name=key)
        blob.upload_from_filename(local_path)

    def download_file(self, key: str, local_path: str):
        """
        Download file to the specified local path.
        """
        blob = self.gcs.get_blob(blob_name=key)
        if not isinstance(blob, storage.blob.Blob):
            raise CortexException(f'key "{key}" in bucket "{self.gcs.name}" not found')

        util.mkdir_p(os.path.dirname(local_path))
        try:
            blob.download_to_filename(local_path)
        except gexp.NotFound:
            raise CortexException(f'key "{key}" in bucket "{self.gcs.name}" not found')

    def download_file_to_dir(self, key: str, local_dir_path: str):
        """
        Download a file inside a local directory.
        """
        filename = os.path.basename(key)
        self.download_file(key, os.path.join(local_dir_path, filename))

    def download_dir(self, prefix: str, local_dir: str):
        """
        Download directory inside the specified local directory.
        """
        dir_name = util.trim_suffix(prefix, "/").split("/")[-1]
        self.download_dir_contents(prefix, os.path.join(local_dir, dir_name))

    def download_dir_contents(self, prefix: str, local_dir: str):
        util.mkdir_p(local_dir)
        prefix = util.ensure_suffix(prefix, "/")
        for blob in self._gcs_matching_blobs_generator(prefix=prefix, include_dir_objects=True):
            relative_path = util.trim_prefix(blob.name, prefix)
            local_dest_path = os.path.join(local_dir, relative_path)

            if not local_dest_path.endswith("/"):
                self.download_file(blob.name, local_dest_path)
            else:
                util.mkdir_p(os.path.dirname(local_dest_path))

    def download(self, prefix: str, local_dir: str):
        """
        Download the object(s) with the given prefix to a local directory.
        """
        if self._is_gcs_dir(prefix):
            self.download_dir(prefix, local_dir)
        else:
            self.download_file_to_dir(prefix, local_dir)
