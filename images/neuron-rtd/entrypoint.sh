#!/usr/bin/env bash

start_cmd="neuron-rtd -g unix:/sock/neuron.sock --log-console"
if [ -f "/mnt/kubexit" ]; then
  start_cmd="/mnt/kubexit ${start_cmd}"
fi

eval "${start_cmd}"
