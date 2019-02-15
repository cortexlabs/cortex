# Copyright 2019 Cortex Labs, Inc.
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
import yaml
import errno
import shutil
import sys
import stat
import pprint
import pickle
import json
import collections
import tempfile
import zipfile
import hashlib
import marshal
import msgpack
from copy import deepcopy
from datetime import datetime

import consts

from lib.log import get_logger

logger = get_logger()


def isclose(a, b, rel_tol=1e-09, abs_tol=0.0):
    return abs(a - b) <= max(rel_tol * max(abs(a), abs(b)), abs_tol)


def indent_str(text, indent):
    if not is_str(text):
        text = repr(text)
    return indent * " " + text.replace("\n", "\n" + indent * " ")


def pp_str(obj, indent=0):
    try:
        out = json.dumps(obj, sort_keys=True, indent=2)
    except:
        out = pprint.pformat(obj, width=120)
    return indent_str(out, indent)


def pp_str_flat(obj, indent=0):
    try:
        out = json.dumps(obj, sort_keys=True)
    except:
        out = str(obj).replace("\n", "")
    return indent_str(out, indent)


def log_indent(obj, indent=0, logging_func=logger.info):
    if not is_str(obj):
        text = repr(obj)
    else:
        text = obj
    logging_func(indent_str(text, indent))


def log_pretty(obj, indent=0, logging_func=logger.info):
    formatted_str = pp_str(obj, indent)
    for line in formatted_str.split("\n"):
        logging_func(line)


def log_pretty_flat(obj, indent=0, logging_func=logger.info):
    logging_func(pp_str_flat(obj), indent)


def pluralize(num, singular, plural):
    if num == 1:
        return str(num) + " " + singular
    else:
        return str(num) + " " + plural


def mkdir_p(dir_path):
    try:
        os.makedirs(dir_path)
    except OSError as e:
        if e.errno == errno.EEXIST and os.path.isdir(dir_path):
            pass
        else:
            raise


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


def list_files_recursive(dir_path, ignore=None):
    all_file_names = []
    for root, _, file_names in os.walk(dir_path):
        if ignore:
            file_names = set(file_names) - ignore(root, file_names)
        for file_name in file_names:
            all_file_names.append(os.path.join(root, file_name))
    return all_file_names


# https://stackoverflow.com/questions/1868714/how-do-i-copy-an-entire-directory-of-files-into-an-existing-directory-using-pyth
def cp_dir(src, dst, symlinks=False, ignore=None):
    if not os.path.exists(dst):
        os.makedirs(dst)
        shutil.copystat(src, dst)
    lst = os.listdir(src)
    if ignore:
        excl = ignore(src, lst)
        lst = [x for x in lst if x not in excl]
    for item in lst:
        s = os.path.join(src, item)
        d = os.path.join(dst, item)
        if symlinks and os.path.islink(s):
            if os.path.lexists(d):
                os.remove(d)
            os.symlink(os.readlink(s), d)
            try:
                st = os.lstat(s)
                mode = stat.S_IMODE(st.st_mode)
                os.lchmod(d, mode)
            except:
                pass  # lchmod not available
        elif os.path.isdir(s):
            cp_dir(s, d, symlinks, ignore)
        else:
            shutil.copy2(s, d)


temp_files_ignore = shutil.ignore_patterns("*.pyc", ".*")


def non_code_ignore(path, names):
    good_patterns = ["*.py", "*.yaml", "*.yml"]
    good_names = []
    for pattern in good_patterns:
        good_names.extend(fnmatch.filter(names, pattern))
    dirs = [f for f in names if os.path.isdir(os.path.join(path, f))]
    return set(names) - set(good_names) - set(dirs)


def make_temp_dir(parent_dir=None, prefix="tmp-", suffix=""):
    return tempfile.mkdtemp(suffix=suffix, prefix=prefix, dir=parent_dir)


class Tempdir:
    def __init__(self, parent_dir=None, prefix="tmp-", suffix=""):
        self.parent_dir = parent_dir
        self.prefix = prefix
        self.suffix = suffix

    def __enter__(self):
        self.temp_dir = make_temp_dir(self.parent_dir, self.prefix, self.suffix)
        return self.temp_dir

    def __exit__(self, type, value, traceback):
        rm_dir(self.temp_dir)


def get_timestamp_from_datetime(dt, delimiter="-"):
    time_format = delimiter.join(["%Y", "%m", "%d", "%H", "%M", "%S", "%f"])
    return dt.strftime(time_format)


