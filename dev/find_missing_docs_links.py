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

import sys
import os
import re
import asyncio
import aiohttp

script_path = os.path.realpath(__file__)
root = os.path.dirname(os.path.dirname(script_path))
docs_root = os.path.join(root, "docs")

skip_http = False
if len(sys.argv) == 2 and sys.argv[1] == "--skip-http":
    skip_http = True


def main():
    files = get_docs_file_paths()

    link_infos = []
    for file in files:
        link_infos += get_links_from_file(file)

    errors = check_links(link_infos)

    for error in errors:
        print(error)


def get_docs_file_paths():
    file_paths = []
    for root, dirs, files in os.walk(docs_root):
        for file in files:
            if file.endswith(".md"):
                file_paths.append(os.path.join(root, file))
    return file_paths


# link_info is (src_file, line_number, original_link_text, target_file, header)
def get_links_from_file(file):
    link_infos = []
    n = 1
    with open(file) as f:
        for line in f:
            for link in re.findall(r"\]\((.+?)\)", line):
                if is_external_link(link):
                    link_infos.append((file, n, link, None, None))
                    continue
                if link.startswith("#"):
                    link_infos.append((file, n, link, file, link[1:]))
                    continue
                if link.endswith(".md"):
                    target = os.path.normpath(os.path.join(file, "..", link))
                    link_infos.append((file, n, link, target, None))
                    continue
                if ".md#" in link:
                    parts = link.split("#")
                    if len(parts) == 2:
                        target = os.path.normpath(os.path.join(file, "..", parts[0]))
                        link_infos.append((file, n, link, target, parts[1]))
                    continue

                # Unexpected link format, will be handled later
                link_infos.append((file, n, link, None, None))

            n += 1

    return link_infos


def check_links(link_infos):
    errors = []
    http_link_infos = []

    for link_info in link_infos:
        src_file, line_num, original_link_text, target_file, header = link_info
        if is_external_link(original_link_text):
            http_link_infos.append(link_info)
            continue

        if not target_file and not header:
            errors.append(err_str(src_file, line_num, original_link_text, "unknown link format")),
            continue

        error = check_local_link(src_file, line_num, original_link_text, target_file, header)
        if error:
            errors.append(error)

    # fail fast if there are local link errors
    if len(errors) > 0:
        return errors

    if not skip_http:
        asyncio.get_event_loop().run_until_complete(check_all_http_links(http_link_infos, errors))

    return errors


async def check_all_http_links(http_link_infos, errors):
    links = set()

    async with aiohttp.ClientSession() as session:
        tasks = []
        for link_info in http_link_infos:
            src_file, line_num, link, _, _ = link_info
            if link in links:
                continue
            if "://localhost:" in link:
                continue
            links.add(link)
            tasks.append(
                asyncio.ensure_future(check_http_link(session, src_file, line_num, link, errors))
            )
        await asyncio.gather(*tasks)


async def check_http_link(session, src_file, line_num, link, errors):
    num_tries = 1
    while True:
        try:
            async with session.get(link, timeout=5) as resp:
                if resp.status != 200:
                    errors.append(
                        err_str(src_file, line_num, link, f"http response code {resp.status}")
                    )
                return
        except asyncio.TimeoutError:
            if num_tries > 2:
                errors.append(err_str(src_file, line_num, link, "http timeout"))
                return
            num_tries += 1


def check_local_link(src_file, line_num, original_link_text, target_file, header):
    if not os.path.isfile(target_file):
        return err_str(src_file, line_num, original_link_text, "file does not exist")

    if header:
        found_header = False
        with open(target_file) as f:
            for line in f:
                if not line.startswith("#"):
                    continue
                if header_matches(line, header):
                    found_header = True
                    break
        if not found_header:
            return err_str(src_file, line_num, original_link_text, "header does not exist")

    return None


def header_matches(text, header):
    text_words = re.findall(r"[a-zA-Z]+", text.lower())
    for word in re.findall(r"[a-zA-Z]+", header.lower()):
        if word not in text_words:
            return False
    return True


def err_str(src_file, line_num, original_link_text, reason):
    clean_src_file = src_file.split("cortexlabs/cortex/")[-1]
    return f"{clean_src_file}:{line_num}: {original_link_text} ({reason})"


def is_external_link(link):
    return link.startswith("http://") or link.startswith("https://")


if __name__ == "__main__":
    main()
