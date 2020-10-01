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

# build cli, start local operator, and watch for changes
devstart:
	@$(MAKE) operator-stop || true
	@./dev/operator_local.sh || true

.PHONY: cli
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

# configure kubectl to point to the cluster specified in dev/config/cluster.yaml
kubectl:
	@eval $$(python3 ./manager/cluster_config_env.py ./dev/config/cluster.yaml) && eksctl utils write-kubeconfig --cluster="$$CORTEX_CLUSTER_NAME" --region="$$CORTEX_REGION" | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

cluster-up:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster up --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY
	@$(MAKE) kubectl

cluster-up-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster up --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes
	@$(MAKE) kubectl

cluster-down:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster down --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY

cluster-down-y:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster down --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --yes

cluster-info:
	@$(MAKE) manager-local
	@$(MAKE) cli
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster info --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY

cluster-configure:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster configure --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY

cluster-configure-y:
	@$(MAKE) registry-all
	@$(MAKE) cli
	@kill $(shell pgrep -f rerun) >/dev/null 2>&1 || true
	@source ./dev/config/env.sh 2>/dev/null; ./bin/cortex cluster configure --config=./dev/config/cluster.yaml --aws-key=$$AWS_ACCESS_KEY_ID --aws-secret=$$AWS_SECRET_ACCESS_KEY --cluster-aws-key=$$CLUSTER_AWS_ACCESS_KEY_ID --cluster-aws-secret=$$CLUSTER_AWS_SECRET_ACCESS_KEY --yes

# stop the in-cluster operator
operator-stop:
	@$(MAKE) kubectl
	@kubectl delete --namespace=default --ignore-not-found=true deployment operator

# Docker images

registry-all:
	@./dev/registry.sh update all
registry-all-local:
	@./dev/registry.sh update all --skip-push
registry-all-slim:
	@./dev/registry.sh update all --include-slim
registry-all-slim-local:
	@./dev/registry.sh update all --include-slim --skip-push
registry-all-local-slim:
	@./dev/registry.sh update all --include-slim --skip-push

registry-dev:
	@./dev/registry.sh update dev
registry-dev-local:
	@./dev/registry.sh update dev --skip-push
registry-dev-slim:
	@./dev/registry.sh update dev --include-slim
registry-dev-slim-local:
	@./dev/registry.sh update dev --include-slim --skip-push
registry-dev-local-slim:
	@./dev/registry.sh update dev --include-slim --skip-push

registry-api:
	@./dev/registry.sh update api
registry-api-local:
	@./dev/registry.sh update api --skip-push
registry-api-slim:
	@./dev/registry.sh update api --include-slim
registry-api-slim-local:
	@./dev/registry.sh update api --include-slim --skip-push
registry-api-local-slim:
	@./dev/registry.sh update api --include-slim --skip-push

registry-create:
	@./dev/registry.sh create

registry-clean:
	@./dev/registry.sh clean

manager-local:
	@./dev/registry.sh update-manager-local

# Misc

aws-clear-bucket:
	@./dev/aws.sh clear-bucket

tools:
	@go get -u -v golang.org/x/lint/golint
	@go get -u -v github.com/VojtechVitek/rerun/cmd/rerun
	@python3 -m pip install black

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
	@./build/build-images.sh

ci-push-images:
	@./build/push-images.sh

ci-build-cli:
	@./build/cli.sh

ci-build-and-upload-cli:
	@./build/cli.sh upload
