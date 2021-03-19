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
devstart-aws:
	@$(MAKE) operator-stop-aws || true
	@./dev/operator_local.sh -p aws || true
devstart-gcp:
	@$(MAKE) operator-stop-gcp || true
	@./dev/operator_local.sh -p gcp || true

cli:
	@mkdir -p ./bin
	@go build -o ./bin/cortex ./cli

# build cli and watch for changes
cli-watch:
	@rerun -watch ./pkg ./cli -run sh -c "clear && echo 'building cli...' && go build -o ./bin/cortex ./cli && clear && echo '\033[1;32mCLI built\033[0m'" || true

# start local operator and watch for changes
operator-local-aws:
	@$(MAKE) operator-stop-aws || true
	@./dev/operator_local.sh --operator-only -p aws || true
operator-local-gcp:
	@$(MAKE) operator-stop-gcp || true
	@./dev/operator_local.sh --operator-only -p gcp || true

# start local operator and attach the delve debugger to it (in server mode)
operator-local-dbg-aws:
	@$(MAKE) operator-stop-aws || true
	@./dev/operator_local.sh --debug -p aws || true
operator-local-dbg-gcp:
	@$(MAKE) operator-stop-gcp || true
	@./dev/operator_local.sh --debug -p gcp || true

# configure kubectl to point to the cluster specified in dev/config/cluster-[aws|gcp].yaml
kubectl-aws:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | (grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true)
kubectl-gcp:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && gcloud container clusters get-credentials "$$CORTEX_CLUSTER_NAME" --project "$$CORTEX_PROJECT" --region "$$CORTEX_ZONE" 2> /dev/stdout 1> /dev/null | (grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true)

cluster-up-aws:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster up ./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws"
	@$(MAKE) kubectl-aws
cluster-up-gcp:
	@$(MAKE) images-all-gcp
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp up ./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp"
	@$(MAKE) kubectl-gcp

cluster-up-aws-y:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster up ./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --yes
	@$(MAKE) kubectl-aws
cluster-up-gcp-y:
	@$(MAKE) images-all-gcp
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp up ./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" --yes
	@$(MAKE) kubectl-gcp

cluster-down-aws:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster-aws.yaml
cluster-down-gcp:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster-gcp down --config=./dev/config/cluster-gcp.yaml

cluster-down-aws-y:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster-aws.yaml --yes
cluster-down-gcp-y:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster-gcp down --config=./dev/config/cluster-gcp.yaml --yes

cluster-info-aws:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster info --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --yes
cluster-info-gcp:
	@$(MAKE) images-manager-skip-push
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp info --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" --yes

cluster-configure-aws:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster configure ./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws"
# cluster-configure-gcp:
# 	@$(MAKE) images-all-gcp
# 	@$(MAKE) cli
# 	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
# 	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp configure ./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp"

cluster-configure-aws-y:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster configure ./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --yes
# cluster-configure-gcp-y:
# 	@$(MAKE) images-all-gcp
# 	@$(MAKE) cli
# 	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
# 	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp configure ./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" --yes

# stop the in-cluster operator
operator-stop-aws:
	@$(MAKE) kubectl-aws
	@kubectl scale --namespace=default deployments/operator --replicas=0
operator-stop-gcp:
	@$(MAKE) kubectl-gcp
	@kubectl scale --namespace=default deployments/operator --replicas=0

# start the in-cluster operator
operator-start-aws:
	@$(MAKE) kubectl-aws
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default
operator-start-gcp:
	@$(MAKE) kubectl-gcp
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# restart the in-cluster operator
operator-restart-aws:
	@$(MAKE) kubectl-aws
	@kubectl delete pods -l workloadID=operator --namespace=default
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default
operator-restart-gcp:
	@$(MAKE) kubectl-gcp
	@kubectl delete pods -l workloadID=operator --namespace=default
	@operator_pod=$$(kubectl get pods -l workloadID=operator -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# build and update the in-cluster operator
operator-update-aws:
	@$(MAKE) kubectl-aws
	@kubectl scale --namespace=default deployments/operator --replicas=0
	@./dev/registry.sh update-single operator -p aws
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default
operator-update-gcp:
	@$(MAKE) kubectl-gcp
	@kubectl scale --namespace=default deployments/operator --replicas=0
	@./dev/registry.sh update-single operator -p gcp
	@kubectl scale --namespace=default deployments/operator --replicas=1
	@operator_pod=$$(kubectl get pods -l workloadID=operator --namespace=default -o jsonpath='{.items[0].metadata.name}') && kubectl wait --for=condition=ready pod $$operator_pod --namespace=default

# Docker images

images-all-skip-push:
	@./dev/registry.sh update all
images-all-aws:
	@./dev/registry.sh update all -p aws
images-all-gcp:
	@./dev/registry.sh update all -p gcp

images-dev-skip-push:
	@./dev/registry.sh update dev
images-dev-aws:
	@./dev/registry.sh update dev -p aws
images-dev-gcp:
	@./dev/registry.sh update dev -p gcp

images-api-skip-push:
	@./dev/registry.sh update api
images-api-aws:
	@./dev/registry.sh update api -p aws
images-api-gcp:
	@./dev/registry.sh update api -p gcp

images-manager-skip-push:
	@./dev/registry.sh update-single manager
images-iris-aws:
	@./dev/registry.sh update-single python-predictor-cpu -p aws
images-iris-gcp:
	@./dev/registry.sh update-single python-predictor-cpu -p gcp

registry-create-aws:
	@./dev/registry.sh create -p aws

registry-clean-aws:
	@./dev/registry.sh clean -p aws

# Misc

tools:
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/kyoh86/looppointer/cmd/looppointer
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@go get -u -v github.com/go-delve/delve/cmd/dlv
	@if [[ "$$OSTYPE" == "darwin"* ]]; then brew install parallel; elif [[ "$$OSTYPE" == "linux"* ]]; then sudo apt-get install -y parallel; else echo "your operating system is not supported"; fi
	@python3 -m pip install aiohttp black 'pydoc-markdown>=3.0.0,<4.0.0'
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

# run e2e tests on existing cluster
# read test/e2e/README.md for instructions first
test-e2e:
	@$(MAKE) test-e2e-aws
	@$(MAKE) test-e2e-gcp
test-e2e-aws:
	@$(MAKE) images-all-aws
	@$(MAKE) operator-restart-aws
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e -p aws -e "$$CORTEX_CLUSTER_NAME-aws"
test-e2e-gcp:
	@$(MAKE) images-all-gcp
	@$(MAKE) operator-restart-gcp
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e -p gcp -e "$$CORTEX_CLUSTER_NAME-gcp"

# run e2e tests with new clusters
# read test/e2e/README.md for instructions first
test-e2e-new:
	@$(MAKE) test-e2e-new-aws
	@$(MAKE) test-e2e-new-gcp
test-e2e-new-aws:
	@$(MAKE) images-all-aws
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e "$$(pwd)/dev/config/cluster-aws.yaml" -p aws --create-cluster
test-e2e-new-gcp:
	@$(MAKE) images-all-gcp
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && CORTEX_CLI_PATH="$$(pwd)/bin/cortex" ./build/test.sh e2e "$$(pwd)/dev/config/cluster-gcp.yaml" -p gcp --create-cluster

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
	@./build/push-images.sh

ci-build-cli:
	@./build/cli.sh

ci-build-and-upload-cli:
	@./build/cli.sh upload
