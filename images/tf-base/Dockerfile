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


FROM ubuntu:18.04

ARG TF_VERSION="1.12.0"

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        curl \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        python3 \
        python3-dev \
        python3-pip \
        rsync \
        software-properties-common \
        unzip \
        zlib1g-dev \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN pip3 install https://storage.googleapis.com/tensorflow/linux/cpu/tensorflow-${TF_VERSION}-cp36-cp36m-linux_x86_64.whl && rm -rf /root/.cache/pip*
