import boto3

ecr = boto3.client("ecr")

response = ecr.describe_repositories(maxResults=1000)

for repo in response["repositories"]:
    ecr.delete_repository(
        registryId=repo["registryId"], repositoryName=repo["repositoryName"], force=True,
    )
    print(f"deleted{repo['repositoryName']}")

print("done")
