import re

non_word = re.compile("\\W")


def transform_python(sample, args):
    text = sample["col"].lower()
    token_index_list = []

    reverse_vocab = args["vocab"]
    stop_words = args["stop_words"]
    reserved_indices = args["reserved_indices"]

    for token in non_word.split(text):
        if len(token) == 0:
            continue
        if token in stop_words:
            continue
        token_index_list.append(reverse_vocab.get(token, reserved_indices["<UNKNOWN>"]))
        if len(token_index_list) == args["max_len"]:
            break

    for i in range(args["max_len"] - len(token_index_list)):
        token_index_list.append(reserved_indices["<PAD>"])

    return token_index_list
