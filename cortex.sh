#!/bin/bash

# TODO get_cli.sh

# CORTEX_VERSION_BRANCH_STABLE in get_cli.sh

case "$OSTYPE" in
  darwin*)  PARSED_OS="darwin" ;;
  linux*)   PARSED_OS="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

function check_dep_curl() {
  if ! command -v curl >/dev/null; then
    echo -e "\nerror: please install \`curl\`"
    exit 1
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

function install_cli() {
  set -e

  echo -e "Installing CLI (/usr/local/bin/cortex) ..."

  check_dep_curl

  CORTEX_SH_TMP_DIR="$HOME/.cortex-sh-tmp"
  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$PARSED_OS/cortex
  chmod +x $CORTEX_SH_TMP_DIR/cortex

  if [ $(id -u) = 0 ]; then
    mv -f $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  else
    ask_sudo
    sudo mv -f $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo -e "\n✓ Installed CLI"

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
    echo -e "\nThe CLI is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/cortex ]]; then
    echo -e "\nThe CLI was not found at /usr/local/bin/cortex, please uninstall it manually"
    return
  fi

  if [ $(id -u) = 0 ]; then
    rm /usr/local/bin/cortex
  else
    ask_sudo
    sudo rm /usr/local/bin/cortex
  fi
  echo -e "\n✓ Uninstalled CLI"

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

##########################
### TOP-LEVEL COMMANDS ###
##########################

function install_eks() {
  echo
  docker run -it --entrypoint /root/install_eks.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER_NAME=$CORTEX_CLUSTER_NAME \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_INSTANCE_TYPE=$CORTEX_INSTANCE_TYPE \
    -e CORTEX_MIN_INSTANCES=$CORTEX_MIN_INSTANCES \
    -e CORTEX_MAX_INSTANCES=$CORTEX_MAX_INSTANCES \
    $CORTEX_IMAGE_MANAGER
}

function uninstall_eks() {
  echo
  docker run -it --entrypoint /root/uninstall_eks.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER_NAME=$CORTEX_CLUSTER_NAME \
    -e CORTEX_REGION=$CORTEX_REGION \
    $CORTEX_IMAGE_MANAGER
}

function install_cortex() {
  echo
  docker run -it -v $HOME/.cortex:/.cortex --entrypoint /root/install_cortex.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_AWS_ACCESS_KEY_ID=$CORTEX_AWS_ACCESS_KEY_ID \
    -e CORTEX_AWS_SECRET_ACCESS_KEY=$CORTEX_AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER_NAME=$CORTEX_CLUSTER_NAME \
    -e CORTEX_REGION=$CORTEX_REGION \
    -e CORTEX_INSTANCE_TYPE=$CORTEX_INSTANCE_TYPE \
    -e CORTEX_LOG_GROUP=$CORTEX_LOG_GROUP \
    -e CORTEX_BUCKET=$CORTEX_BUCKET \
    -e CORTEX_IMAGE_FLUENTD=$CORTEX_IMAGE_FLUENTD \
    -e CORTEX_IMAGE_STATSD=$CORTEX_IMAGE_STATSD \
    -e CORTEX_IMAGE_OPERATOR=$CORTEX_IMAGE_OPERATOR \
    -e CORTEX_IMAGE_TF_SERVE=$CORTEX_IMAGE_TF_SERVE \
    -e CORTEX_IMAGE_TF_API=$CORTEX_IMAGE_TF_API \
    -e CORTEX_IMAGE_TF_SERVE_GPU=$CORTEX_IMAGE_TF_SERVE_GPU \
    -e CORTEX_IMAGE_ONNX_SERVE=$CORTEX_IMAGE_ONNX_SERVE \
    -e CORTEX_IMAGE_ONNX_SERVE_GPU=$CORTEX_IMAGE_ONNX_SERVE_GPU \
    -e CORTEX_IMAGE_CLUSTER_AUTOSCALER=$CORTEX_IMAGE_CLUSTER_AUTOSCALER \
    -e CORTEX_IMAGE_NVIDIA=$CORTEX_IMAGE_NVIDIA \
    -e CORTEX_IMAGE_METRICS_SERVER=$CORTEX_IMAGE_METRICS_SERVER \
    -e CORTEX_IMAGE_ISTIO_CITADEL=$CORTEX_IMAGE_ISTIO_CITADEL \
    -e CORTEX_IMAGE_ISTIO_GALLEY=$CORTEX_IMAGE_ISTIO_GALLEY \
    -e CORTEX_IMAGE_ISTIO_PILOT=$CORTEX_IMAGE_ISTIO_PILOT \
    -e CORTEX_IMAGE_ISTIO_PROXY=$CORTEX_IMAGE_ISTIO_PROXY \
    -e CORTEX_IMAGE_DOWNLOADER=$CORTEX_IMAGE_DOWNLOADER \
    -e CORTEX_TELEMETRY=$CORTEX_TELEMETRY \
    $CORTEX_IMAGE_MANAGER
}

function info() {
  echo
  docker run -it --entrypoint /root/info.sh \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    -e CORTEX_CLUSTER_NAME=$CORTEX_CLUSTER_NAME \
    -e CORTEX_REGION=$CORTEX_REGION \
    $CORTEX_IMAGE_MANAGER
}

#############################
### DEPENDENCY MANAGEMENT ###
#############################


function prompt_for_email() {
  if [ "$CORTEX_TELEMETRY" != "false" ]; then
    echo
    read -p "Email address [press enter to skip]: "
    echo

    if [[ ! -z "$REPLY" ]]; then
      curl -k -X POST -H "Content-Type: application/json" $CORTEX_TELEMETRY_URL/support -d '{"email_address": "'$REPLY'", "source": "cortex.sh"}' >/dev/null 2>&1 || true
    fi
  fi
}

function confirm_for_uninstall() {
  while true; do
    echo
    read -p "Are you sure you want to uninstall Cortex? (Your cluster will be spun down and all APIs will be deleted) [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      break
    elif [[ $REPLY =~ ^[Nn]$ ]]; then
      exit 0
    fi
    echo "Unexpected value: $REPLY. Please enter \"Y\" or \"n\""
  done
}
