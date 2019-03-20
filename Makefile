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

SHELL := /bin/bash

#######
# Dev #
#######

# Operator

olocal:
	@./dev/operator_local.sh || true

oinstall:
	@./cortex-installer.sh -c=./dev/config/cortex.sh install operator

oupdate:
	@./cortex-installer.sh -c=./dev/config/cortex.sh update operator

ouninstall:
	@./cortex-installer.sh -c=./dev/config/cortex.sh uninstall operator

ostop:
	@kubectl -n=cortex delete --ignore-not-found=true deployment operator

# EKS

eks-up:
	@./dev/eks.sh start
	$(MAKE) oinstall

eks-up-dev:
	$(MAKE) eks-up
	$(MAKE) ostop

eks-down:
	$(MAKE) ouninstall || true
	@./dev/eks.sh stop

eks-set:
	@./dev/eks.sh set

# KOPS

kops-up:
	@./dev/kops.sh start
	$(MAKE) oinstall

kops-up-dev:
	$(MAKE) kops-up
	$(MAKE) ostop

kops-down:
	@./dev/kops.sh stop

kops-set:
	@./dev/kops.sh set

# Docker images

registry-all:
	@./dev/registry.sh update

registry-dev:
	@./dev/registry.sh update dev

registry-create:
	@./dev/registry.sh create

# Misc

.PHONY: cli
cli:
	@mkdir -p ./bin
	@GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/cortex ./cli

aws-clear-bucket:
	@./dev/aws.sh clear-bucket

tools:
	@GO111MODULE=off go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@pip3 install black

format:
	@./dev/format.sh

#########
# Tests #
#########

test:
	@./build/test.sh

test-go:
	@./build/test.sh go

test-python:
	@./build/test.sh python

lint:
	@./build/lint.sh

find-missing-version:
	@./build/find-missing-version.sh

test-examples:
	$(MAKE) registry-all
	@./build/test-examples.sh

###############
# CI Commands #
###############

build-images:
	@./build/build-image.sh images/spark-base spark-base
	@./build/build-image.sh images/tf-base tf-base
	@./build/build-image.sh images/tf-base-gpu tf-base-gpu
	@./build/build-image.sh images/spark spark
	@./build/build-image.sh images/spark-operator spark-operator
	@./build/build-image.sh images/tf-train tf-train
	@./build/build-image.sh images/tf-train-gpu tf-train-gpu
	@./build/build-image.sh images/tf-serve tf-serve
	@./build/build-image.sh images/tf-serve-gpu tf-serve-gpu
	@./build/build-image.sh images/tf-api tf-api
	@./build/build-image.sh images/operator operator
	@./build/build-image.sh images/fluentd fluentd
	@./build/build-image.sh images/nginx-controller nginx-controller
	@./build/build-image.sh images/nginx-backend nginx-backend
	@./build/build-image.sh images/argo-controller argo-controller
	@./build/build-image.sh images/argo-executor argo-executor
	@./build/build-image.sh images/python-packager python-packager

push-images:
	@./build/push-image.sh spark
	@./build/push-image.sh spark-operator
	@./build/push-image.sh tf-train
	@./build/push-image.sh tf-train-gpu
	@./build/push-image.sh tf-serve
	@./build/push-image.sh tf-serve-gpu
	@./build/push-image.sh tf-api
	@./build/push-image.sh operator
	@./build/push-image.sh fluentd
	@./build/push-image.sh nginx-controller
	@./build/push-image.sh nginx-backend
	@./build/push-image.sh argo-controller
	@./build/push-image.sh argo-executor
	@./build/push-image.sh python-packager

build-cli:
	@./build/cli.sh

build-and-upload-cli:
	@./build/cli.sh upload
