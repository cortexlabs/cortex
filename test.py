import sys

# print(sys.argv)
file_name = sys.argv[1]


with open(file_name, "r") as f:
    text = f.readlines()

for first, _ in enumerate(text):
    text[first] = text[first].strip()

for first, line in enumerate(text):
    if "ErrorKind = iota" in text[first]:
        break

for last, line in enumerate(text):
    if last < first:
        continue
    if "Err" not in text[last + 1]:
        break


errors = []
for i in range(first, last + 1):
    errors.append(text[i].split(" ")[0])

for first, line in enumerate(text):
    if "var _errorKinds = []string{" in line:
        break
first += 1

for last, line in enumerate(text):
    if last < first:
        continue
    if "}" in text[last + 1]:
        break

for j in range(first, last + 1):
    real_idx = j - first
    errors[real_idx] = errors[real_idx] + " = " + text[j][:-1]


for e in errors:
    print(e)
