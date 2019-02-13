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

###############
# Dev Helpers #
###############
GOBIN ?= $$PWD/bin
define build
    mkdir -p $(GOBIN)
    GOGC=off GOBIN=$(GOBIN) \
		go install -v \
		-gcflags='-e' \
		$(1)
endef

olocal:
	./dev/operator_local.sh

eks-up:
	./dev/eks.sh start
	$(MAKE) install

eks-down:
	./dev/eks.sh stop

install:
	./cortex.sh -c=./dev/config/cortex.sh install

tools:
	go get -u -v github.com/VojtechVitek/rerun/cmd/rerun

build-cli:
	 $(call build, ./cli)

test:
	./build/test.sh

find-missing-licenses:
	@./build/find-missing-license.sh
	
###############
# CI Commands #
###############
spark-base:
	docker build . -f images/spark-base/Dockerfile -t cortexlabs/spark-base:latest
	
tf-base:
	docker build . -f images/tf-base/Dockerfile -t cortexlabs/tf-base:latest

base: spark-base tf-base

tf-dev:
	./build/images.sh images/tf-train tf-train
	./build/images.sh images/tf-serve tf-serve
	./build/images.sh images/tf-api tf-api

spark-dev:
	./build/images.sh images/spark spark
	./build/images.sh images/spark-operator spark-operator

tf-images: tf-base tf-dev
spark-images: spark-base spark-dev

argo-images:
	./build/images.sh images/argo-controller argo-controller
	./build/images.sh images/argo-executor argo-executor

operator-images:
	./build/images.sh images/operator operator
	./build/images.sh images/nginx-controller nginx-controller
	./build/images.sh images/nginx-backend nginx-backend
	./build/images.sh images/fluentd fluentd

build-and-upload:
	./build/cli.sh

test-go:
	./build/test.sh go

test-python:
	./build/test.sh python

test-license:
	@if [ $(make find-missing-licenses) ]; then exit 1; fi
