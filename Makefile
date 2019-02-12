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
	@grep -RL \
	--exclude-dir=docs \
	--exclude-dir=examples \
	--exclude-dir=dev/config \
	--exclude-dir=vendor \
	--exclude-dir=bin \
	--exclude-dir=.git \
	--exclude=LICENSE \
	--exclude=requirements.txt \
	--exclude='go.*' \
	--exclude='*.md' \
	--exclude='.git*' \
	--exclude='.travis.yml' \
	--exclude='.dockerignore' \
	"Copyright 2019 Cortex Labs, Inc" 

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
