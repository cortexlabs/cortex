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

import sys
import os


def main():
    in_file = sys.argv[1]
    out_file = sys.argv[2]

    with open(in_file, "r") as f:
        data = f.read()

    expanded_data = os.path.expandvars(data)

    with open(out_file, "w") as f:
        f.write(expanded_data)


if __name__ == "__main__":
    main()
