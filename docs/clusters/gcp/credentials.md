# Credentials

1. Create a service account for your GCP project as described [here](https://cloud.google.com/iam/docs/creating-managing-service-accounts#iam-service-accounts-create-console) with the following roles (these roles could be more restrictive if required):
    1. `Editor` role.
    1. `Kubernetes Engine Admin` role.
    1. `Container Registry Service Agent` role.
    1. `Storage Admin` role.
    1. `Storage Object Admin` role.
1. Generate a service account key for your service account as described [here](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) and export it as a JSON file.
1. Export the `GOOGLE_APPLICATION_CREDENTIALS` variable and point it to the downloaded service account key from the previous step. For example: `export GOOGLE_APPLICATION_CREDENTIALS=/home/ubuntu/.config/gcloud/sample-269400-9a41792a969b.json`
