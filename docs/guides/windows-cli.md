# Install CLI on Windows

This guide walks you through installing the Cortex CLI on a Windows 10 machine using the Linux Subsystem.

The requirement is an x64 system with Windows 10 of **Version 1903** or higher, with **Build 18362** or higher.

## Step 1

Install and configure the WSL (Windows Subsystem for Linux) version 2 on your machine following [this installation guide](https://docs.microsoft.com/en-us/windows/wsl/install-win10).

In our example, we assume the installation of the Ubuntu distribution.

## Step 2

Install and configure the Docker Desktop Engine app to use WSL 2 as its backend by following the steps in the [Docker Desktop WSL 2 backend guide](https://docs.docker.com/docker-for-windows/wsl/).

## Step 3

Run Ubuntu in the terminal on your Windows machine and right-click the window's bar and click on *Properties*:

![step-3a](https://user-images.githubusercontent.com/26958764/96926494-493cdf80-14be-11eb-9fac-4c81e1fac55c.png)

In the *Font* category, set the font to one of the following fonts: **SimSun-ExtB** (recommended), **MS Gothic**, or **NSimSun**. Choosing one of these fonts helps render all Unicode characters correctly. Once selected, click *Okay*.

![step-3b](https://user-images.githubusercontent.com/26958764/96926763-adf83a00-14be-11eb-9584-4eff3faf2377.png)

## Step 4

Within the Ubuntu terminal, install the Cortex CLI as you would on a Mac/Linux machine:

### Install the CLI with Python Client

```bash
pip install cortex
```

### Install the CLI without Python Client

```bash
# Replace `INSERT_CORTEX_VERSION` with the complete CLI version (e.g. 0.18.1):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/vINSERT_CORTEX_VERSION/get-cli.sh)"

# For example to download CLI version 0.18.1 (Note the "v"):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.18.1/get-cli.sh)"
```

## Step 5

Start using the Cortex CLI. The CLI commands are documented [here](../miscellaneous/cli.md#command-overview).

![step-5](https://user-images.githubusercontent.com/26958764/96927485-ca48a680-14bf-11eb-909c-f15dbc52af08.png)
