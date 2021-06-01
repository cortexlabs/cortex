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

import os
import subprocess
import sys
from typing import List

from cortex.exceptions import CortexBinaryException


def run():
    """
    Runs the CLI from terminal.
    """
    try:
        process = subprocess.run([get_cli_path()] + sys.argv[1:], cwd=os.getcwd())
    except KeyboardInterrupt:
        sys.exit(130)  # Ctrl + C
    sys.exit(process.returncode)


def run_cli(
    args: List[str],
    hide_output: bool = False,
) -> str:
    """
    Runs the Cortex binary with the specified arguments.

    Args:
        args: Arguments to use when invoking the Cortex CLI.
        hide_output: Flag to prevent streaming CLI output to stdout.

    Raises:
        CortexBinaryException: Cortex CLI command returned an error.

    Returns:
        The stdout from the Cortex CLI command.
    """

    env = os.environ.copy()
    env["CORTEX_CLI_INVOKER"] = "python"
    process = subprocess.Popen(
        [get_cli_path()] + args,
        stderr=subprocess.PIPE,
        stdout=subprocess.PIPE,
        encoding="utf8",
        env=env,
    )

    output = ""
    result = ""

    for c in iter(lambda: process.stdout.read(1), ""):
        output += c

        if not hide_output:
            sys.stdout.write(c)
            sys.stdout.flush()

    process.wait()

    if process.returncode == 0:
        return output

    if result != "":
        raise CortexBinaryException(result + "\n" + process.stderr.read())

    raise CortexBinaryException(process.stderr.read())


def get_cli_path() -> str:
    """
    Get the location of the CLI.

    Default location is the directory containing the `cortex.binary` package.
    The location can be overridden by setting the `CORTEX_CLI_PATH` environment variable.

    Raises:
        Exception: Unable to find the CLI.

    Returns:
        str: The location of the CLI in the local filesystem.
    """
    if os.environ.get("CORTEX_CLI_PATH") is not None:
        cli_path = os.environ["CORTEX_CLI_PATH"]
        if not os.path.exists(cli_path):
            raise Exception(
                f"unable to find cortex binary at {cli_path} as specified in `CORTEX_CLI_PATH` environment variable"
            )
        return cli_path

    try:
        import importlib.resources as pkg_resources
    except ImportError:
        # Try backported to PY<37 `importlib_resources`.
        import importlib_resources as pkg_resources

    from cortex import binary

    try:
        with pkg_resources.path(binary, "cli") as p:
            cli_path = p
    except FileNotFoundError as e:
        raise Exception(
            "unable to find cortex binary, please reinstall the cortex client by running `pip uninstall cortex` and then `pip install cortex`"
        ) from e

    return str(cli_path)
