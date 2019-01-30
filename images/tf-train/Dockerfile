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


FROM cortexlabs/tf-base

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
COPY pkg/workloads/tf_train/requirements.txt /src/tf_train/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/tf_train/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/tf_train /src/tf_train

ENTRYPOINT ["/usr/bin/python3", "/src/tf_train/train.py"]
