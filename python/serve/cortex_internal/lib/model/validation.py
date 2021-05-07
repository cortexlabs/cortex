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

import collections
import operator
import os
import uuid
from enum import IntEnum
from fnmatch import fnmatchcase
from typing import List, Any, Tuple

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import CortexException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.type import (
    PythonHandlerType,
    TensorFlowHandlerType,
    TensorFlowNeuronHandlerType,
    HandlerType,
)

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class TemplatePlaceholder(collections.namedtuple("TemplatePlaceholder", "placeholder priority")):
    """
    Placeholder type that denotes an operation, a text placeholder, etc.

    Accessible properties: type, priority.
    """

    def __new__(cls, placeholder: str, priority: int):
        return super(cls, TemplatePlaceholder).__new__(cls, "<" + placeholder + ">", priority)

    def __str__(self) -> str:
        return str(self.placeholder)

    def __repr__(self) -> str:
        return str(self.placeholder)

    @property
    def type(self) -> str:
        return str(self.placeholder).strip("<>")


class GenericPlaceholder(
    collections.namedtuple("GenericPlaceholder", "placeholder value priority")
):
    """
    Generic placeholder.

    Can hold any value.
    Can be of one type only: generic.

    Accessible properties: placeholder, value, type, priority.
    """

    def __new__(cls, value: str):
        return super(cls, GenericPlaceholder).__new__(cls, "<generic>", value, 0)

    def __eq__(self, other) -> bool:
        return isinstance(other, GenericPlaceholder)

    def __hash__(self):
        return hash((self.placeholder, self.value))

    def __str__(self) -> str:
        return f"<{self.type}>" + str(self.value) + f"</{self.type}>"

    def __repr__(self) -> str:
        return f"<{self.type}>" + str(self.value) + f"</{self.type}>"

    @property
    def type(self) -> str:
        return str(self.placeholder).strip("<>")


class PlaceholderGroup:
    """
    Order-based addition of placeholder types.
    Can use AnyPlaceholder, GenericPlaceholder, SinglePlaceholder.

    Accessible properties: parts, type, priority.
    """

    def __init__(self, *args, priority=0):
        self.parts = args
        self.priority = priority

    def __getitem__(self, index: int):
        return self.parts[index]

    def __len__(self) -> int:
        return len(self.parts)

    def __str__(self) -> str:
        return "<group>" + str(self.parts) + "</group>"

    def __repr__(self) -> str:
        return "<group>" + str(self.parts) + "</group>"

    @property
    def type(self) -> str:
        return str(self.parts)


class OneOfAllPlaceholder:
    """
    Can be any of the provided alternatives.

    Accessible properties: parts, type, priority, ID.
    """

    def __init__(self, ID: Any = None):
        self._placeholder = TemplatePlaceholder("oneofall", priority=-1)
        if not ID:
            ID = uuid.uuid4().int
        self.ID = ID

    def __str__(self) -> str:
        return str(self._placeholder)

    def __repr__(self) -> str:
        return str(self._placeholder)

    @property
    def type(self) -> str:
        return str(self._placeholder).strip("<>")

    @property
    def priority(self) -> int:
        return self._placeholder.priority


IntegerPlaceholder = TemplatePlaceholder("integer", priority=1)  # the path name must be an integer
SinglePlaceholder = TemplatePlaceholder(
    "single", priority=2
)  # can only have a single occurrence of this, but its name can take any form
AnyPlaceholder = TemplatePlaceholder(
    "any", priority=4
)  # the path can be any file or any directory (with multiple subdirectories)


class ModelVersion(IntEnum):
    NOT_PROVIDED = 1  # for models provided without a specific version
    PROVIDED = 2  # for models provided with version directories (1, 2, 452, etc).


