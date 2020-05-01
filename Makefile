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

SHELL := /bin/bash

#######
# Dev #
#######

# Cortex

devstart:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh || true

kubectl:
	@eval $$(python ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

cluster-up:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster up
	@$(MAKE) kubectl

cluster-up-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster up --yes
	@$(MAKE) kubectl

cluster-down:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster down

cluster-down-y:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster down --yes

cluster-info:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@./bin/cortex -c=./dev/config/cluster.yaml cluster info

cluster-update:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster update

cluster-update-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex -c=./dev/config/cluster.yaml cluster update --yes

operator-stop:
	@$(MAKE) kubectl
	@kubectl delete --namespace=default --ignore-not-found=true deployment operator

# Docker images

registry-all:
	@./dev/registry.sh update

registry-dev:
	@./dev/registry.sh update dev

registry-create:
	@./dev/registry.sh create

registry-clean:
	@./dev/registry.sh clean

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

test-examples:
	@$(MAKE) registry-all
	@./build/test-examples.sh

###############
# CI Commands #
###############

ci-build-images:
	@./build/build-image.sh images/python-predictor-cpu python-predictor-cpu
	@./build/build-image.sh images/python-predictor-gpu python-predictor-gpu
	@./build/build-image.sh images/tensorflow-serving-cpu tensorflow-serving-cpu
	@./build/build-image.sh images/tensorflow-serving-gpu tensorflow-serving-gpu
	@./build/build-image.sh images/tensorflow-predictor tensorflow-predictor
	@./build/build-image.sh images/onnx-predictor-cpu onnx-predictor-cpu
	@./build/build-image.sh images/onnx-predictor-gpu onnx-predictor-gpu
	@./build/build-image.sh images/operator operator
	@./build/build-image.sh images/manager manager
	@./build/build-image.sh images/downloader downloader
	@./build/build-image.sh images/request-monitor request-monitor
	@./build/build-image.sh images/cluster-autoscaler cluster-autoscaler
	@./build/build-image.sh images/metrics-server metrics-server
	@./build/build-image.sh images/nvidia nvidia
	@./build/build-image.sh images/fluentd fluentd
	@./build/build-image.sh images/statsd statsd
	@./build/build-image.sh images/istio-proxy istio-proxy
	@./build/build-image.sh images/istio-pilot istio-pilot
	@./build/build-image.sh images/istio-citadel istio-citadel
	@./build/build-image.sh images/istio-galley istio-galley

ci-push-images:
	@./build/push-image.sh python-predictor-cpu
	@./build/push-image.sh python-predictor-gpu
	@./build/push-image.sh tensorflow-serving-cpu
	@./build/push-image.sh tensorflow-serving-gpu
	@./build/push-image.sh tensorflow-predictor
	@./build/push-image.sh onnx-predictor-cpu
	@./build/push-image.sh onnx-predictor-gpu
	@./build/push-image.sh operator
	@./build/push-image.sh manager
	@./build/push-image.sh downloader
	@./build/push-image.sh request-monitor
	@./build/push-image.sh cluster-autoscaler
	@./build/push-image.sh metrics-server
	@./build/push-image.sh nvidia
	@./build/push-image.sh fluentd
	@./build/push-image.sh statsd
	@./build/push-image.sh istio-proxy
	@./build/push-image.sh istio-pilot
	@./build/push-image.sh istio-citadel
	@./build/push-image.sh istio-galley

ci-build-cli:
	@./build/cli.sh

ci-build-and-upload-cli:
	@./build/cli.sh upload
