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

CORTEX_VERSION_BRANCH_STABLE=0.10.0

case "$OSTYPE" in
  darwin*)  parsed_os="darwin" ;;
  linux*)   parsed_os="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

function main() {
  echo -e "\nInstalling CLI (/usr/local/bin/cortex) ...\n"

  cortex_sh_tmp_dir="$HOME/.cortex-sh-tmp"
  rm -rf $cortex_sh_tmp_dir && mkdir -p $cortex_sh_tmp_dir

  if command -v curl >/dev/null; then
    curl -s -o $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  elif command -v wget >/dev/null; then
    wget -q -O $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  else
    echo "error: please install \`curl\` or \`wget\`"
    exit 1
  fi

  chmod +x $cortex_sh_tmp_dir/cortex

  if [ $(id -u) = 0 ]; then
    mv -f $cortex_sh_tmp_dir/cortex /usr/local/bin/cortex
  else
    ask_sudo
    sudo mv -f $cortex_sh_tmp_dir/cortex /usr/local/bin/cortex
  fi

  rm -rf $cortex_sh_tmp_dir
  echo "✓ Installed CLI"

  update_bash_profile
}

function ask_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo -e "Please enter your sudo password\n"
  fi
}

function get_bash_profile_path() {
  if [ "$parsed_os" = "darwin" ]; then
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

function guess_if_bash_completion_installed() {
  if [ -f $HOME/.bash_profile ]; then
    if grep -q -e "^[^#]*bash-completion" -e "^[^#]*bash_completion" "$HOME/.bash_profile"; then
      echo "true"
      return
    fi
  fi

  if [ -f $HOME/.bashrc ]; then
    if grep -q -e "^[^#]*bash-completion" -e "^[^#]*bash_completion" "$HOME/.bashrc"; then
      echo "true"
      return
    fi
  fi

  echo "false"
}

function update_bash_profile() {
  bash_profile_path=$(get_bash_profile_path)
  maybe_bash_completion_installed=$(guess_if_bash_completion_installed)

  if [ "$bash_profile_path" != "" ]; then
    if ! grep -Fxq "source <(cortex completion)" "$bash_profile_path"; then
      echo
      read -p "Would you like to modify your bash profile ($bash_profile_path) to enable cortex command completion and the cx alias? [y/n] " -n 1 -r
      echo
      if [ "$REPLY" != "" ]; then
        echo
      fi
      if [[ "$REPLY" =~ ^[Yy]$ ]] || [ "$REPLY" = "" ]; then
        echo -e "\nsource <(cortex completion)" >> $bash_profile_path
        echo -e "✓ Your bash profile has been updated"
        echo -e "\nCommand to update your current terminal session:"
        echo "  source $bash_profile_path"
        if [ ! "$maybe_bash_completion_installed" = "true" ]; then
          echo -e "Note: \`bash-completion\` must be installed on your system for cortex command completion to function properly"
        fi
      else
        echo -e "Your bash profile has not been modified. If you would like to modify it manually, add this line to your bash profile:"
        echo "  source <(cortex completion)"
        if [ ! "$maybe_bash_completion_installed" = "true" ]; then
          echo "Note: \`bash-completion\` must be installed on your system for cortex command completion to function properly"
        fi
      fi
    fi
  else
    echo -e "\nIf your would like to enable cortex command completion and the cx alias, add this line to your bash profile:"
    echo "  source <(cortex completion)"
    if [ ! "$maybe_bash_completion_installed" = "true" ]; then
      echo "Note: \`bash-completion\` must be installed on your system for cortex command completion to function properly"
    fi
  fi
}

main
