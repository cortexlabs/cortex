import re

non_word = re.compile("\\W")


def transform_python(input):
    lol = input
    text = lol["col"].lower()
    token_index_list = []
    vocab = lol["vocab"]

    for token in non_word.split(text):
        if len(token) == 0:
            continue
        token_index_list.append(vocab.get(token, vocab["<UNKNOWN>"]))
        if len(token_index_list) == lol["max_len"]:
            break

    for i in range(lol["max_len"] - len(token_index_list)):
        token_index_list.append(vocab["<PAD>"])

    return token_index_list