def format_datetime(dt):
    time_format = "%Y-%m-%d %H:%M:%S"
    return dt.strftime(time_format)


def get_now_formatted():
    return format_datetime(datetime.utcnow())


def now_timestamp_rfc_3339():
    return datetime.utcnow().isoformat("T") + "Z"


def remove_prefix_if_present(string, prefix):
    if string.startswith(prefix):
        return string[len(prefix) :]
    return string


def add_prefix_unless_present(string, prefix):
    if string.startswith(prefix):
        return string
    return prefix + string


def remove_suffix_if_present(string, suffix):
    if string.endswith(suffix):
        return string[: -len(suffix)]
    return string


def add_suffix_unless_present(string, suffix):
    if string.endswith(suffix):
        return string
    return string + suffix


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


# Note: this converts tuples to lists
def flatten(var):
    if is_list(var) or is_tuple(var):
        return [a for i in var for a in flatten(i)]
    else:
        return [var]


def subtract_lists(lst1, lst2):
    return [e for e in lst1 if e not in lst2]


def keep_dict_keys(d, keys):
    key_set = set(keys)
    for key in list(d.keys()):
        if key not in key_set:
            del d[key]


def create_multi_map(d, key_func):
    """Create a dictionary that returns a list of values for a specific key.

    Args:
        d (dict): The dictionary to convert to multimap.
        key_func (callable): A callable that gets called with the key and value to create a new ID.

    Returns:
        type: A dict of lists.

    """
    new_dict = {}
    for k, v in d.items():
        new_key = key_func(k, v)
        ls = new_dict.get(new_key, [])
        ls.append(v)
        new_dict[new_key] = ls
    return new_dict


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


def normalize_path(path, rel_dir):
    if os.path.isabs(path):
        return path
    else:
        return os.path.normpath(os.path.join(rel_dir, path))


def read_file(path):
    if not os.path.isfile(path):
        return None
    with open(path, "r") as file:
        return file.read()


def read_file_strip(path):
    contents = read_file(path)
    if is_str(contents):
        contents = contents.strip()
    return contents


def read_yaml(yaml_path):
    with open(yaml_path, "r") as yaml_file:
        return yaml.safe_load(yaml_file)


def read_json(json_path):
    with open(json_path, "r") as json_file:
        return json.load(json_file)


def read_msgpack(msgpack_path):
    with open(msgpack_path, "rb") as msgpack_file:
        return msgpack.load(msgpack_file, raw=False)


def write_yaml(dictionary, file_path):
    with open(file_path, "w") as file:
        file.write(yaml.dump(dictionary))


def hash_str(string, alg="sha256"):
    hashing_alg = getattr(hashlib, alg)
    return hashing_alg(string).hexdigest()


def hash_file(file_path, alg="sha256"):
    file_str = read_file(file_path)
    return hash_str(file_str, alg)


def zip_dir(src_dir, dest_path, nest_dir=False, ignore=None):
    """Note: files are added at the root level of the zip, unless nest_dir=True"""
    if nest_dir:
        root = os.path.basename(src_dir)
    else:
        root = ""

    return zip_dispersed_files(src_files=[(src_dir, root)], dest_path=dest_path, ignore=ignore)


def zip_files(
    src_files,
    dest_path,
    flatten=False,
    remove_common_prefix=False,
    remove_prefix="",
    add_prefix="",
    empty_files=[],
    ignore=None,
    allow_missing_files=False,
):
    """src_files is a list of strings (path_to_file/dir)"""
    dest_path = add_suffix_unless_present(dest_path, ".zip")
    add_prefix = remove_prefix_if_present(add_prefix, "/")

    if remove_prefix != "" and not remove_prefix.endswith("/"):
        remove_prefix = remove_prefix + "/"

    common_prefix = ""
    if remove_common_prefix:
        common_prefix = os.path.commonprefix(src_files)

    src_dirs = [f for f in src_files if f and os.path.isdir(f)]
    src_files = [f for f in src_files if f and os.path.isfile(f)]
    for src_dir in src_dirs:
        src_files += list_files_recursive(src_dir, ignore=ignore)

    with zipfile.ZipFile(dest_path, "w", zipfile.ZIP_DEFLATED) as myzip:
        for empty_file_path in empty_files:
            empty_file_path = remove_prefix_if_present(empty_file_path, "/")
            myzip.writestr(empty_file_path, "")

        for src_file in src_files:
            if flatten:
                zip_name = os.path.basename(src_file)
            else:
                zip_name = src_file
                zip_name = remove_prefix_if_present(zip_name, remove_prefix)
                zip_name = remove_prefix_if_present(zip_name, common_prefix)

            zip_name = os.path.join(add_prefix, zip_name)
            if allow_missing_files:
                if os.path.isfile(src_file):
                    myzip.write(src_file, arcname=zip_name)
            else:
                myzip.write(src_file, arcname=zip_name)


