# SSH into AWS instance

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

There can be cases when SSH-ing into an AWS Cortex instance is asked for. The following 5 steps are identical for both approaches of SSH-ing in.

## Step 1

From the AWS EC2 dashboard, locate the instance you would like to connect to (it will start with something like `cortex-ng-cortex-worker`). Then in the "Security groups" section in the "Description", locate the group that is named something like `eksctl-cortex-cluster-ClusterSharedNodeSecurityGroup-***` and click on it.

![step 1](https://user-images.githubusercontent.com/26958764/80001314-e5ae0700-84c6-11ea-8f2e-349d4149a3a1.png)

## Step 2

On the Security Groups page, locate the same security group again and click on its ID.

![step 2](https://user-images.githubusercontent.com/26958764/80001399-fc545e00-84c6-11ea-8a8a-f6c566f67ba9.png)

## Step 3

Click "Edit inbound rules".

![step 3](https://user-images.githubusercontent.com/26958764/80001481-15f5a580-84c7-11ea-83f3-0257ae753af7.png)

## Step 4

Click "Add rule".

![step 4](https://user-images.githubusercontent.com/26958764/80001533-27d74880-84c7-11ea-9aa7-8d11be6c7598.png)

## Step 5

Select "SSH" for "Type" and "Anywhere" for "Source", and click "Save rules" (if you would like to have narrower access, this [Stack Overflow](https://stackoverflow.com/a/56918352/7143662) answer describes how).

![step 5](https://user-images.githubusercontent.com/26958764/80001609-3b82af00-84c7-11ea-911c-4d115d24aef7.png)

## Web Console

### Step 6 - Web Console

Back on the AWS EC2 dashboard, select the worker instance again and click "Connect".

![step 6](https://user-images.githubusercontent.com/26958764/80001744-666d0300-84c7-11ea-9783-9a1efd579404.png)

### Step 7 - Web Console

Select "EC2 Instance Connect (browser-based SSH connection)" and click "Connect".

![step 7](https://user-images.githubusercontent.com/26958764/80001831-813f7780-84c7-11ea-8200-52edc6efde94.png)

### Step 8 - Web Console

You should be SSH'd in!

![step 8](https://user-images.githubusercontent.com/26958764/80001894-9916fb80-84c7-11ea-8883-cc530293f17f.png)

*Note: some browsers may not be compatible with the AWS EC2 Instance Connect window and may throw a timeout. It is therefore recommended to switch to Google Chrome if it doesn't work.*

## Terminal


### Step 6 - Terminal

Take note of "Instance ID", "Availability Zone" and "Public DNS (IPv4)" fields of your worker instance.

![step 6](https://user-images.githubusercontent.com/26958764/80010486-2875dc00-84d3-11ea-8edf-afb3cdda6c17.png)

### Step 7 - Terminal

Generate a new RSA key pair. OpenSSH and SSH2 are supported alongside 2048 and 4096 bit lengths.
```bash
ssh-keygen -t rsa -f my_rsa_key
```

### Step 8 - Terminal

Provide the public key to the worker instance with `aws ec2-instance-connect send-ssh-public-key` command. The key is removed from the instance metadata within a 60 second timeframe. The public key can be reused any number of times. Then SSH in.

```bash
aws ec2-instance-connect send-ssh-public-key \
--instance-id <Instance ID> \
--availability-zone <Availability Zone> \
--instance-os-user root \
--ssh-public-key file://my_rsa_key.pub && \
ssh -i my_rsa_key <Public DNS (IPv4)>
```
