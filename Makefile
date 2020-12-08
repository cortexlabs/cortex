#!make

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
	@./dev/operator_local.sh --aws || true
devstart-gcp:
	@$(MAKE) operator-stop-gcp || true
	@./dev/operator_local.sh --gcp || true

cli:
	@mkdir -p ./bin
	@go build -o ./bin/cortex ./cli

# build cli and watch for changes
cli-watch:
	@rerun -watch ./pkg ./cli -run sh -c "clear && echo 'building cli...' && go build -o ./bin/cortex ./cli && clear && echo '\033[1;32mCLI built\033[0m'" || true

# start local operator and watch for changes
operator-local-aws:
	@$(MAKE) operator-stop-aws || true
	@./dev/operator_local.sh --operator-only --aws || true
operator-local-gcp:
	@$(MAKE) operator-stop-gcp || true
	@./dev/operator_local.sh --operator-only --gcp || true

# start local operator and attach the delve debugger to it (in server mode)
operator-local-dbg-aws:
	@$(MAKE) operator-stop || true
	@./dev/operator_local_debugger.sh --aws || true
operator-local-dbg-gcp:
	@$(MAKE) operator-stop || true
	@./dev/operator_local_debugger.sh --gcp || true

# configure kubectl to point to the cluster specified in dev/config/cluster-[aws|gcp].yaml
kubectl-aws:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true
kubectl-gcp:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && gcloud container clusters get-credentials "$$CORTEX_CLUSTER_NAME" --zone "$$CORTEX_ZONE" --project "$$CORTEX_PROJECT" 2>&1 | grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true

# configure kubectl to point to the cluster specified in dev/config/cluster.yaml
.PHONY: kubectl
kubectl:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

cluster-up-aws:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster up --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-aws"
	@$(MAKE) kubectl-aws
cluster-up-gcp:
	@$(MAKE) images-all-gcp
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp up --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-gcp"
	@$(MAKE) kubectl-gcp

cluster-up-aws-y:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster up --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-aws"
	@$(MAKE) kubectl-aws
cluster-up-gcp-y:
	@$(MAKE) images-all-gcp
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp up --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-gcp"
	@$(MAKE) kubectl-gcp

cluster-down-aws:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster-aws.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY
cluster-down-gcp:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster-gcp down --config=./dev/config/cluster-gcp.yaml

cluster-down-aws-y:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster-aws.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --yes
cluster-down-gcp-y:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster-gcp down --config=./dev/config/cluster-gcp.yaml --yes

cluster-info-aws:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster info --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-aws"
cluster-info-gcp:
	@$(MAKE) images-manager-local
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp info --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-gcp"

cluster-configure-aws:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster configure --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-aws"
# cluster-configure-gcp:
# 	@$(MAKE) images-all-gcp
# 	@$(MAKE) cli
# 	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
# 	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp configure --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-gcp"

cluster-configure-aws-y:
	@$(MAKE) images-all-aws
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-aws.yaml) && ./bin/cortex cluster configure --config=./dev/config/cluster-aws.yaml --configure-env="$$CORTEX_CLUSTER_NAME-aws" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-aws"
# cluster-configure-gcp-y:
# 	@$(MAKE) images-all-gcp
# 	@$(MAKE) cli
# 	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
# 	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster-gcp.yaml) && ./bin/cortex cluster-gcp configure --config=./dev/config/cluster-gcp.yaml --configure-env="$$CORTEX_CLUSTER_NAME-gcp" --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME-gcp"

# stop the in-cluster operator
operator-stop-aws:
	@$(MAKE) kubectl-aws
	@kubectl delete --namespace=default --ignore-not-found=true deployment operator
operator-stop-gcp:
	@$(MAKE) kubectl-gcp
	@kubectl delete --namespace=default --ignore-not-found=true deployment operator

# Docker images

images-all-aws:
	@./dev/registry.sh update all -p aws
images-all-gcp:
	@./dev/registry.sh update all -p gcp
images-all-local:
	@./dev/registry.sh update all -p local
images-all-slim-aws:
	@./dev/registry.sh update all -p aws --include-slim
images-all-slim-gcp:
	@./dev/registry.sh update all -p gcp --include-slim
images-all-slim-local:
	@./dev/registry.sh update all -p local --include-slim

images-dev-aws:
	@./dev/registry.sh update dev -p aws
images-dev-gcp:
	@./dev/registry.sh update dev -p gcp
images-dev-local:
	@./dev/registry.sh update dev -p local
images-dev-slim-aws:
	@./dev/registry.sh update dev -p aws --include-slim
images-dev-slim-gcp:
	@./dev/registry.sh update dev -p gcp --include-slim
images-dev-slim-local:
	@./dev/registry.sh update dev -p local --include-slim

images-api-aws:
	@./dev/registry.sh update api -p aws
images-api-gcp:
	@./dev/registry.sh update api -p gcp
images-api-local:
	@./dev/registry.sh update api -p local
images-api-slim-aws:
	@./dev/registry.sh update api -p aws --include-slim
images-api-slim-gcp:
	@./dev/registry.sh update api -p gcp --include-slim
images-api-slim-local:
	@./dev/registry.sh update api -p local --include-slim

images-manager-local:
	@./dev/registry.sh update-single manager -p local
images-iris-local:
	@./dev/registry.sh update-single python-predictor-cpu -p local
images-iris-aws:
	@./dev/registry.sh update-single python-predictor-cpu -p aws
images-iris-gcp:
	@./dev/registry.sh update-single python-predictor-cpu -p gcp

registry-create-aws:
	@./dev/registry.sh create -p aws

registry-clean-aws:
	@./dev/registry.sh clean -p aws
registry-clean-local:
	@./dev/registry.sh clean -p local

# Misc

tools:
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@go get -u -v github.com/go-delve/delve/cmd/dlv
	@python3 -m pip install black 'pydoc-markdown>=3.0.0,<4.0.0'
	@if [[ "$$OSTYPE" == "darwin"* ]]; then brew install parallel; elif [[ "$$OSTYPE" == "linux"* ]]; then sudo apt-get install -y parallel; else echo "your operating system is not supported"; fi

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
