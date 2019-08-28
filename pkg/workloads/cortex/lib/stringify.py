
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

def truncate(item, truncate=75):
    trim = truncate - 3

    if isinstance(item, str) and truncate > 3 and len(item) > truncate:
        return item[:trim] + "..."

    if not isinstance(item, dict) :
        data = to_string(item, flat=True)
        return (data[:trim] + "...") if truncate > 3  and len(data) > truncate else data

    data = {}
    for key in item:
        data[key] = truncate(item[key])

    return data


