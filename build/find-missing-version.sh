#!/bin/bash

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

command='grep --exclude="check-cortex-version.sh" -R -A 50 -e "CORTEX_VERSION" | grep -e "master"'


if [[ $1 == "test" ]] && [[ $(eval $command) ]]; then
	echo "there are still CORTEX_VERSION references to master. run 'make find-missing-version' to find offending files."; \
    exit 1
fi

eval $command
