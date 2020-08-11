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
import operator
from typing import List, Any

from cortex.lib import util
from cortex.lib.storage import S3, LocalStorage
from cortex.lib.log import cx_logger
from cortex.lib.exceptions import CortexException
from cortex.lib.model import ModelsTree
from cortex.lib.api import (
    PythonPredictorType,
    TensorFlowPredictorType,
    ONNXPredictorType,
    PredictorType,
)

import collections
from fnmatch import fnmatchcase


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
        if isinstance(other, GenericPlaceholder):
            return self.placeholder == other.placeholder
        return False

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
    Order-based addition of placeholder types (Groups, Generics or Templates).

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

    Accessible properties: parts, type, priority.
    """

    def __init__(self):
        self._placeholder = TemplatePlaceholder("oneofall", priority=-1)

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

TensorFlowNeuronPredictor = PredictorType("tensorflow-neuron")

# to be used when predictor:model_path or predictor:models:paths is used
ModelTemplate = {
    PythonPredictorType: {IntegerPlaceholder: AnyPlaceholder},
    TensorFlowPredictorType: {
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
    TensorFlowNeuronPredictor: {IntegerPlaceholder: {GenericPlaceholder("saved_model.pb"): None}},
    ONNXPredictorType: {
        OneOfAllPlaceholder(): {
            IntegerPlaceholder: {
                PlaceholderGroup(SinglePlaceholder, GenericPlaceholder(".onnx")): None,
            },
        },
        OneOfAllPlaceholder(): {
            PlaceholderGroup(SinglePlaceholder, GenericPlaceholder(".onnx")): None,
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


def dir_models_pattern(predictor_type: PredictorType) -> dict:
    """
    To be used when predictor:models:dir in cortex.yaml is used.
    """
    return {SinglePlaceholder: ModelTemplate[predictor_type]}


def single_model_pattern(predictor_type: PredictorType) -> dict:
    """
    To be used when predictor:model_path or predictor:models:paths in cortex.yaml is used.
    """
    return ModelTemplate[predictor_type]


def validate_s3_models_dir_paths(
    s3_paths: List[str], predictor_type: PredictorType, commonprefix: str
) -> List[str]:
    """
    To be used when predictor:models:dir in cortex.yaml is used.

    Args:
        s3_paths: A list of all paths for a given S3 prefix. Must be the top directory of multiple models.
        predictor_type: Predictor type. Can be PythonPredictorType, TensorFlowPredictorType, TensorFlowNeuronPredictorType or ONNXPredictorType.
        commonprefix: The commonprefix of the directory which holds all models.

    Returns:
        A list with the prefix of each model that's valid.
    """
    if len(s3_paths) == 0:
        raise CortexException(
            f"{predictor_type} predictor at '{commonprefix}'", "model top path can't be empty"
        )

    pattern = dir_models_pattern(predictor_type)

    paths = [os.path.relpath(s3_top_path, commonprefix) for s3_top_path in s3_paths]
    paths = [path for path in paths if not path.startswith("../")]

    model_names = [get_leftmost_part_of_path(path) for path in paths]
    model_names = list(set(model_names))

    valid_model_prefixes = []
    for idx, model_name in enumerate(model_names):
        try:
            validate_s3_model_paths(paths, predictor_type, model_name)
            valid_model_prefixes.append(os.path.join(commonprefix, model_name))
        except CortexException as e:
            continue

    return valid_model_prefixes


def validate_s3_model_paths(
    s3_paths: List[str], predictor_type: PredictorType, commonprefix: str
) -> None:
    """
    To be used when predictor:model_path or predictor:models:paths in cortex.yaml is used.

    Args:
        s3_top_paths: A list of all paths for a given S3 prefix. Must be the top directory of a model.
        predictor_type: Predictor type. Can be PythonPredictorType, TensorFlowPredictorType, TensorFlowNeuronPredictorType or ONNXPredictorType.
        commonprefix: The commonprefix of the directory which holds all models.

    Exception:
        CortexException if the paths don't match the model's template.
    """
    if len(s3_paths) == 0:
        raise CortexException(
            f"{predictor_type} predictor at '{commonprefix}'", "model path can't be empty"
        )

    def _validate_s3_model_paths(pattern: Any, s3_paths: List[str], commonprefix: str) -> None:
        paths = [os.path.relpath(s3_path, commonprefix) for s3_path in s3_paths]
        paths = [path for path in paths if not path.startswith("../")]

        objects = [get_leftmost_part_of_path(path) for path in paths]
        objects = list(set(objects))
        visited_objects = len(objects) * [False]

        if pattern is None:
            if len(objects) == 1 and objects[0] == ".":
                return
            raise CortexException(
                f"{predictor_type} predictor at '{commonprefix}'",
                "template doesn't specify a substructure for the given path",
            )
        if not isinstance(pattern, dict):
            pattern = {pattern: None}

        keys = list(pattern.keys())
        keys.sort(key=operator.attrgetter("priority"))

        try:
            if (
                any(isinstance(x, OneOfAllPlaceholder) for x in keys)
                and not all(isinstance(x, OneOfAllPlaceholder) for x in keys)
                and len(set(keys)) > 1
            ):
                raise CortexException(
                    f"{predictor_type} predictor at '{commonprefix}'",
                    f"{OneOfAllPlaceholder()} is a mutual-exclusive key with all other keys",
                )
            elif all(isinstance(x, OneOfAllPlaceholder) for x in keys):
                num_keys = len(keys)
                num_validation_failures = 0

            for key_id, key in enumerate(keys):
                if key == IntegerPlaceholder:
                    validate_integer_placeholder(keys, key_id, objects, visited_objects)
                elif key == AnyPlaceholder:
                    validate_any_placeholder(keys, key_id, objects, visited_objects)
                elif key == SinglePlaceholder:
                    validate_single_placeholder(keys, key_id, objects, visited_objects)
                elif key == GenericPlaceholder(""):
                    validate_generic_placeholder(keys, key_id, objects, visited_objects, key)
                elif isinstance(key, PlaceholderGroup):
                    validate_group_placeholder(keys, key_id, objects, visited_objects)
                elif isinstance(key, OneOfAllPlaceholder):
                    try:
                        _validate_s3_model_paths(pattern[key], s3_paths, commonprefix)
                    except CortexException:
                        num_validation_failures += 1
                else:
                    raise CortexException("found a non-placeholder object in model template")

        except CortexException as e:
            raise CortexException(f"{predictor_type} predictor at '{commonprefix}'", str(e))

        if (
            all(isinstance(x, OneOfAllPlaceholder) for x in keys)
            and num_validation_failures == num_keys
        ):
            raise CortexException(
                f"couldn't validate for any of the {OneOfAllPlaceholder()} placeholders"
            )
        if all(isinstance(x, OneOfAllPlaceholder) for x in keys):
            return

        unvisited_paths = []
        for idx, visited in enumerate(visited_objects):
            if visited is False:
                untraced_common_prefix = os.path.join(commonprefix, objects[idx])
                untraced_paths = [
                    os.path.relpath(path, untraced_common_prefix) for path in s3_paths
                ]
                untraced_paths = [
                    os.path.join(objects[idx], path)
                    for path in untraced_paths
                    if not path.startswith("../")
                ]
                unvisited_paths += untraced_paths
        if len(unvisited_paths) > 0:
            raise CortexException(
                f"{predictor_type} predictor model at '{commonprefix}'",
                "unexpected path(s) for " + str(unvisited_paths),
            )

        for obj_id, key_id in enumerate(visited_objects):
            obj = objects[obj_id]
            key = keys[key_id]

            new_commonprefix = os.path.join(commonprefix, obj)
            sub_pattern = pattern[key]

            if key != AnyPlaceholder:
                _validate_s3_model_paths(sub_pattern, s3_paths, new_commonprefix)

    pattern = single_model_pattern(predictor_type)
    _validate_s3_model_paths(pattern, s3_paths, commonprefix)


def get_leftmost_part_of_path(path: str) -> str:
    basename = ""
    while path:
        path, basename = os.path.split(path)
    return basename


def validate_integer_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    appearances = 0
    for idx, obj in enumerate(objects):
        if obj.isnumeric() and visited[idx] is False:
            visited[idx] = key_id
            appearances += 1

    if appearances > 1 and len(placeholders) > 0:
        raise CortexException(f"too many {IntegerPlaceholder} appearances in path")
    if appearances == 0:
        raise CortexException(f"{IntegerPlaceholder} not found in path")


def validate_any_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list,
) -> None:
    for idx, obj in enumerate(objects):
        if visited[idx] is False and obj != ".":
            visited[idx] = key_id


def validate_single_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    if len(placeholders) > 1 or len(objects) > 1:
        raise CortexException(f"only a single {SinglePlaceholder} is allowed per directory")
    if len(visited) > 0 and visited[0] is False:
        visited[0] = key_id


def validate_generic_placeholder(
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
            break

    if not found:
        raise CortexException(f"{generical.type} placeholder for {generical} wasn't found")


def validate_group_placeholder(
    placeholders: list, key_id: int, objects: List[str], visited: list
) -> None:
    """
    Can use AnyPlaceholder, GenericPlaceholder, SinglePlaceholder.

    The minimum number of placeholders a group must hold is 2.

    The accepted formats are:
    - ... AnyPlaceholder, GenericPlaceholder, AnyPlaceholder, ...
    - ... SinglePlaceholder, GenericPlaceholder, SinglePlaceholder, ...

    AnyPlaceholder and GenericPlaceholder cannot be mixed together in one group.
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

    if util.is_subset([AnyPlaceholder, SinglePlaceholder], placeholder_group):
        raise CortexException(
            f"{placeholder_group} cannot have a mix of the following placeholder types: {AnyPlaceholder} and {SinglePlaceholder}"
        )

    group_len = len(placeholder_group)
    found_same_kind_together = False
    for idx in range(group_len):
        if idx + 1 < group_len:
            a = placeholder_group[idx]
            b = placeholder_group[idx + 1]
            if a == b:
                found_same_kind_together = True
                break

    if found_same_kind_together:
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