# to be used when handler:models:path, handler:models:paths or handler:models:dir is used
ModelTemplate = {
    PythonHandlerType: {
        OneOfAllPlaceholder(ModelVersion.PROVIDED): {
            IntegerPlaceholder: AnyPlaceholder,
        },
        OneOfAllPlaceholder(ModelVersion.NOT_PROVIDED): {
            AnyPlaceholder: None,
        },
    },
    TensorFlowHandlerType: {
        OneOfAllPlaceholder(ModelVersion.PROVIDED): {
            IntegerPlaceholder: {
                AnyPlaceholder: None,
                GenericPlaceholder("saved_model.pb"): None,
                GenericPlaceholder("variables"): {
                    GenericPlaceholder("variables.index"): None,
                    PlaceholderGroup(
                        GenericPlaceholder("variables.data-00000-of-"), AnyPlaceholder
                    ): None,
                    AnyPlaceholder: None,
                },
            },
        },
        OneOfAllPlaceholder(ModelVersion.NOT_PROVIDED): {
            AnyPlaceholder: None,
            GenericPlaceholder("saved_model.pb"): None,
            GenericPlaceholder("variables"): {
                GenericPlaceholder("variables.index"): None,
                PlaceholderGroup(
                    GenericPlaceholder("variables.data-00000-of-"), AnyPlaceholder
                ): None,
                AnyPlaceholder: None,
            },
        },
    },
    TensorFlowNeuronHandlerType: {
        OneOfAllPlaceholder(ModelVersion.PROVIDED): {
            IntegerPlaceholder: {
                GenericPlaceholder("saved_model.pb"): None,
                AnyPlaceholder: None,
            }
        },
        OneOfAllPlaceholder(ModelVersion.NOT_PROVIDED): {
            GenericPlaceholder("saved_model.pb"): None,
            AnyPlaceholder: None,
        },
    },
}


def json_model_template_representation(model_template) -> dict:
    dct = {}
    if model_template is None:
        return None
    if isinstance(model_template, dict):
        if any(isinstance(x, OneOfAllPlaceholder) for x in model_template):
            oneofall_placeholder_index = 0
        for key in model_template:
            if isinstance(key, OneOfAllPlaceholder):
                dct[
                    str(key) + f"-{oneofall_placeholder_index}"
                ] = json_model_template_representation(model_template[key])
                oneofall_placeholder_index += 1
            else:
                dct[str(key)] = json_model_template_representation(model_template[key])
        return dct
    else:
        return str(model_template)


def _single_model_pattern(handler_type: HandlerType) -> dict:
    """
    To be used when handler:models:path or handler:models:paths in cortex.yaml is used.
    """
    return ModelTemplate[handler_type]


def validate_models_dir_paths(
    paths: List[str], handler_type: HandlerType, common_prefix: str
) -> Tuple[List[str], List[List[int]]]:
    """
    Validates the models paths based on the given handler type.
    To be used when handler:models:dir in cortex.yaml is used.

    Args:
        paths: A list of all paths for a given s3/local prefix. Must be underneath the common prefix.
        handler_type: The handler type.
        common_prefix: The common prefix of the directory which holds all models. AKA handler:models:dir.

    Returns:
        List with the prefix of each model that's valid.
        List with the OneOfAllPlaceholder IDs validated for each valid model.
    """
    if len(paths) == 0:
        raise CortexException(
            f"{handler_type} handler at '{common_prefix}'", "model top path can't be empty"
        )

    rel_paths = [os.path.relpath(top_path, common_prefix) for top_path in paths]
    rel_paths = [path for path in rel_paths if not path.startswith("../")]

    model_names = [util.get_leftmost_part_of_path(path) for path in rel_paths]
    model_names = list(set(model_names))

    valid_model_prefixes = []
    ooa_valid_key_ids = []
    for model_name in model_names:
        try:
            ooa_valid_key_ids.append(validate_model_paths(rel_paths, handler_type, model_name))
            valid_model_prefixes.append(os.path.join(common_prefix, model_name))
        except CortexException as e:
            logger.debug(f"failed validating model {model_name}: {str(e)}")
            continue

    return valid_model_prefixes, ooa_valid_key_ids


