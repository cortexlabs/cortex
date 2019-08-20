#!/bin/bash

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

set -e

####################
### FLAG PARSING ###
####################

flag_help=false
positional_args=()

while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -c|--config)
    export CORTEX_CONFIG="$2"
    shift
    shift
    ;;
    -h|--help)
    flag_help="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done

set -- "${positional_args[@]}"
positional_args=()
for i in "$@"; do
  case $i in
    -c=*|--config=*)
    export CORTEX_CONFIG="${i#*=}"
    shift
    ;;
    -h=*|--help=*)
    flag_help="true"
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done

set -- "${positional_args[@]}"
if [ "$flag_help" == "true" ]; then
  show_help
  exit 0
fi

for arg in "$@"; do
  if [[ "$arg" == -* ]]; then
    echo "unknown flag: $arg"
    show_help
    exit 1
  fi
done

#####################
### CONFIGURATION ###
#####################

if [ "$CORTEX_CONFIG" != "" ]; then
  if [ ! -f "$CORTEX_CONFIG" ]; then
    echo "Cortex config file does not exist: $CORTEX_CONFIG"
    exit 1
  fi
  source $CORTEX_CONFIG
fi

set -u

export CORTEX_VERSION_STABLE=0.7.3

# Defaults
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-""}"

if [ "$AWS_ACCESS_KEY_ID" = "" ]; then
  echo -e "\nPlease set AWS_ACCESS_KEY_ID"
  exit 1
fi

export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"

if [ "$AWS_SECRET_ACCESS_KEY" = "" ]; then
  echo -e "\nPlease set AWS_SECRET_ACCESS_KEY"
  exit 1
fi

export CORTEX_LOG_GROUP="${CORTEX_LOG_GROUP:-cortex}"
export CORTEX_BUCKET="${CORTEX_BUCKET:-""}"
export CORTEX_REGION="${CORTEX_REGION:-us-west-2}"
export CORTEX_ZONES="${CORTEX_ZONES:-""}"

export CORTEX_CLUSTER="${CORTEX_CLUSTER:-cortex}"
export CORTEX_NODE_TYPE="${CORTEX_NODE_TYPE:-t3.large}"
export CORTEX_NODES_MIN="${CORTEX_NODES_MIN:-2}"
export CORTEX_NODES_MAX="${CORTEX_NODES_MAX:-5}"
export CORTEX_NAMESPACE="${CORTEX_NAMESPACE:-cortex}"

