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
-include ./dev/config/env.sh
export $(shell sed 's/=.*//' ./dev/config/env.sh 2>/dev/null)

#######
# Dev #
#######

# Cortex

# build cli, start local operator, and watch for changes
.PHONY: devstart
devstart:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh || true

.PHONY: cli
cli:
	@mkdir -p ./bin
	@go build -o ./bin/cortex ./cli

# build cli and watch for changes
.PHONY: cli-watch
cli-watch:
	@rerun -watch ./pkg ./cli -run sh -c "clear && echo 'building cli...' && go build -o ./bin/cortex ./cli && clear && echo '\033[1;32mCLI built\033[0m'" || true

# start local operator and watch for changes
.PHONY: operator-local
operator-local:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh --operator-only || true

# configure kubectl to point to the cluster specified in dev/config/cluster.yaml
.PHONY: kubectl
kubectl:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

.PHONY: cluster-up
cluster-up:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && ./bin/cortex cluster up --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME"
	@$(MAKE) kubectl

# TODO add GCP versions of all commands. auto-configure env like "cortex-gcp".

.PHONY: cluster-up-y
cluster-up-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && ./bin/cortex cluster up --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME"
	@$(MAKE) kubectl

.PHONY: cluster-down
cluster-down:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY

.PHONY: cluster-down-y
cluster-down-y:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@./bin/cortex cluster down --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --yes

.PHONY: cluster-info
cluster-info:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && ./bin/cortex cluster info --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME"

.PHONY: cluster-configure
cluster-configure:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && ./bin/cortex cluster configure --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME"

.PHONY: cluster-configure-y
cluster-configure-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && ./bin/cortex cluster configure --config=./dev/config/cluster.yaml --configure-env="$$CORTEX_CLUSTER_NAME" --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes && ./bin/cortex env default "$$CORTEX_CLUSTER_NAME"

# stop the in-cluster operator
.PHONY: operator-stop
operator-stop:
	@$(MAKE) kubectl
	@kubectl delete --namespace=default --ignore-not-found=true deployment operator

# Docker images

.PHONY: registry-all
registry-all:
	@./dev/registry.sh update all
.PHONY: registry-all-gcp
# TODO: revisit
registry-all-gcp:
	@./dev/registry-gcp.sh update all
.PHONY: registry-all-local
registry-all-local:
	@./dev/registry.sh update all --skip-push
.PHONY: registry-all-slim
registry-all-slim:
	@./dev/registry.sh update all --include-slim
.PHONY: registry-all-slim-local
registry-all-slim-local:
	@./dev/registry.sh update all --include-slim --skip-push
.PHONY: registry-all-local-slim
registry-all-local-slim:
	@./dev/registry.sh update all --include-slim --skip-push

.PHONY: registry-dev
registry-dev:
	@./dev/registry.sh update dev
.PHONY: registry-dev-gcp
registry-dev-gcp:
	@./dev/registry-gcp.sh update dev
.PHONY: registry-dev-local
registry-dev-local:
	@./dev/registry.sh update dev --skip-push
.PHONY: registry-dev-slim
registry-dev-slim:
	@./dev/registry.sh update dev --include-slim
.PHONY: registry-dev-slim-local
registry-dev-slim-local:
	@./dev/registry.sh update dev --include-slim --skip-push
.PHONY: registry-dev-local-slim
registry-dev-local-slim:
	@./dev/registry.sh update dev --include-slim --skip-push

.PHONY: registry-api
registry-api:
	@./dev/registry.sh update api
.PHONY: registry-api-local
registry-api-local:
	@./dev/registry.sh update api --skip-push
.PHONY: registry-api-slim
registry-api-slim:
	@./dev/registry.sh update api --include-slim
.PHONY: registry-api-slim-local
registry-api-slim-local:
	@./dev/registry.sh update api --include-slim --skip-push
.PHONY: registry-api-local-slim
registry-api-local-slim:
	@./dev/registry.sh update api --include-slim --skip-push

.PHONY: registry-create
registry-create:
	@./dev/registry.sh create

.PHONY: registry-clean
registry-clean:
	@./dev/registry.sh clean

.PHONY: manager-local
manager-local:
	@./dev/registry.sh update-manager-local

# Misc

.PHONY: aws-clear-bucket
aws-clear-bucket:
	@./dev/aws.sh clear-bucket

.PHONY: tools
tools:
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@python3 -m pip install black 'pydoc-markdown>=3.0.0,<4.0.0'
	@if [[ "$$OSTYPE" == "darwin"* ]]; then brew install parallel; elif [[ "$$OSTYPE" == "linux"* ]]; then sudo apt-get install -y parallel; else echo "your operating system is not supported"; fi

.PHONY: format
format:
	@./dev/format.sh

#########
# Tests #
#########

.PHONY: test
test:
	@./build/test.sh

.PHONY: test-go
test-go:
	@./build/test.sh go

.PHONY: test-python
test-python:
	@./build/test.sh python

.PHONY: lint
lint:
	@./build/lint.sh

.PHONY: test-examples
test-examples:
	@$(MAKE) registry-all
	@./build/test-examples.sh

###############
# CI Commands #
###############

.PHONY: ci-build-images
ci-build-images:
	@./build/build-images.sh

.PHONY: ci-push-images
ci-push-images:
	@./build/push-images.sh

.PHONY: ci-build-cli
ci-build-cli:
	@./build/cli.sh

.PHONY: ci-build-and-upload-cli
ci-build-and-upload-cli:
	@./build/cli.sh upload
