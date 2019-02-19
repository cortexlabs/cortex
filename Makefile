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
SHELL := /bin/bash
GOBIN ?= $$PWD/bin
GOOS ?= linux
define build
    mkdir -p $(GOBIN) && \
		GOOS=$(GOOS) \
		GOARCH=amd64 \
		CGO_ENABLED=0 \
		GOBIN=$(GOBIN) \
		GO111MODULE=on \
		go install -v \
		-gcflags='-e' \
		$(1)
endef


#####################
# Operator commands #
#####################
olocal:
	@./dev/operator_local.sh

ostop:
	@.kubectl -n=cortex delete --ignore-not-found=true deployment operator

opudate:
	@./cortex.sh -c=./dev/config/cortex.sh update operator

oinstall:
	@./cortex.sh -c=./dev/config/cortex.sh install operator

ouninstall:
	@./cortex.sh -c=./dev/config/cortex.sh uninstall operator


eks-up:
	@./dev/eks.sh start
	$(MAKE) oinstall

eks-down:
	$(MAKE) ouninstall || true
	@./dev/eks.sh stop

eks-set:
	@./dev/eks.sh set


kops-up:
	@./dev/kops.sh start
	@./cortex.sh -c=dev/config/cortex.sh install operator 
	@kubectl -n=cortex delete --ignore-not-found=true deployment operator
	
kops-down:
	@./dev/kops.sh stop

kops-set:
	@./dev/kops.sh set

tools:
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun

build-cli:
	 @$(call build, ./cli)

test:
	@./build/test.sh

find-missing-license:
	@./build/find-missing-license.sh

find-missing-version:
	@./build/check-cortex-version.sh
	
###############
# CI Commands #
###############
build-spark-base:
	@docker build . -f images/spark-base/Dockerfile -t cortexlabs/spark-base:latest
	
build-tf-base:
	@docker build . -f images/tf-base/Dockerfile -t cortexlabs/tf-base:latest
	@docker build . -f images/tf-base-gpu/Dockerfile -t cortexlabs/tf-base-gpu:latest

build-base: spark-base tf-base

build-tf-dev:
	@./build/build-image.sh images/tf-train tf-train
	@./build/build-image.sh images/tf-serve tf-serve
	@./build/build-image.sh images/tf-api tf-api
	@./build/build-image.sh images/tf-train-gpu tf-train-gpu
	@./build/build-image.sh images/tf-serve-gpu tf-serve-gpu

build-spark-dev:
	@./build/build-image.sh images/spark spark
	@./build/build-image.sh images/spark-operator spark-operator

build-tf-images: build-tf-base build-tf-dev
build-spark-images: build-spark-base build-spark-dev

build-argo-images:
	@./build/build-image.sh images/argo-controller argo-controller
	@./build/build-image.sh images/argo-executor argo-executor

build-operator-images:
	@./build/build-image.sh images/operator operator
	@./build/build-image.sh images/nginx-controller nginx-controller
	@./build/build-image.sh images/nginx-backend nginx-backend
	@./build/build-image.sh images/fluentd fluentd

build-images: build-tf-images build-spark-images build-argo-images build-operator-images

push-tf-dev:
	@./build/push-image.sh images/tf-train tf-train
	@./build/push-image.sh images/tf-serve tf-serve
	@./build/push-image.sh images/tf-api tf-api
	@./build/push-image.sh images/tf-train-gpu tf-train-gpu
	@./build/push-image.sh images/tf-serve-gpu tf-serve-gpu

push-spark-dev:
	@./build/push-image.sh images/spark spark
	@./build/push-image.sh images/spark-operator spark-operator

push-tf-images: push-tf-base push-tf-dev
push-spark-images: push-spark-base push-spark-dev

push-argo-images:
	@./build/push-image.sh images/argo-controller argo-controller
	@./build/push-image.sh images/argo-executor argo-executor

push-operator-images:
	@./build/push-image.sh images/operator operator
	@./build/push-image.sh images/nginx-controller nginx-controller
	@./build/push-image.sh images/nginx-backend nginx-backend
	@./build/push-image.sh images/fluentd fluentd

push-images: push-tf-images push-spark-images push-argo-images push-operator-images

build-and-upload-cli:
	@./build/cli.sh

test-go:
	@./build/test.sh go

test-python:
	@./build/test.sh python

test-license:
	@if [ "$$(./build/find-missing-license.sh)" ]; then echo "some files are missing license headers, run 'make find-missing-license' to find offending files."; exit 1; fi

test-version:
	@if [ "$$(./build/find-missing-version.sh)" ]; then echo "there are still CORTEX_VERSION references to master. run 'make find-missing-version'"; exit 1; fi
