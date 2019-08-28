
import json_tricks
import pprint

TRUNCATE_LIMIT = 75


def indent_str(text, indent):
    if isinstance(text, str):
        text = repr(text)
    return indent * " " + text.replace("\n", "\n" + indent * " ")


def json_tricks_dump(obj, **kwargs):
    return json_tricks.dumps(obj, primitives=True, **kwargs)


def json_tricks_encoder(*args, **kwargs):
    kwargs["primitives"] = True
    kwargs["obj_encoders"] = json_tricks.nonp.DEFAULT_ENCODERS
    return json_tricks.TricksEncoder(*args, **kwargs)


def to_string(obj, indent=0, flat=False):
    if not flat:
        try:
            out = json_tricks_dump(obj, sort_keys=True, indent=2)
        except:
            out = pprint.pformat(obj, width=120)
    else:
        try:
            out = json_tricks_dump(obj, sort_keys=True)
        except:
            out = str(obj).replace("\n", "")
    return indent_str(out, indent)

def truncate_obj(d):
    if not isinstance(d, dict):
        data = to_string(d, flat=True)
        return (data[:TRUNCATE_LIMIT] + "...") if len(data) > TRUNCATE_LIMIT else data

    data = {}
    for key in d:
        data[key] = truncate_obj(d[key])

    return data

def str_rep(item, truncate=20, round=2):
    if item is None:
        return ""
    if isinstance(item, float):
        out = "{0:.2f}".format(item)
    else:
        out = str(item)

    return truncate_str(out, truncate)


def truncate_str(item, truncate=20):
    if isinstance(item, str) and truncate is not None and truncate > 3 and len(item) > truncate:
        trim = truncate - 3
        return item[:trim] + "..."
    return item
