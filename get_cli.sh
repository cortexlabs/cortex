#!/bin/bash

set -e

CORTEX_VERSION_BRANCH_STABLE=master

case "$OSTYPE" in
  darwin*)  parsed_os="darwin" ;;
  linux*)   parsed_os="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

function main() {
  echo -e "Installing CLI (/usr/local/bin/cortex) ..."

  cortex_sh_tmp_dir="$HOME/.cortex-sh-tmp"
  rm -rf $cortex_sh_tmp_dir && mkdir -p $cortex_sh_tmp_dir

  if command -v curl >/dev/null; then
    curl -s -o $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  elif command -v wget >/dev/null; then
    wget -q -O $cortex_sh_tmp_dir/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_BRANCH_STABLE/cli/$parsed_os/cortex
  else
    echo -e "\nerror: please install \`curl\` or \`wget\`"
  fi

  chmod +x $cortex_sh_tmp_dir/cortex

  if [ $(id -u) = 0 ]; then
    mv -f $cortex_sh_tmp_dir/cortex /usr/local/bin/cortex
  else
    ask_sudo
    sudo mv -f $cortex_sh_tmp_dir/cortex /usr/local/bin/cortex
  fi

  rm -rf $cortex_sh_tmp_dir
  echo -e "\n✓ Installed CLI"

  update_bash_profile
}

function ask_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo -e "\nPlease enter your sudo password"
  fi
}

function get_bash_profile() {
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

function update_bash_profile() {
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
