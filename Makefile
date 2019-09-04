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

# Cortex

devstart:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh || true

kubectl:
	@eksctl utils write-kubeconfig --name="cortex" | grep -v "saved kubeconfig as" | grep -v "using region" || true
	@kubectl config set-context --current --namespace="cortex" >/dev/null

cortex-up:
	@$(MAKE) registry-all
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh install
	@$(MAKE) kubectl

cortex-up-dev:
	@$(MAKE) cortex-up
	@$(MAKE) operator-stop

cortex-down:
	@$(MAKE) manager-local
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh uninstall

cortex-install:
	@$(MAKE) registry-all
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh install cortex
	@$(MAKE) kubectl

cortex-uninstall:
	@./dev/uninstall_cortex.sh

cortex-info:
	@$(MAKE) manager-local
	@./cortex.sh -c=./dev/config/cortex.sh info

cortex-update:
	@$(MAKE) registry-all
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh update

operator-start:
	@$(MAKE) registry-all
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh update

operator-update:
	@$(MAKE) registry-all
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./cortex.sh -c=./dev/config/cortex.sh update

operator-stop:
	@$(MAKE) kubectl
	@kubectl delete --namespace="cortex" --ignore-not-found=true deployment operator

# Docker images

registry-all:
	@./dev/registry.sh update

registry-dev:
	@./dev/registry.sh update dev

registry-create:
	@./dev/registry.sh create

manager-local:
	@./dev/registry.sh update-manager-local

# Misc

.PHONY: cli
cli:
	@mkdir -p ./bin
	@GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/cortex ./cli

cli-watch:
	@rerun -watch ./pkg ./cli -ignore ./vendor ./bin -run sh -c "go build -installsuffix cgo -o ./bin/cortex ./cli && echo 'CLI built.'"

aws-clear-bucket:
	@./dev/aws.sh clear-bucket

tools:
	@go get -u -v golang.org/x/lint/golint
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
	@$(MAKE) registry-all
	@./build/test-examples.sh

###############
# CI Commands #
###############

ci-build-images:
	@./build/build-image.sh images/manager manager
	@./build/build-image.sh images/tf-serve tf-serve
	@./build/build-image.sh images/tf-serve-gpu tf-serve-gpu
	@./build/build-image.sh images/tf-api tf-api
	@./build/build-image.sh images/onnx-serve onnx-serve
	@./build/build-image.sh images/onnx-serve-gpu onnx-serve-gpu
	@./build/build-image.sh images/operator operator
	@./build/build-image.sh images/fluentd fluentd
	@./build/build-image.sh images/statsd statsd
	@./build/build-image.sh images/cluster-autoscaler cluster-autoscaler
	@./build/build-image.sh images/nvidia nvidia
	@./build/build-image.sh images/metrics-server metrics-server
	@./build/build-image.sh images/istio-citadel istio-citadel
	@./build/build-image.sh images/istio-galley istio-galley
	@./build/build-image.sh images/istio-pilot istio-pilot
	@./build/build-image.sh images/istio-proxy istio-proxy
	@./build/build-image.sh images/downloader downloader

ci-push-images:
	@./build/push-image.sh manager
	@./build/push-image.sh tf-serve
	@./build/push-image.sh tf-serve-gpu
	@./build/push-image.sh tf-api
	@./build/push-image.sh onnx-serve
	@./build/push-image.sh onnx-serve-gpu
	@./build/push-image.sh operator
	@./build/push-image.sh fluentd
	@./build/push-image.sh statsd
	@./build/push-image.sh cluster-autoscaler
	@./build/push-image.sh nvidia
	@./build/push-image.sh metrics-server
	@./build/push-image.sh istio-citadel
	@./build/push-image.sh istio-galley
	@./build/push-image.sh istio-pilot
	@./build/push-image.sh istio-proxy
	@./build/push-image.sh downloader


ci-build-cli:
	@./build/cli.sh

ci-build-and-upload-cli:
	@./build/cli.sh upload