def zip_dispersed_files(
    src_files, dest_path, add_prefix="", empty_files=[], ignore=None, allow_missing_files=False
):
    """src_files is a list of tuples (path_to_file/dir, path_in_zip)"""
    dest_path = add_suffix_unless_present(dest_path, ".zip")
    add_prefix = remove_prefix_if_present(add_prefix, "/")

    src_dirs = [(f, p) for f, p in src_files if f and os.path.isdir(f)]
    src_files = [(f, p) for f, p in src_files if f and os.path.isfile(f)]
    for src_dir, zip_name in src_dirs:
        for src_file in list_files_recursive(src_dir, ignore=ignore):
            common_prefix = os.path.commonprefix([src_dir, src_file])
            rel_src_file = remove_prefix_if_present(src_file, common_prefix)
            rel_src_file = remove_prefix_if_present(rel_src_file, "/")
            updated_zip_name = os.path.join(zip_name, rel_src_file)
            src_files.append((src_file, updated_zip_name))

    with zipfile.ZipFile(dest_path, "w", zipfile.ZIP_DEFLATED) as myzip:
        for empty_file_path in empty_files:
            empty_file_path = remove_prefix_if_present(empty_file_path, "/")
            myzip.writestr(empty_file_path, "")

        for src_file, zip_name in src_files:
            zip_name = remove_prefix_if_present(zip_name, "/")
            zip_name = os.path.join(add_prefix, zip_name)
            if allow_missing_files:
                if os.path.isfile(src_file):
                    myzip.write(src_file, arcname=zip_name)
            else:
                myzip.write(src_file, arcname=zip_name)


def extract_zip(zip_path, dest_dir=None, delete_zip_file=False):
    if dest_dir is None:
        dest_dir = os.path.dirname(zip_path)

    zip_ref = zipfile.ZipFile(zip_path, "r")
    zip_ref.extractall(dest_dir)
    zip_ref.close()

    if delete_zip_file:
        rm_file(zip_path)


# config_path may already be a dict
def read_config(config_path, defaults={}):
    if is_str(config_path):
        try:
            config = read_yaml(config_path)
        except IOError as e:
            config = {}
    else:
        config = config_path

    config = config or {}
    return merge_dicts_no_overwrite(config, defaults)


# The order for maps is deterministic. Returns a list
def flatten_all_values(obj):
    if is_list(obj):
        flattened_values = []
        for e in obj:
            flattened_values += flatten_all_values(e)
        return flattened_values

    if is_dict(obj):
        flattened_values = []
        for key in sorted(obj.keys()):
            flattened_values += flatten_all_values(obj[key])
        return flattened_values

    return [obj]


def print_samples_horiz(
    samples, truncate=20, sep=",", first_sep=":", pad=1, first_pad=None, key_list=None
):
    if first_pad is None:
        first_pad = pad

    if not key_list:
        field_names = sorted(list(set(flatten([list(sample.keys()) for sample in samples]))))
    else:
        field_names = key_list

    rows = []
    for field_name in field_names:
        rows.append([field_name] + [sample.get(field_name, None) for sample in samples])

    rows_strs = []
    for row in rows:
        row_strs = []
        for i, item in enumerate(row):
            row_strs.append(str_rep(item, truncate))
        rows_strs.append(row_strs)

    max_lens = []
    for i in range(len(rows_strs[0])):
        max_lens.append(max_len([row[i] for row in rows_strs]))

    types = []
    for i in range(len(rows[0])):
        types.append(is_number_col([row[i] for row in rows]))

    for row_strs in rows_strs:
        row_str = ""
        for i, item in enumerate(row_strs):
            if i == 0:
                row_str += pad_smart(item + first_sep, max_lens[i] + len(first_sep), types[i])
                row_str += " " * first_pad
            elif i == len(row_strs) - 1:
                row_str += pad_smart(item, max_lens[i], types[i]).rstrip()
            else:
                row_str += pad_smart(item + sep, max_lens[i] + len(sep), types[i])
                row_str += " " * pad

        logger.info(row_str.strip())


