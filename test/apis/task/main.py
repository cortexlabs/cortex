import json, re, os, boto3


def main():
    with open("/cortex/job_spec.json", "r") as f:
        job_spec = json.load(f)
    print(json.dumps(job_spec, indent=2))

    # get metadata
    config = job_spec["config"]
    job_id = job_spec["job_id"]
    s3_path = config["s3_path"]

    # touch file on s3
    bucket, key = re.match("s3://(.+?)/(.+)", s3_path).groups()
    s3 = boto3.client("s3")
    s3.put_object(Bucket=bucket, Key=os.path.join(key, job_id), Body="")


if __name__ == "__main__":
    main()
