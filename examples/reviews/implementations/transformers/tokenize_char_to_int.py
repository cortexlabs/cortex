import re


def transform_python(sample, args):
    text = sample["col"]
    reserved_indices = args["reserved_indices"]
    encoded = [c + len(reserved_indices) for c in text.encode("utf-8")]
    encoded = encoded[:500]

    for i in range(500 - len(encoded)):
        encoded.append(reserved_indices["<PAD>"])
    return encoded