def validate_model_paths(
    paths: List[str], handler_type: HandlerType, common_prefix: str
) -> List[int]:
    """
    To be used when handler:models:path or handler:models:paths in cortex.yaml is used.

    Args:
        paths: A list of all paths for a given s3/local prefix. Must be the top directory of a model.
        handler_type: Handler type. Can be PythonHandlerType, TensorFlowHandlerType or TensorFlowNeuronHandlerType.
        common_prefix: The common prefix of the directory which holds all models.

    Returns:
        List of all OneOfAllPlaceholder IDs that had been validated.

    Exception:
        CortexException if the paths don't match the model's template.
    """
    if len(paths) == 0:
        raise CortexException(
            f"{handler_type} handler at '{common_prefix}'", "model path can't be empty"
        )

    paths_by_prefix_cache = {}

    def _validate_model_paths(pattern: Any, paths: List[str], common_prefix: str) -> None:
        if common_prefix not in paths_by_prefix_cache:
            paths_by_prefix_cache[common_prefix] = util.get_paths_with_prefix(paths, common_prefix)
        paths = paths_by_prefix_cache[common_prefix]

        rel_paths = [os.path.relpath(path, common_prefix) for path in paths]
        rel_paths = [path for path in rel_paths if not path.startswith("../")]

        objects = [util.get_leftmost_part_of_path(path) for path in rel_paths]
        objects = list(set(objects))
        visited_objects = len(objects) * [False]

        ooa_valid_key_ids = []  # OneOfAllPlaceholder IDs that are valid

        if pattern is None:
            if len(objects) == 1 and objects[0] == ".":
                return ooa_valid_key_ids
            raise CortexException(
                f"{handler_type} handler at '{common_prefix}'",
                "template doesn't specify a substructure for the given path",
            )
        if not isinstance(pattern, dict):
            pattern = {pattern: None}

        keys = list(pattern.keys())
        keys.sort(key=operator.attrgetter("priority"))

        try:
            if any(isinstance(x, OneOfAllPlaceholder) for x in keys) and not all(
                isinstance(x, OneOfAllPlaceholder) for x in keys
            ):
                raise CortexException(
                    f"{handler_type} handler at '{common_prefix}'",
                    f"{OneOfAllPlaceholder()} is a mutual-exclusive key with all other keys",
                )
            elif all(isinstance(x, OneOfAllPlaceholder) for x in keys):
                num_keys = len(keys)
                num_validation_failures = 0

            for key_id, key in enumerate(keys):
                if key == IntegerPlaceholder:
                    _validate_integer_placeholder(keys, key_id, objects, visited_objects)
                elif key == AnyPlaceholder:
                    _validate_any_placeholder(keys, key_id, objects, visited_objects)
                elif key == SinglePlaceholder:
                    _validate_single_placeholder(keys, key_id, objects, visited_objects)
                elif isinstance(key, GenericPlaceholder):
                    _validate_generic_placeholder(keys, key_id, objects, visited_objects, key)
                elif isinstance(key, PlaceholderGroup):
                    _validate_group_placeholder(keys, key_id, objects, visited_objects)
                elif isinstance(key, OneOfAllPlaceholder):
                    try:
                        _validate_model_paths(pattern[key], paths, common_prefix)
                        ooa_valid_key_ids.append(key.ID)
                    except CortexException:
                        num_validation_failures += 1
                else:
                    raise CortexException("found a non-placeholder object in model template")

        except CortexException as e:
            raise CortexException(f"{handler_type} handler at '{common_prefix}'", str(e))

        if (
            all(isinstance(x, OneOfAllPlaceholder) for x in keys)
            and num_validation_failures == num_keys
        ):
            raise CortexException(
                f"couldn't validate for any of the {OneOfAllPlaceholder()} placeholders"
            )
        if all(isinstance(x, OneOfAllPlaceholder) for x in keys):
            return ooa_valid_key_ids

        unvisited_paths = []
        for idx, visited in enumerate(visited_objects):
            if visited is False:
                untraced_common_prefix = os.path.join(common_prefix, objects[idx])
                untraced_paths = [os.path.relpath(path, untraced_common_prefix) for path in paths]
                untraced_paths = [
                    os.path.join(objects[idx], path)
                    for path in untraced_paths
                    if not path.startswith("../")
                ]
                unvisited_paths += untraced_paths
        if len(unvisited_paths) > 0:
            raise CortexException(
                f"{handler_type} handler model at '{common_prefix}'",
                "unexpected path(s) for " + str(unvisited_paths),
            )

        new_common_prefixes = []
        sub_patterns = []
        paths_by_prefix = {}
        for obj_id, key_id in enumerate(visited_objects):
            obj = objects[obj_id]
            key = keys[key_id]
            if key != AnyPlaceholder:
                new_common_prefixes.append(os.path.join(common_prefix, obj))
                sub_patterns.append(pattern[key])

        if len(new_common_prefixes) > 0:
            paths_by_prefix = util.get_paths_by_prefixes(paths, new_common_prefixes)

        aggregated_ooa_valid_key_ids = []
        for sub_pattern, new_common_prefix in zip(sub_patterns, new_common_prefixes):
            aggregated_ooa_valid_key_ids += _validate_model_paths(
                sub_pattern, paths_by_prefix[new_common_prefix], new_common_prefix
            )

        return aggregated_ooa_valid_key_ids

    pattern = _single_model_pattern(handler_type)
    return _validate_model_paths(pattern, paths, common_prefix)