export CORTEX_IMAGE_MANAGER="${CORTEX_IMAGE_MANAGER:-cortexlabs/manager:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_FLUENTD="${CORTEX_IMAGE_FLUENTD:-cortexlabs/fluentd:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_NGINX_BACKEND="${CORTEX_IMAGE_NGINX_BACKEND:-cortexlabs/nginx-backend:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_NGINX_CONTROLLER="${CORTEX_IMAGE_NGINX_CONTROLLER:-cortexlabs/nginx-controller:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_OPERATOR="${CORTEX_IMAGE_OPERATOR:-cortexlabs/operator:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_SPARK="${CORTEX_IMAGE_SPARK:-cortexlabs/spark:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_SPARK_OPERATOR="${CORTEX_IMAGE_SPARK_OPERATOR:-cortexlabs/spark-operator:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_SERVE="${CORTEX_IMAGE_TF_SERVE:-cortexlabs/tf-serve:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_TRAIN="${CORTEX_IMAGE_TF_TRAIN:-cortexlabs/tf-train:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_API="${CORTEX_IMAGE_TF_API:-cortexlabs/tf-api:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_PYTHON_PACKAGER="${CORTEX_IMAGE_PYTHON_PACKAGER:-cortexlabs/python-packager:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_SERVE_GPU="${CORTEX_IMAGE_TF_SERVE_GPU:-cortexlabs/tf-serve-gpu:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_TRAIN_GPU="${CORTEX_IMAGE_TF_TRAIN_GPU:-cortexlabs/tf-train-gpu:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_ONNX_SERVE="${CORTEX_IMAGE_ONNX_SERVE:-cortexlabs/onnx-serve:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_ONNX_SERVE_GPU="${CORTEX_IMAGE_ONNX_SERVE_GPU:-cortexlabs/onnx-serve-gpu:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_CLUSTER_AUTOSCALER="${CORTEX_IMAGE_CLUSTER_AUTOSCALER:-cortexlabs/cluster-autoscaler:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_NVIDIA="${CORTEX_IMAGE_NVIDIA:-cortexlabs/nvidia:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_METRICS_SERVER="${CORTEX_IMAGE_METRICS_SERVER:-cortexlabs/metrics-server:$CORTEX_VERSION_STABLE}"

export CORTEX_ENABLE_TELEMETRY="${CORTEX_ENABLE_TELEMETRY:-""}"

##########################
### TOP-LEVEL COMMANDS ###
##########################

function install_eks() {
  echo
  docker run -it --entrypoint /root/install_eks.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NODE_TYPE=$CORTEX_NODE_TYPE \
    -e CORTEX_NODES_MIN=$CORTEX_NODES_MIN \
    -e CORTEX_NODES_MAX=$CORTEX_NODES_MAX \
    $CORTEX_IMAGE_MANAGER
}

function uninstall_eks() {
  echo
  docker run -it --entrypoint /root/uninstall_eks.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    $CORTEX_IMAGE_MANAGER
}

function install_cortex() {
  echo
  docker run -it --entrypoint /root/install_cortex.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    -e CORTEX_NODE_TYPE=$CORTEX_NODE_TYPE \
    -e CORTEX_LOG_GROUP=$CORTEX_LOG_GROUP \
    -e CORTEX_BUCKET=$CORTEX_BUCKET \
    -e CORTEX_IMAGE_FLUENTD=$CORTEX_IMAGE_FLUENTD \
    -e CORTEX_IMAGE_NGINX_BACKEND=$CORTEX_IMAGE_NGINX_BACKEND \
    -e CORTEX_IMAGE_NGINX_CONTROLLER=$CORTEX_IMAGE_NGINX_CONTROLLER \
    -e CORTEX_IMAGE_OPERATOR=$CORTEX_IMAGE_OPERATOR \
    -e CORTEX_IMAGE_SPARK=$CORTEX_IMAGE_SPARK \
    -e CORTEX_IMAGE_SPARK_OPERATOR=$CORTEX_IMAGE_SPARK_OPERATOR \
    -e CORTEX_IMAGE_TF_SERVE=$CORTEX_IMAGE_TF_SERVE \
    -e CORTEX_IMAGE_TF_TRAIN=$CORTEX_IMAGE_TF_TRAIN \
    -e CORTEX_IMAGE_TF_API=$CORTEX_IMAGE_TF_API \
    -e CORTEX_IMAGE_PYTHON_PACKAGER=$CORTEX_IMAGE_PYTHON_PACKAGER \
    -e CORTEX_IMAGE_TF_SERVE_GPU=$CORTEX_IMAGE_TF_SERVE_GPU \
    -e CORTEX_IMAGE_TF_TRAIN_GPU=$CORTEX_IMAGE_TF_TRAIN_GPU \
    -e CORTEX_IMAGE_ONNX_SERVE=$CORTEX_IMAGE_ONNX_SERVE \
    -e CORTEX_IMAGE_ONNX_SERVE_GPU=$CORTEX_IMAGE_ONNX_SERVE_GPU \
    -e CORTEX_IMAGE_CLUSTER_AUTOSCALER=$CORTEX_IMAGE_CLUSTER_AUTOSCALER \
    -e CORTEX_IMAGE_NVIDIA=$CORTEX_IMAGE_NVIDIA \
    -e CORTEX_IMAGE_METRICS_SERVER=$CORTEX_IMAGE_METRICS_SERVER \
    -e CORTEX_ENABLE_TELEMETRY=$CORTEX_ENABLE_TELEMETRY \
    $CORTEX_IMAGE_MANAGER
}

function uninstall_operator() {
  echo
  docker run -it --entrypoint /root/uninstall_operator.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    $CORTEX_IMAGE_MANAGER
}

function info() {
  echo
  docker run -it --entrypoint /root/info.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER=$CORTEX_CLUSTER \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_NAMESPACE=$CORTEX_NAMESPACE \
    $CORTEX_IMAGE_MANAGER
}

################
### CHECK OS ###
################

case "$OSTYPE" in
  darwin*)  PARSED_OS="darwin" ;;
  linux*)   PARSED_OS="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

#############################
### DEPENDENCY MANAGEMENT ###
#############################

function check_dep_curl() {
  if ! command -v curl >/dev/null; then
    echo -e "\nerror: please install \`curl\`"
    exit 1
  fi
}

function install_cli() {
  set -e

  check_dep_curl

  echo -e "\nInstalling the Cortex CLI (/usr/local/bin/cortex) ..."

  CORTEX_SH_TMP_DIR="$HOME/.cortex-sh-tmp"
  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_STABLE/cli/$PARSED_OS/cortex
  chmod +x $CORTEX_SH_TMP_DIR/cortex

  if [ $(id -u) = 0 ]; then
    mv -f $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  else
    ask_sudo
    sudo mv -f $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed the Cortex CLI"

  bash_profile_path=$(get_bash_profile)
  if [ ! "$bash_profile_path" = "" ]; then
    if ! grep -Fxq "source <(cortex completion)" "$bash_profile_path"; then
      echo
      read -p "Would you like to modify your bash profile ($bash_profile_path) to enable cortex command completion and the cx alias? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "\nsource <(cortex completion)" >> $bash_profile_path
        echo "✓ Your bash profile ($bash_profile_path) has been updated"
        echo
        echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
        echo
        echo "Command to update your current terminal session:"
        echo "  source $bash_profile_path"
      else
        echo "Your bash profile has not been modified. If you would like to modify it manually, add this line to your bash profile:"
        echo "  source <(cortex completion)"
        echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
      fi
    fi
  else
    echo -e "\nIf your would like to enable cortex command completion and the cx alias, add this line to your bash profile:"
    echo "  source <(cortex completion)"
    echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
  fi
}

function uninstall_cli() {
  set -e

  rm -rf $HOME/.cortex

  if ! command -v cortex >/dev/null; then
    echo -e "\nThe Cortex CLI is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/cortex ]]; then
    echo -e "\nThe Cortex CLI was not found at /usr/local/bin/cortex, please uninstall it manually"
    return
  fi

  if [ $(id -u) = 0 ]; then
    rm /usr/local/bin/cortex
  else
    ask_sudo
    sudo rm /usr/local/bin/cortex
  fi
  echo -e "\n✓ Uninstalled the Cortex CLI"

  bash_profile_path=$(get_bash_profile)
  if [ ! "$bash_profile_path" = "" ]; then
    if grep -Fxq "source <(cortex completion)" "$bash_profile_path"; then
      echo
      read -p "Would you like to remove \"source <(cortex completion)\" from your bash profile ($bash_profile_path)? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        sed '/^source <(cortex completion)$/d' "$bash_profile_path" > "${bash_profile_path}_cortex_modified" && mv -f "${bash_profile_path}_cortex_modified" "$bash_profile_path"
        echo "✓ Your bash profile ($bash_profile_path) has been updated"
      fi
    fi
  fi
}

function get_bash_profile() {
  if [ "$PARSED_OS" = "darwin" ]; then
    if [ -f $HOME/.bash_profile ]; then
      echo $HOME/.bash_profile
      return
    elif [ -f $HOME/.bashrc ]; then
      echo $HOME/.bashrc
      return
    fi
  else
    if [ -f $HOME/.bashrc ]; then
      echo $HOME/.bashrc
      return
    elif [ -f $HOME/.bash_profile ]; then
      echo $HOME/.bash_profile
      return
    fi
  fi

  echo ""
}

function ask_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo -e "\nPlease enter your sudo password"
  fi
}

function prompt_for_telemetry() {
  if [ "$CORTEX_ENABLE_TELEMETRY" != "true" ] && [ "$CORTEX_ENABLE_TELEMETRY" != "false" ]; then
    while true
    do
      echo
      read -p "Would you like to help improve Cortex by anonymously sending error reports and usage stats to the dev team? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        export CORTEX_ENABLE_TELEMETRY=true
        break
      elif [[ $REPLY =~ ^[Nn]$ ]]; then
        export CORTEX_ENABLE_TELEMETRY=false
        break
      fi
      echo "Unexpected value, please enter \"Y\" or \"n\""
    done
  fi
}

############
### HELP ###
############

function show_help() {
  echo "
Usage:
  ./cortex.sh command [sub-command] [flags]

Available Commands:
  install             install Cortex
  uninstall           uninstall Cortex
  update              update Cortex
  info                information about Cortex

  install cli         install the Cortex CLI
  uninstall cli       uninstall the Cortex CLI

Flags:
  -c, --config  path to a Cortex config file
  -h, --help
"
}

######################
### ARG PROCESSING ###
######################

arg1=${1:-""}
arg2=${2:-""}
arg3=${3:-""}

if [ -z "$arg1" ]; then
  show_help
  exit 0
fi

if [ "$arg1" = "install" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for install command"
    show_help
    exit 1
  elif [ "$arg2" = "" ]; then
    prompt_for_telemetry && install_eks && install_cortex && info
  elif [ "$arg2" = "cli" ]; then
    install_cli
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for install"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for install: $arg2"
    show_help
    exit 1
  fi
elif [ "$arg1" = "uninstall" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for uninstall command"
    show_help
    exit 1
  elif [ "$arg2" = "" ]; then
    uninstall_eks
  elif [ "$arg2" = "cli" ]; then
    uninstall_cli
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for uninstall"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for uninstall: $arg2"
    show_help
    exit 1
  fi
elif [ "$arg1" = "update" ]; then
  if [ ! "$arg2" = "" ]; then
    echo -e "\nerror: too many arguments for get command"
    show_help
    exit 1
  else
    uninstall_operator && install_cortex
  fi
elif [ "$arg1" = "info" ]; then
  if [ ! "$arg2" = "" ]; then
    echo -e "\nerror: too many arguments for get command"
    show_help
    exit 1
  else
    info
  fi
else
  echo -e "\nerror: unknown command: $arg1"
  show_help
  exit 1
fi
