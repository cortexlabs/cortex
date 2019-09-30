# AWS credentials

As of now, Cortex only runs on AWS. We plan to support other cloud providers in the future. If you don't have an AWS account you can get started with one [here](https://portal.aws.amazon.com/billing/signup#/start).

Follow this [tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/create-access-key) to create an access key.

**Note:**

* Enable programmatic access for the IAM user.
* Attach the existing policy `AdministratorAccess` to your IAM user, or see [security](security.md) for a minimal access configuration.
* Each of the steps below requires different permissions. Please ensure you have the required permissions for each of the steps you need to run. The `AdministratorAccess` policy will work for all steps.
