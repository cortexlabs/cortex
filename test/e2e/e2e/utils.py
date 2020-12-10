# Copyright 2020 Cortex Labs, Inc.
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

import time
from typing import List, Optional, Dict, Union

import cortex as cx
import requests


def apis_ready(client: cx.Client, api_names: List[str], timeout: Optional[int] = None) -> bool:
    count = 0
    while True:
        if timeout is not None and count > timeout:
            return False

        ready = all(
            [client.get_api(name)["status"]["status_code"] == "status_live" for name in api_names]
        )
        if ready:
            return True

        time.sleep(1)
        count += 1


def request_prediction(
    client: cx.Client, api_name: str, payload: Union[List, Dict]
) -> requests.Response:

    api_info = client.get_api(api_name)
    response = requests.post(api_info["endpoint"], json=payload)

    return response
