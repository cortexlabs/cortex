# Single node deployment

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can use Cortex to deploy models on a single node. Deploying to a single node can be cheaper than spinning up a Cortex cluster with 1 worker node. It is useful to for testing on a GPU if you don't have access to one locally but you won't get the the advantages of deploying to a cluster such as autoscaling or rolling updates.

## AWS

Navigate to the [EC2 dashboard](https://console.aws.amazon.com/ec2/v2/home) in your AWS Web Console.

![ec2 dashboard](https://user-images.githubusercontent.com/4365343/81063901-ad370180-8ea6-11ea-8cd8-f552911043a9.png)

### Step 1

Choose a Linux based AMI instance. We recommend using "Ubuntu Server 18.04 LTS" (ami-085925f297f89fce1) for serving models. If you plan to serve models on a GPU we recommend using "Deep Learning AMI (Ubuntu 18.04) Version 28.1" (ami-078a7f1dda72c0775) because it a lot of the depenencies such as Docker Engine and NVIDIA libraries pre-installed and configured.

![step 2](https://user-images.githubusercontent.com/4365343/81064199-41a16400-8ea7-11ea-8d69-ae4ead6bf0be.png)

### Step 2

Choose your desired instance type with enough CPU and Memory to run your model. Typically it is a good idea match your CPU requirements and have atleast 1 GB of memory to spare for your operating system and any other processes that you might want to run on the node. To run most Cortex examples, an m5.large instance is sufficient.

Selecting an appropriate GPU instance depends on the kind of GPU card you want. Different GPU instance families have different GPU cards (i.e. g4* family instances use NVIDIA T4 while p2* family instances use NVIDIA K80). For typical GPU use cases, g4dn.xlarge which costs approximately 0.526 USD/hour (us-west-2 price) is one of the cheaper instances that should be able to serve models even large deep learning models such as GPT-2 .

Once you've chosen the instance click on "Next: Configure instance details".

![step 3](https://user-images.githubusercontent.com/4365343/81065727-07859180-8eaa-11ea-9293-af89906e4c6a.png)

### Step 3

To make things easy for testing and development, we should make sure that EC2 instance has a public ip address so that it is publically accessible. Then click on "Next: Add Storage".

![step 3](https://user-images.githubusercontent.com/4365343/81064806-5af6e000-8ea8-11ea-8e94-838fbea2710f.png)

### Step 4

We recommend having at least 30 GB of storage to save your models to disk and to download docker images needed to serve your model. Then click on "Next: Add Tags".

![step 4](https://user-images.githubusercontent.com/4365343/81078638-7cfa5d80-8ebc-11ea-820d-3baba690dbf8.png)

### Step 5

Adding tags is optional. You can add tags to your node to improve seachability and to have clear picture when reviewing you spend. Then click on "Next: Configure Security Group".

### Step 6

Configure your security group to allow inbound traffic to `local_port` number you specified in your cortex.yaml (default if not specified is 8888) from any source. Exposing this port allows you to make requests to your API but it also exposes it to the world so be careful not to do this in production. Then click on "Next: Review and Launch".

![step 6](https://user-images.githubusercontent.com/4365343/81065102-e2445380-8ea8-11ea-96e0-65676a0bafa8.png)

### Step 7

Double check details such as instance type, CPU, Memory and exposed ports and then click on "Launch".

![step 7](https://user-images.githubusercontent.com/4365343/81065800-26842380-8eaa-11ea-9c73-60ba0586ba38.png)

### Step 8

You will be prompted to select a key. This key is used to connect to your instance. Choose a key pair that you have access to. If you don't have one, you can create one and it will be downloaded to your browser's downloads folder. Then click on "Launch Instances".

![step 8](https://user-images.githubusercontent.com/4365343/81074878-9d73e900-8eb7-11ea-9c03-79ffea902dee.png)

### Step 9

Once your instance is spun up, follow the [relevant instructions](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstances.html) to connect to your instance.

If you are using a Mac or a Linux based OS then these instructions can help you ssh into your instance:
```
# make sure you have .ssh folder
$ mkdir -p ~/.ssh

# if your key was downloaded to your Downloads folder, move it to your .ssh folder
$ mv ~/Downloads/cortex-node.pem ~/.ssh/

# modify your key's permissions
$ chmod 400 ~/.ssh/cortex-node.pem

# get your instance's public DNS and then ssh into it
$ ssh -i "cortex-node.pem" ubuntu@ec2-3-235-100-162.compute-1.amazonaws.com
```

![step 9](https://user-images.githubusercontent.com/4365343/81078225-f180cc80-8ebb-11ea-81ae-5f5f0e76e623.png)

### Step 10

Docker Engine needs to be installed on your instance before you can use cortex. Skip to Step 11 if you are using "Deep Learning AMI (Ubuntu 18.04) Version 28.1" because Docker Engine is already installed and configured. Otherwise, follow these [instructions](https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository
) to install Docker Engine using the Docker repository.

Once Docker Engine is installed enable the use Docker commands without sudo:
```
$ sudo groupadd docker; sudo gpasswd -a $USER docker
```

_Note: you should reconnect to your instance to allow the permission changes to be effective._

If you have setup Docker correctly, you should be able to run docker commands such as `docker run hello-world` without running into permission issues and needing sudo.

### Step 11

Install Cortex CLI.

<!-- CORTEX_VERSION_MINOR -->
```
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"
```

### Step 12

Clone your repository and use Cortex to deploy your model.
```
$ git clone https://github.com/cortexlabs/cortex.git

$ cd cortex/examples/tensorflow/iris-classifier

$ cortex deploy

# take note of the curl command
$ cortex get iris-classifier
```

### Step 13

Make requests by replacing localhost in curl command with your instance's public DNS:
```
$ curl -X POST -H "Content-Type: application/json" \
    -d '{ "sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3 }' \
    <Instance public DNS>:<Port>
```
