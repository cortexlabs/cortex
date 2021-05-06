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
import os
import pathlib
import shutil
import zipfile
from copy import deepcopy
from typing import List, Dict, Optional


def has_method(object, method: str):
    return callable(getattr(object, method, None))


def expand_environment_vars_on_file(in_file: str, out_file: Optional[str] = None):
    if out_file is None:
        out_file = in_file
    with open(in_file, "r") as f:
        data = f.read()
    with open(out_file, "w") as f:
        f.write(os.path.expandvars(data))


def extract_zip(zip_path, dest_dir=None, delete_zip_file=False):
    if dest_dir is None:
        dest_dir = os.path.dirname(zip_path)

    zip_ref = zipfile.ZipFile(zip_path, "r")
    zip_ref.extractall(dest_dir)
    zip_ref.close()

    if delete_zip_file:
        rm_file(zip_path)


def mkdir_p(dir_path):
    pathlib.Path(dir_path).mkdir(parents=True, exist_ok=True)


def rm_dir(dir_path):
    if os.path.isdir(dir_path):
        shutil.rmtree(dir_path)
        return True
    return False


def rm_file(path):
    if os.path.isfile(path):
        os.remove(path)
        return True
    return False


def trim_prefix(string, prefix):
    if string.startswith(prefix):
        return string[len(prefix) :]
    return string


def ensure_prefix(string, prefix):
    if string.startswith(prefix):
        return string
    return prefix + string


def trim_suffix(string, suffix):
    if string.endswith(suffix):
        return string[: -len(suffix)]
    return string


def ensure_suffix(string, suffix):
    if string.endswith(suffix):
        return string
    return string + suffix


def get_paths_with_prefix(paths: List[str], prefix: str) -> List[str]:
    return list(filter(lambda path: path.startswith(prefix), paths))


def get_paths_by_prefixes(paths: List[str], prefixes: List[str]) -> Dict[str, List[str]]:
    paths_by_prefix = {}
    for path in paths:
        for prefix in prefixes:
            if not path.startswith(prefix):
                continue
            if prefix not in paths_by_prefix:
                paths_by_prefix[prefix] = [path]
            else:
                paths_by_prefix[prefix].append(path)
    return paths_by_prefix


def get_leftmost_part_of_path(path: str) -> str:
    """
    Gets the leftmost part of a path.

    If a path looks like
    models/tensorflow/iris/15559399

    Then this function will return
    models
    """
    if path == "." or path == "./":
        return "."
    return pathlib.PurePath(path).parts[0]


def remove_non_empty_directory_paths(paths: List[str]) -> List[str]:
    """
    Eliminates dir paths from the tree that are not empty.

    If paths looks like:
    models/tensorflow/
    models/tensorflow/iris/1569001258
    models/tensorflow/iris/1569001258/saved_model.pb

    Then after calling this function, it will look like:
    models/tensorflow/iris/1569001258/saved_model.pb
    """

    leading_slash_paths_mask = [path.startswith("/") for path in paths]
    all_paths_start_with_leading_slash = all(leading_slash_paths_mask)
    some_paths_start_with_leading_slash = any(leading_slash_paths_mask)

    if not all_paths_start_with_leading_slash and some_paths_start_with_leading_slash:
        raise ValueError("can only either pass in absolute paths or relative paths")

    path_map = {}
    split_paths = [list(filter(lambda x: x != "", path.split("/"))) for path in paths]

    for split_path in split_paths:
        composed_path = ""
        split_path_length = len(split_path)
        for depth, path_level in enumerate(split_path):
            if composed_path != "":
                composed_path += "/"
            composed_path += path_level
            if composed_path not in path_map:
                path_map[composed_path] = 1
                if depth < split_path_length - 1:
                    path_map[composed_path] += 1
            else:
                path_map[composed_path] += 1

    file_paths = []
    for file_path, appearances in path_map.items():
        if appearances == 1:
            file_paths.append(all_paths_start_with_leading_slash * "/" + file_path)

    return file_paths


def merge_dicts_in_place_overwrite(*dicts):
    """Merge dicts, right into left, with overwriting. First dict is updated in place"""
    dicts = list(dicts)
    target = dicts.pop(0)
    for d in dicts:
        merge_two_dicts_in_place_overwrite(target, d)
    return target


