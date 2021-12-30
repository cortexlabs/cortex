#!/usr/bin/env bash

nucleus generate model-server-config.yaml
docker build -f nucleus.Dockerfile -t 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/realtime-sleep-cpu:latest .
docker push 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/realtime-sleep-cpu:latest
