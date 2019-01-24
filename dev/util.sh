#!/bin/bash

function blue_echo() {
  echo -e "\033[1;34m$1\033[0m"
}

function green_echo() {
  echo -e "\033[1;32m$1\033[0m"
}

function error_echo() {
  echo -e "\033[1;31mERROR: \033[0m$1"
}