def merge_dicts_in_place_no_overwrite(*dicts):
    """Merge dicts, right into left, without overwriting. First dict is updated in place"""
    dicts = list(dicts)
    target = dicts.pop(0)
    for d in dicts:
        merge_two_dicts_in_place_no_overwrite(target, d)
    return target


def merge_dicts_overwrite(*dicts):
    """Merge dicts, right into left, with overwriting. A new dict is created, original ones not modified."""
    result = {}
    for d in dicts:
        result = merge_two_dicts_overwrite(result, d)
    return result


def merge_dicts_no_overwrite(*dicts):
    """Merge dicts, right into left, without overwriting. A new dict is created, original ones not modified."""
    result = {}
    for d in dicts:
        result = merge_two_dicts_no_overwrite(result, d)
    return result


def merge_two_dicts_in_place_overwrite(x, y):
    """Merge y into x, with overwriting. x is updated in place"""
    if x is None:
        x = {}

    if y is None:
        y = {}

    for k, v in y.items():
        if k in x and isinstance(x[k], dict) and isinstance(y[k], collections.Mapping):
            merge_dicts_in_place_overwrite(x[k], y[k])
        else:
            x[k] = y[k]
    return x


def merge_two_dicts_in_place_no_overwrite(x, y):
    """Merge y into x, without overwriting. x is updated in place"""
    for k, v in y.items():
        if k in x and isinstance(x[k], dict) and isinstance(y[k], collections.Mapping):
            merge_dicts_in_place_no_overwrite(x[k], y[k])
        else:
            if k not in x:
                x[k] = y[k]
    return x


def merge_two_dicts_overwrite(x, y):
    """Merge y into x, with overwriting. A new dict is created, original ones not modified."""
    x = deepcopy(x)
    return merge_dicts_in_place_overwrite(x, y)


def merge_two_dicts_no_overwrite(x, y):
    """Merge y into x, without overwriting. A new dict is created, original ones not modified."""
    y = deepcopy(y)
    return merge_dicts_in_place_overwrite(y, x)


def is_bool(var):
    return isinstance(var, bool)


def is_float(var):
    return isinstance(var, float)


def is_int(var):
    return isinstance(var, int) and not isinstance(var, bool)


def is_str(var):
    return isinstance(var, str)


def is_dict(var):
    return isinstance(var, dict)


def is_list(var):
    return isinstance(var, list)


def is_tuple(var):
    return isinstance(var, tuple)


def is_float_or_int(var):
    return is_int(var) or is_float(var)


def is_int_list(var):
    if not is_list(var):
        return False
    for item in var:
        if not is_int(item):
            return False
    return True


def is_float_list(var):
    if not is_list(var):
        return False
    for item in var:
        if not is_float(item):
            return False
    return True


def is_str_list(var):
    if not is_list(var):
        return False
    for item in var:
        if not is_str(item):
            return False
    return True


def is_bool_list(var):
    if not is_list(var):
        return False
    for item in var:
        if not is_bool(item):
            return False
    return True


def is_float_or_int_list(var):
    if not is_list(var):
        return False
    for item in var:
        if not is_float_or_int(item):
            return False
    return True


def and_list_with_quotes(values: List) -> str:
    """
    Converts a list like ["a", "b", "c"] to '"a", "b" and "c"'".
    """
    string = ""

    if len(values) == 1:
        string = '"' + values[0] + '"'
    elif len(values) > 1:
        for val in values[:-2]:
            string += '"' + val + '", '
        string += '"' + values[-2] + '" and "' + values[-1] + '"'

    return string


def or_list_with_quotes(values: List) -> str:
    """
    Converts a list like ["a", "b", "c"] to '"a", "b" or "c"'.
    """
    string = ""

    if len(values) == 1:
        string = '"' + values[0] + '"'
    elif len(values) > 1:
        for val in values[:-2]:
            string += '"' + val + '", '
        string += '"' + values[-2] + '" or "' + values[-1] + '"'

    return string


def string_plural_with_s(string: str, count: int) -> str:
    """
    Pluralize the word with an "s" character if the count is greater than 1.
    """
    if count > 1:
        string += "s"
    return string


def render_jinja_template(jinja_template_file: str, context: dict) -> str:
    from jinja2 import Environment, FileSystemLoader

    template_path = pathlib.Path(jinja_template_file)

    env = Environment(loader=FileSystemLoader(str(template_path.parent)))
    env.trim_blocks = True
    env.lstrip_blocks = True
    env.rstrip_blocks = True

    template = env.get_template(str(template_path.name))
    return template.render(**context)
