# AWS credentials

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

As of now, Cortex only runs on AWS. We plan to support other cloud providers in the future. If you don't have an AWS account you can get started with one [here](https://portal.aws.amazon.com/billing/signup#/start).

Follow this [tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/create-access-key) to create an access key. Enable programmatic access for the IAM user, and attach the built-in `AdministratorAccess` policy to your IAM user (or see [security](security.md) for a minimal access configuration).
