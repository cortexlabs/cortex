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

import logging
from cortex.lib import stringify

logger = logging.getLogger("cortex")
handler = logging.StreamHandler()
handler.setFormatter(
    logging.Formatter("%(asctime)s:%(name)s:%(levelname)s:%(message)s", "'%Y-%m-%d %H:%M:%S.%f'")
)
logger.addHandler(handler)
logger.setLevel(logging.DEBUG)


def debug_obj(name, sample, debug):
    if not debug:
        return

    logger.info("{}: {}".format(name, stringify.truncate(sample)))


def get_logger():
    return logging.getLogger("cortex")
