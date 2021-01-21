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

from contextlib import contextmanager
import os
from pathlib import Path
import shutil


@contextmanager
def open_temporarily(path, mode):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    file = open(path, mode)

    try:
        yield file
    finally:
        file.close()
        os.remove(path)


@contextmanager
def open_tempdir(dir_path):
    Path(dir_path).mkdir(parents=True)

    try:
        yield dir_path
    finally:
        shutil.rmtree(dir_path)
