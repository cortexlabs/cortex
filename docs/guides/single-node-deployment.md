# Single node deployment

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can use Cortex to deploy models on a single node. Deploying to a single node can be cheaper than spinning up a Cortex cluster with 1 worker node. It also may be useful for testing your model on a GPU if you don't have access to one locally.

Deploying on a single node entails `ssh`ing into that instance and running Cortex locally. When using this approach, you won't get the the advantages of deploying to a cluster such as autoscaling, rolling updates, etc.

We've included the step-by-step instructions for AWS, although the process should be similar for any other cloud provider.

## AWS

### Step 1

Navigate to the [EC2 dashboard](https://console.aws.amazon.com/ec2/home) in your AWS Web Console and click "Launch instance".

![ec2 dashboard](https://user-images.githubusercontent.com/4365343/81063901-ad370180-8ea6-11ea-8cd8-f552911043a9.png)

### Step 2

Choose a Linux based AMI instance. We recommend using "Ubuntu Server 18.04 LTS". If you plan to serve models on a GPU, we recommend using "Deep Learning AMI (Ubuntu 18.04)" because it comes with Docker Engine and NVIDIA pre-installed.

![step 2](https://user-images.githubusercontent.com/4365343/81064199-41a16400-8ea7-11ea-8d69-ae4ead6bf0be.png)

### Step 3

Choose your desired instance type (it should have enough CPU and Memory to run your model). Typically it is a good idea have at least 1 GB of memory to spare for your operating system and any other processes that you might want to run on the instance. To run most Cortex examples, an m5.large instance is sufficient.

Selecting an appropriate GPU instance depends on the kind of GPU card you want. Different GPU instance families have different GPU cards (i.e. g4 family uses NVIDIA T4 while the p2 family uses NVIDIA K80). For typical GPU use cases, g4dn.xlarge is one of the cheaper instances that should be able to serve most large models, including deep learning models such as GPT-2.

Once you've chosen your instance click "Next: Configure instance details".

![step 3](https://user-images.githubusercontent.com/4365343/81065727-07859180-8eaa-11ea-9293-af89906e4c6a.png)

### Step 4

To make things easy for testing and development, we should make sure that the EC2 instance has a public IP address. Then click "Next: Add Storage".

![step 4](https://user-images.githubusercontent.com/4365343/81064806-5af6e000-8ea8-11ea-8e94-838fbea2710f.png)

### Step 5

We recommend having at least 50 GB of storage to save your models to disk and to download the docker images needed to serve your model. Then click "Next: Add Tags".

![step 5](https://user-images.githubusercontent.com/4365343/81078638-7cfa5d80-8ebc-11ea-820d-3baba690dbf8.png)

### Step 6

Adding tags is optional. You can add tags to your instance to improve searchability. Then click "Next: Configure Security Group".

### Step 7

Configure your security group to allow inbound traffic to the `local_port` number you specified in your `cortex.yaml` (the default is 8888 if not specified). Exposing this port allows you to make requests to your API but it also exposes it to the world so be careful. Then click "Next: Review and Launch".

![step 7](https://user-images.githubusercontent.com/4365343/81065102-e2445380-8ea8-11ea-96e0-65676a0bafa8.png)

### Step 8

Double check details such as instance type, CPU, Memory, and exposed ports. Then click "Launch".

![step 8](https://user-images.githubusercontent.com/4365343/81065800-26842380-8eaa-11ea-9c73-60ba0586ba38.png)

### Step 9

You will be prompted to select a key pair, which is used to connect to your instance. Choose a key pair that you have access to. If you don't have one, you can create one and it will be downloaded to your browser's downloads folder. Then click "Launch Instances".

![step 9](https://user-images.githubusercontent.com/4365343/81074878-9d73e900-8eb7-11ea-9c03-79ffea902dee.png)

### Step 10

Once your instance is running, follow the [relevant instructions](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstances.html) to connect to your instance.

If you are using a Mac or a Linux based OS, these instructions can help you ssh into your instance:

```bash
# make sure you have .ssh folder
$ mkdir -p ~/.ssh

# if your key was downloaded to your Downloads folder, move it to your .ssh folder
$ mv ~/Downloads/cortex-node.pem ~/.ssh/

# modify your key's permissions
$ chmod 400 ~/.ssh/cortex-node.pem

# get your instance's public DNS and then ssh into it
$ ssh -i "cortex-node.pem" ubuntu@ec2-3-235-100-162.compute-1.amazonaws.com
```

![step 10](https://user-images.githubusercontent.com/4365343/81078225-f180cc80-8ebb-11ea-81ae-5f5f0e76e623.png)

### Step 11

Docker Engine needs to be installed on your instance before you can use Cortex. Skip to Step 12 if you are using the "Deep Learning AMI" because Docker Engine is already installed. Otherwise, follow these [instructions](https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository) to install Docker Engine.

Once Docker Engine is installed, enable the Docker commands to be used without `sudo`:

```bash
$ sudo groupadd docker; sudo gpasswd -a $USER docker

# you must log out and back in for the permission changes to be effective
$ logout
```

If you have installed Docker correctly, you should be able to run docker commands such as `docker run hello-world` without running into permission issues or needing `sudo`.

### Step 12

Install the Cortex CLI.

<!-- CORTEX_VERSION_MINOR -->
```bash
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"
```

### Step 13

You can now use Cortex to deploy your model:

<!-- CORTEX_VERSION_MINOR -->
```bash
$ git clone -b master https://github.com/cortexlabs/cortex.git

$ cd cortex/examples/pytorch/text-generator

$ cortex deploy

# take note of the curl command
$ cortex get text-generator
```

### Step 14

Make requests by replacing "localhost" in the curl command with your instance's public DNS:

```bash
$ curl <instance public DNS>:<Port> \
    -X POST -H "Content-Type: application/json" \
    -d '{"text": "machine learning is"}'
```
