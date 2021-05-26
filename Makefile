#!make

# Copyright 2021 Cortex Labs, Inc.
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
export BASH_ENV=./dev/config/env.sh

# declare all targets as phony to avoid collisions with local files or folders
.PHONY: $(MAKECMDGOALS)

#######
# Dev #
#######

# Cortex

# build cli, start local operator, and watch for changes
devstart:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh || true

cli:
	@mkdir -p ./bin
	@go build -o ./bin/cortex ./cli

# build cli and watch for changes
cli-watch:
	@rerun -watch ./pkg ./cli -run sh -c "clear && echo 'building cli...' && go build -o ./bin/cortex ./cli && clear && echo '\033[1;32mCLI built\033[0m'" || true

# start local operator and watch for changes
operator-local:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh --operator-only || true

# start local operator and attach the delve debugger to it (in server mode)
operator-local-dbg:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh --debug || true

# configure kubectl to point to the cluster specified in dev/config/cluster.yaml
kubectl:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | (grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true); eksctl create iamidentitymapping --region $$CORTEX_REGION --cluster $$CORTEX_CLUSTER_NAME --arn $$DEFAULT_USER_ARN --group system:masters --username $$DEFAULT_USER_ARN

cluster-up:
	@$(MAKE) images-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && sleep 10 && ./bin/cortex cluster up ./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME"; eksctl create iamidentitymapping --region $$CORTEX_REGION --cluster $$CORTEX_CLUSTER_NAME --arn $$DEFAULT_USER_ARN --group system:masters --username $$DEFAULT_USER_ARN

	@$(MAKE) kubectl

cluster-up-y:
	@$(MAKE) images-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && sleep 10 && ./bin/cortex cluster up ./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --yes; eksctl create iamidentitymapping --region $$CORTEX_REGION --cluster $$CORTEX_CLUSTER_NAME --arn $$DEFAULT_USER_ARN --group system:masters --username $$DEFAULT_USER_ARN
	@$(MAKE) kubectl

cluster-down:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && sleep 10 && ./bin/cortex cluster down --config=./dev/config/cluster.yaml

cluster-down-y:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && sleep 10 && ./bin/cortex cluster down --config=./dev/config/cluster.yaml --yes

cluster-info:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eval $$(python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION) && sleep 10 && ./bin/cortex cluster info --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --yes

update-credentials:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && python3 ./dev/create_user.py $$CORTEX_CLUSTER_NAME $$AWS_ACCOUNT_ID $$CORTEX_REGION

# stop the in-cluster operator
operator-stop:
	@$(MAKE) kubectl
	@kubectl scale --namespace=default deployments/operator --replicas=0

# start the in-cluster operator
operator-start:
	@$(MAKE) kubectl
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# restart the in-cluster operator
operator-restart:
	@$(MAKE) kubectl
	@kubectl delete pods -l workloadID=operator --namespace=default
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# build and update the in-cluster operator
operator-update:
	@$(MAKE) kubectl
	@kubectl scale --namespace=default deployments/operator --replicas=0
	@./dev/registry.sh update-single operator
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# restart all in-cluster async-gateways
async-gateway-restart:
	@$(MAKE) kubectl
	@kubectl delete pods -l cortex.dev/async=gateway --namespace=default

# build and update all in-cluster async-gateways
async-gateway-update:
	@$(MAKE) kubectl
	@./dev/registry.sh update-single async-gateway
	@kubectl delete pods -l cortex.dev/async=gateway --namespace=default

# docker images
images-all:
	@./dev/registry.sh update all
images-all-skip-push:
	@./dev/registry.sh update all --skip-push

images-dev:
	@./dev/registry.sh update dev
images-dev-skip-push:
	@./dev/registry.sh update dev --skip-push

images-manager-skip-push:
	@./dev/registry.sh update-single manager --skip-push

registry-create:
	@./dev/registry.sh create

registry-clean:
	@./dev/registry.sh clean

# Misc

tools:
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/kyoh86/looppointer/cmd/looppointer
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@go get -u -v github.com/go-delve/delve/cmd/dlv
	@if [[ "$$OSTYPE" == "darwin"* ]]; then brew install parallel; elif [[ "$$OSTYPE" == "linux"* ]]; then sudo apt-get install -y parallel; else echo "your operating system is not supported"; fi
	@python3 -m pip install aiohttp black 'pydoc-markdown>=3.0.0,<4.0.0' boto3 pyyaml
	@python3 -m pip install -e test/e2e

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


# test api docker images
test-images:
	@./test/dev/registry.sh update
test-images-skip-push:
	@./test/dev/registry.sh update --skip-push

test-registry-create:
	@./test/dev/registry.sh create

test-registry-clean:
	@./test/dev/registry.sh clean

# run e2e tests on an existing cluster
# read test/e2e/README.md for instructions first
test-e2e:
	@$(MAKE) images-all
	@$(MAKE) operator-restart
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e -e "$$CORTEX_CLUSTER_NAME"

# run e2e tests with a new cluster
# read test/e2e/README.md for instructions first
test-e2e-new:
	@$(MAKE) images-all
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e "$$(pwd)/dev/config/cluster.yaml" --create-cluster

lint:
	@./build/lint.sh

# this is a subset of lint.sh, and is only meant to be run on master
lint-docs:
	@./build/lint-docs.sh

###############
# CI Commands #
###############

ci-build-images:
	@./build/build-images.sh

ci-push-images:
	@./build/push-images.sh quay.io

ci-backup-images:
	@./build/push-images.sh docker.io

ci-build-cli:
	@./build/cli.sh

ci-build-and-upload-cli:
	@./build/cli.sh upload

ci-test-build-images:
	@./test/dev/build-images.sh

ci-test-push-images:
	@./test/dev/push-images.sh quay.io
