#!/bin/bash

# Copyright 2022 Cortex Labs, Inc.
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


import boto3
import configparser
from pathlib import Path
import sys
import os

# Usage: python create_user.py $CORTEX_CLUSTER_NAME $CORTEX_ACCOUNT_ID $CORTEX_REGION

cluster_name = sys.argv[1]
account_id = sys.argv[2]
cortex_region = sys.argv[3]

dir_path = os.path.dirname(os.path.realpath(__file__))

with open(f"{dir_path}/minimum_aws_policy.json", "r") as f:
    policy_string = f.read()
policy_string = policy_string.replace("$CORTEX_CLUSTER_NAME", cluster_name)
policy_string = policy_string.replace("$CORTEX_REGION", cortex_region)
policy_string = policy_string.replace("$CORTEX_ACCOUNT_ID", account_id)

user_name = f"dev-{cluster_name}-{cortex_region}"

iam_client = boto3.client("iam", region_name=cortex_region)

try:
    iam_client.get_user(UserName=user_name)
except iam_client.exceptions.NoSuchEntityException:
    iam_client.create_user(UserName=user_name)

partition = "aws"
if "us-gov" in cortex_region:
    partition = "aws-us-gov"

policy_arn = f"arn:{partition}:iam::{account_id}:policy/{user_name}"

try:
    iam_client.get_policy(PolicyArn=policy_arn)
except iam_client.exceptions.NoSuchEntityException:
    iam_client.create_policy(
        PolicyName=user_name,
        PolicyDocument=policy_string,
    )

policy_versions = iam_client.list_policy_versions(PolicyArn=policy_arn)["Versions"]

if len(policy_versions) == 5:
    policy_versions.sort(key=lambda x: x["CreateDate"])
    oldest_version = policy_versions[0]["VersionId"]
    iam_client.delete_policy_version(PolicyArn=policy_arn, VersionId=oldest_version)


iam_client.create_policy_version(
    PolicyArn=policy_arn, PolicyDocument=policy_string, SetAsDefault=True
)

iam_client.attach_user_policy(UserName=user_name, PolicyArn=policy_arn)

aws_credentials_path = Path("~/.aws/credentials").expanduser()
if not aws_credentials_path.exists():
    aws_credentials_path.parent.mkdir(parents=True, exist_ok=True)
    aws_credentials_path.touch()

aws_credentials = configparser.ConfigParser()
aws_credentials.read(aws_credentials_path)
if not user_name in aws_credentials:
    access_keys = iam_client.list_access_keys(UserName=user_name)["AccessKeyMetadata"]
    if len(access_keys) == 2:
        access_keys.sort(key=lambda x: x["CreateDate"])
        iam_client.delete_access_key(UserName=user_name, AccessKeyId=access_keys[0]["AccessKeyId"])
    key = iam_client.create_access_key(UserName=user_name)["AccessKey"]
    aws_credentials[user_name] = {
        "aws_access_key_id": key["AccessKeyId"],
        "aws_secret_access_key": key["SecretAccessKey"],
    }

    with open(aws_credentials_path, "w") as configfile:
        aws_credentials.write(configfile)

aws_access_key_id = aws_credentials[user_name]["aws_access_key_id"]
aws_secret_access_key = aws_credentials[user_name]["aws_secret_access_key"]

print(f"export AWS_ACCESS_KEY_ID={aws_access_key_id}")
print(f"export AWS_SECRET_ACCESS_KEY={aws_secret_access_key}")
