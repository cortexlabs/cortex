#!/bin/bash

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

set -e

CORTEX_VERSION_BRANCH_STABLE=0.31.1
CORTEX_INSTALL_PATH="${CORTEX_INSTALL_PATH:-/usr/local/bin/cortex}"

# replace ~ with the home directory path
CORTEX_INSTALL_PATH="${CORTEX_INSTALL_PATH/#\~/$HOME}"

case "$OSTYPE" in
  darwin*)  parsed_os="darwin" ;;
  linux*)   parsed_os="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

function main() {
  echo -e "\ndownloading cli (${CORTEX_INSTALL_PATH}) ...\n"

  cortex_sh_tmp_dir="$HOME/.cortex-sh-tmp"
  rm -rf $cortex_sh_tmp_dir && mkdir -p $cortex_sh_tmp_dir

  if command -v curl >/dev/null; then
    curl -s -w "" -o $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  elif command -v wget >/dev/null; then
    wget -q -O $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  else
    echo "error: please install \`curl\` or \`wget\`"
    exit 1
  fi

  chmod +x $cortex_sh_tmp_dir/cortex

  if [ $(id -u) = 0 ]; then
    mv -f $cortex_sh_tmp_dir/cortex $CORTEX_INSTALL_PATH
  else
    ask_sudo
    sudo mv -f $cortex_sh_tmp_dir/cortex $CORTEX_INSTALL_PATH
  fi

  rm -rf $cortex_sh_tmp_dir
  echo "✓ installed cli"

  # prompt to update bash profile if running interactively
  if [ -t 1 ]; then
    update_bash_profile
  fi
}

function ask_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo -e "please enter your sudo password\n"
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

function get_zsh_profile_path() {
  if [ -f $HOME/.zshrc ]; then
    echo $HOME/.zshrc
    return
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
  zsh_profile_path=$(get_zsh_profile_path)
  maybe_bash_completion_installed=$(guess_if_bash_completion_installed)
  did_locate_shell_profile="false"

  if [ "$bash_profile_path" != "" ]; then
    did_locate_shell_profile="true"
    if ! grep -Fxq "source <(cortex completion bash)" "$bash_profile_path"; then
      echo
      read -p "Would you like to modify your bash profile ($bash_profile_path) to enable cortex command completion and the cx alias in bash? [y/n] " -r
      echo
      if [ "$REPLY" = "y" ] || [ "$REPLY" = "Y" ] || [ "$REPLY" = "yes" ] || [ "$REPLY" = "Yes" ] || [ "$REPLY" = "YES" ]; then
        echo -e "\nsource <(cortex completion bash)" >> $bash_profile_path
        echo -e "✓ Your bash profile has been updated"
        echo -e "\nCommand to update your current terminal session:"
        echo "  source $bash_profile_path"
        if [ ! "$maybe_bash_completion_installed" = "true" ]; then
          echo -e "Note: \`bash-completion\` must be installed on your system for cortex command completion to function properly"
        fi
      else
        echo -e "Your bash profile has not been modified (run \`cortex completion --help\` to show how to enable bash completion manually)"
      fi
    fi
  fi

  if [ "$zsh_profile_path" != "" ]; then
    did_locate_shell_profile="true"
    if ! grep -Fxq "source <(cortex completion zsh)" "$zsh_profile_path"; then
      echo
      read -p "Would you like to modify your zsh profile ($zsh_profile_path) to enable cortex command completion and the cx alias in zsh? [y/n] " -r
      echo
      if [ "$REPLY" = "y" ] || [ "$REPLY" = "Y" ] || [ "$REPLY" = "yes" ] || [ "$REPLY" = "Yes" ] || [ "$REPLY" = "YES" ]; then
        echo -e "\nsource <(cortex completion zsh)" >> $zsh_profile_path
        echo -e "✓ Your zsh profile has been updated"
        echo -e "\nStart a new zsh shell for completion to take effect. If completion still doesn't work, you can try adding this line to the top of $zsh_profile_path: autoload -Uz compinit && compinit"
      else
        echo -e "Your zsh profile has not been modified (run \`cortex completion --help\` to show how to enable zsh completion manually)"
      fi
    fi
  fi

  if [ "$did_locate_shell_profile" = "false" ]; then
    echo -e "\nIf you would like to enable cortex bash completion and the cx alias, run \`cortex completion --help\` for instructions"
  fi
}

main
