# Copyright 2022 Cortex Labs, Inc.
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

import importlib
import pathlib
from typing import Any, Callable, List
import sys
import inspect

from e2e.exceptions import GeneratorValidationException


def load_generator(sample_generator: pathlib.Path) -> Callable[[], List[int]]:
    api_dir = str(sample_generator.parent)

    sys.path.append(api_dir)
    sample_generator_module = importlib.import_module(str(pathlib.Path(sample_generator).stem))
    sys.path.pop()

    validate_module(sample_generator_module)

    return sample_generator_module.generate_sample


def validate_module(sample_generator_module: Any):
    if not hasattr(sample_generator_module, "generate_sample"):
        raise GeneratorValidationException(
            "sample generator module doesn't have a function called 'generate_sample'"
        )

    if not inspect.isfunction(getattr(sample_generator_module, "generate_sample")):
        raise GeneratorValidationException("'generate_sample' is not a function")

    if inspect.getfullargspec(getattr(sample_generator_module, "generate_sample")).args != []:
        raise GeneratorValidationException(
            "'generate_sample' function must not have any parameters"
        )