def max_len(strings):
    return max(len(s) for s in strings)


def pad_smart(string, width, is_number):
    if is_number:
        return pad_left(string, width)
    else:
        return pad_right(string, width)


def pad_right(string, width):
    return string.ljust(width)


def pad_left(string, width):
    return string.rjust(width)


def str_rep(item, truncate=20, round=2):
    if item is None:
        return ""
    if is_float(item):
        out = "{0:.2f}".format(item)
    else:
        out = str(item)

    return truncate_str(out, truncate)


def truncate_str(item, truncate=20):
    if is_str(item) and truncate is not None and truncate > 3 and len(item) > truncate:
        trim = truncate - 3
        return item[:trim] + "..."
    return item


def is_number_col(items):
    if all(item is None for item in items):
        return False

    for item in items:
        if not is_int(item) and not is_float(item) and item != None:
            return False

    return True


def log_job_finished(workload_id):
    timestamp = now_timestamp_rfc_3339()
    logger.info("workload: {}, completed: {}".format(workload_id, timestamp))


CORTEX_TYPE_TO_VALIDATOR = {
    consts.COLUMN_TYPE_INT: is_int,
    consts.COLUMN_TYPE_INT_LIST: is_int_list,
    consts.COLUMN_TYPE_FLOAT: is_float,
    consts.COLUMN_TYPE_FLOAT_LIST: is_float_list,
    consts.COLUMN_TYPE_STRING: is_str,
    consts.COLUMN_TYPE_STRING_LIST: is_str_list,
    consts.VALUE_TYPE_INT: is_int,
    consts.VALUE_TYPE_FLOAT: is_float,
    consts.VALUE_TYPE_STRING: is_str,
    consts.VALUE_TYPE_BOOL: is_bool,
}

CORTEX_TYPE_TO_UPCAST_VALIDATOR = merge_dicts_overwrite(
    CORTEX_TYPE_TO_VALIDATOR,
    {
        consts.COLUMN_TYPE_FLOAT: is_float_or_int,
        consts.COLUMN_TYPE_FLOAT_LIST: is_float_or_int_list,
    },
)

CORTEX_TYPE_TO_UPCASTER = {
    consts.COLUMN_TYPE_FLOAT: lambda x: float(x),
    consts.COLUMN_TYPE_FLOAT_LIST: lambda ls: [float(item) for item in ls],
}


def upcast(value, column_type):
    upcaster = CORTEX_TYPE_TO_UPCASTER.get(column_type, None)
    if upcaster:
        return upcaster(value)
    return value


def validate_column_type(value, column_type):
    if value is None:
        return True

    if not is_str(column_type):
        raise

    valid_types = column_type.split("|")
    for valid_type in valid_types:
        if CORTEX_TYPE_TO_VALIDATOR[valid_type](value):
            return True

    return False


def validate_value_type(value, value_type):
    if value is None:
        return True

    if is_str(value_type):
        valid_types = value_type.split("|")

        for valid_type in valid_types:
            if CORTEX_TYPE_TO_VALIDATOR[valid_type](value):
                return True

        return False

    if is_dict(value_type):
        if not is_dict(value):
            return False
        if len(value_type) == 0:
            if len(value) == 0:
                return True
            return False

        is_generic_map = False
        if len(value_type) == 1:
            value_type_key = next(iter(value_type.keys()))
            if value_type_key in consts.VALUE_TYPES:
                is_generic_map = True
                generic_map_key = value_type_key
                generic_map_value = value_type[value_type_key]

        if is_generic_map:
            for value_key, value_val in value.items():
                if not validate_value_type(value_key, generic_map_key):
                    return False
                if not validate_value_type(value_val, generic_map_value):
                    return False
            return True

        if len(value) != len(value_type):
            return False
        for value_key, value_val in value.items():
            if value_key not in value_type:
                return False
            if not validate_value_type(value_val, value_type[value_key]):
                return False
        return True

    if is_list(value_type):
        if not (len(value_type) == 1 and is_str(value_type[0])):
            return False
        if not is_list(value):
            return False
        for value_item in value:
            if not validate_value_type(value_item, value_type[0]):
                return False
        return True

    return False
