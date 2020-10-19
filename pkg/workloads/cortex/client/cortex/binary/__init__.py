# Copyright 2020 Cortex Labs, Inc.
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
from os import path
import base64
import sys
import subprocess
from typing import List, Optional, Tuple, Callable
from io import BytesIO
from cortex.exceptions import CortexBinaryException

MIXED_CORTEX_MARKER = "~~cortex~~"


def run():
    try:
        process = subprocess.run([get_cli_path()] + sys.argv[1:], cwd=os.getcwd())
    except KeyboardInterrupt:
        sys.exit(130)  # Ctrl + C
    sys.exit(process.returncode)


def run_cli(
    args: List[str],
    cwd: Optional[str] = None,
    hide_output: bool = False,
    mixed: bool = False,
) -> str:
    output = ""
    result = ""
    result_found = False

    env = os.environ.copy()
    env["CORTEX_CLI_INVOKER"] = "python"
    process = subprocess.Popen(
        [get_cli_path()] + args,
        cwd=cwd,
        stderr=subprocess.PIPE,
        stdout=subprocess.PIPE,
        encoding="utf8",
        env=env,
    )

    for c in iter(lambda: process.stdout.read(1), ""):
        output += c
        if mixed:
            if output[-2:] == "\n~":
                result_found = True
                result = "~"
                output = output[:-1]
            if result_found:
                if c == "\n":
                    result_found = False
                    result = result[len(MIXED_CORTEX_MARKER) : -len(MIXED_CORTEX_MARKER)]
                    result = base64.b64decode(result).decode("utf8")
                else:
                    result += c

        if not hide_output:
            if (not mixed) or (mixed and not result_found):
                sys.stdout.write(c)

    process.wait()

    if process.returncode == 0:
        if mixed:
            return result
        return output

    raise CortexBinaryException(process.stderr.read())


def get_cli_path() -> str:
    if os.environ.get("CORTEX_CLI_PATH") is not None:
        cli_path = os.environ["CORTEX_CLI_PATH"]
        if not os.path.exists(cli_path):
            raise Exception(
                f"unable to find cortex binary at {cli_path} as specified in `CORTEX_CLI_PATH` environment variable"
            )
        return cli_path

    cli_path = os.path.join(os.path.abspath(os.path.dirname(__file__)), "cli")
    if not os.path.exists(cli_path):
        raise Exception(
            f"unable to find cortex binary at {cli_path}, please reinstall the cortex client using `pip uninstall cortex` and then `pip install cortex`"
        )
    return cli_path