def _validate_integer_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    appearances = 0
    for idx, obj in enumerate(objects):
        if obj.isnumeric() and visited[idx] is False:
            visited[idx] = key_id
            appearances += 1

    if appearances > 1 and len(placeholders) > 1:
        raise CortexException(f"too many {IntegerPlaceholder} appearances in path")
    if appearances == 0:
        raise CortexException(f"{IntegerPlaceholder} not found in path")


def _validate_any_placeholder(
    placeholders: list,
    key_id: int,
    objects: List[str],
    visited: list,
) -> None:
    for idx, obj in enumerate(objects):
        if visited[idx] is False and obj != ".":
            visited[idx] = key_id


def _validate_single_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    if len(placeholders) > 1 or len(objects) > 1:
        raise CortexException(f"only a single {SinglePlaceholder} is allowed per directory")
    if len(visited) > 0 and visited[0] is False:
        visited[0] = key_id


def _validate_generic_placeholder(
    placeholders: list,
    key_id: int,
    objects: List[str],
    visited: list,
    generical: GenericPlaceholder,
) -> None:
    found = False
    for idx, obj in enumerate(objects):
        if obj == generical.value:
            if visited[idx] is False:
                visited[idx] = key_id
            found = True
            return

    if not found:
        raise CortexException(f"{generical.type} placeholder for {generical} wasn't found")


def _validate_group_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    """
    Can use AnyPlaceholder, GenericPlaceholder, SinglePlaceholder.

    The minimum number of placeholders a group must hold is 2.

    The accepted formats are:
    - ... AnyPlaceholder, GenericPlaceholder, AnyPlaceholder, ...
    - ... SinglePlaceholder, GenericPlaceholder, SinglePlaceholder, ...

    AnyPlaceholder and SinglePlaceholder cannot be mixed together in one group.
    """

    placeholder_group = placeholders[key_id]

    if len(placeholder_group) < 2:
        raise CortexException(f"{placeholder_group} must come with at least 2 placeholders")

    for placeholder in placeholder_group:
        if placeholder not in [AnyPlaceholder, SinglePlaceholder] and not isinstance(
            placeholder, GenericPlaceholder
        ):
            raise CortexException(
                f'{placeholder_group} must have a combination of the following placeholder types: {AnyPlaceholder}, {SinglePlaceholder}, {GenericPlaceholder("").placeholder}'
            )

    if {AnyPlaceholder, SinglePlaceholder}.issubset(set(placeholder_group)):
        raise CortexException(
            f"{placeholder_group} cannot have a mix of the following placeholder types: {AnyPlaceholder} and {SinglePlaceholder}"
        )

    group_len = len(placeholder_group)
    for idx in range(group_len):
        if idx + 1 < group_len:
            a = placeholder_group[idx]
            b = placeholder_group[idx + 1]
            if a == b:
                raise CortexException(
                    f'{placeholder_group} cannot accept the same type to be specified consecutively ({AnyPlaceholder}, {SinglePlaceholder} or {GenericPlaceholder("").placeholder})'
                )

    pattern = ""
    for placeholder in placeholder_group:
        if placeholder in [AnyPlaceholder, SinglePlaceholder]:
            pattern += "*"
        if isinstance(placeholder, GenericPlaceholder):
            pattern += str(placeholder.value)

    num_occurences = 0
    for idx, obj in enumerate(objects):
        if visited[idx] is False and fnmatchcase(obj, pattern):
            visited[idx] = key_id
            num_occurences += 1

    if SinglePlaceholder in placeholder_group and num_occurences > 1:
        raise CortexException(
            f"{placeholder_group} must match once (not {num_occurences} times) because {SinglePlaceholder} is present"
        )
