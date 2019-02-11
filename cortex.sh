#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
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

set -e

############
### HELP ###
############

CORTEX_SH_TMP_DIR="$HOME/.cortex-sh-tmp"

function show_help() {
  echo "
Usage:
  ./cortex.sh command [sub-command] [flags]

Available Commands:
  install operator            install the operator (and the AWS CLI if necessary)
  install cli                 install the CLI
  install kubernetes-tools    install aws-iam-authenticator, eksctl, kubectl

  uninstall operator          uninstall the operator
  uninstall cli               uninstall the CLI
  uninstall kubernetes-tools  uninstall aws-iam-authenticator, eksctl, kubectl

  update operator             update the operator config and restart the operator

  endpoints                   show the operator and API endpoints

Flags:
  -c, --config  path to a cortex config file
  -h, --help
"
}

####################
### FLAG PARSING ###
####################

flag_help=false
positional_args=()

while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -c|--config)
    export CORTEX_CONFIG="$2"
    shift
    shift
    ;;
    -h|--help)
    flag_help="true"
    shift
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done

set -- "${positional_args[@]}"
positional_args=()
for i in "$@"; do
  case $i in
    -c=*|--config=*)
    export CORTEX_CONFIG="${i#*=}"
    shift
    ;;
    -h=*|--help=*)
    flag_help="true"
    ;;
    *)
    positional_args+=("$1")
    shift
    ;;
  esac
done

set -- "${positional_args[@]}"
if [ "$flag_help" == "true" ]; then
  show_help
  exit 0
fi

for arg in "$@"; do
  if [[ "$arg" == -* ]]; then
    echo "unknown flag: $arg"
    show_help
    exit 1
  fi
done

set -u

################
### CHECK OS ###
################

case "$OSTYPE" in
  darwin*)  PARSED_OS="darwin" ;;
  linux*)   PARSED_OS="linux" ;;
  *)        echo "\nerror: only mac and linux are supported"; exit 1 ;;
esac

#####################
### CONFIGURATION ###
#####################

export CORTEX_CONFIG="${CORTEX_CONFIG:-config.sh}"
if [ -f "$CORTEX_CONFIG" ]; then
  source $CORTEX_CONFIG
fi

export CORTEX_VERSION_STABLE=master

# Defaults
random_id=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-z0-9' | fold -w 12 | head -n 1)

export CORTEX_LOG_GROUP="${CORTEX_LOG_GROUP:-cortex}"
export CORTEX_BUCKET="${CORTEX_BUCKET:-cortex-$random_id}"
export CORTEX_REGION="${CORTEX_REGION:-us-west-2}"
export CORTEX_NAMESPACE="${CORTEX_NAMESPACE:-cortex}"

export CORTEX_IMAGE_ARGO_CONTROLLER="${CORTEX_IMAGE_ARGO_CONTROLLER:-cortexlabs/argo-controller:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_ARGO_EXECUTOR="${CORTEX_IMAGE_ARGO_EXECUTOR:-cortexlabs/argo-executor:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_FLUENTD="${CORTEX_IMAGE_FLUENTD:-cortexlabs/fluentd:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_NGINX_BACKEND="${CORTEX_IMAGE_NGINX_BACKEND:-cortexlabs/nginx-backend:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_NGINX_CONTROLLER="${CORTEX_IMAGE_NGINX_CONTROLLER:-cortexlabs/nginx-controller:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_OPERATOR="${CORTEX_IMAGE_OPERATOR:-cortexlabs/operator:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_SPARK="${CORTEX_IMAGE_SPARK:-cortexlabs/spark:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_SPARK_OPERATOR="${CORTEX_IMAGE_SPARK_OPERATOR:-cortexlabs/spark-operator:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_SERVE="${CORTEX_IMAGE_TF_SERVE:-cortexlabs/tf-serve:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_TRAIN="${CORTEX_IMAGE_TF_TRAIN:-cortexlabs/tf-train:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_API="${CORTEX_IMAGE_TF_API:-cortexlabs/tf-api:$CORTEX_VERSION_STABLE}"

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-""}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"

##########################
### TOP-LEVEL COMMANDS ###
##########################

function install_operator() {
  check_dep_curl
  check_dep_aws
  check_dep_kubectl

  setup_bucket

  echo "Installing the Cortex operator ..."

  setup_namespace
  setup_configmap
  setup_secrets
  setup_spark
  setup_argo
  setup_nginx
  setup_logging
  setup_operator

  validate_cortex
}

function install_cli() {
  install_cortex_cli
}

function install_kubernetes_tools() {
  install_aws_iam_authenticator
  install_eksctl
  install_kubectl

  echo
  echo "You can now spin up an EKS cluster using the command below (see eksctl.io for more configuration options):"
  echo "  eksctl create cluster --name=cortex --nodes=2 --node-type=t3.medium  # this takes ~20 minutes"
  echo
  echo "Note: we recommend a minimum cluster size of 2 t3.medium AWS instances. Cortex may not run successfully on clusters with less compute resources."
}

function uninstall_operator() {
  check_dep_kubectl

  echo
  if kubectl get namespace cortex >/dev/null 2>&1 || kubectl get customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1 || kubectl get customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io >/dev/null 2>&1 || kubectl get customresourcedefinition workflows.argoproj.io >/dev/null 2>&1; then
    echo "Uninstalling the Cortex operator from your Kubernetes cluster ..."
    kubectl delete --ignore-not-found=true customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
    kubectl delete --ignore-not-found=true customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
    kubectl delete --ignore-not-found=true customresourcedefinition workflows.argoproj.io >/dev/null 2>&1
    kubectl delete --ignore-not-found=true namespace $CORTEX_NAMESPACE >/dev/null 2>&1
    echo "✓ Uninstalled the Cortex operator"
  else
    echo "The Cortex operator is not installed on your Kubernetes cluster"
  fi

  echo
  echo "Command to spin down your Kubernetes cluster:"
  echo "  eksctl delete cluster --name=cortex"
  echo
  echo "Command to remove Kubernetes tools:"
  echo "  ./cortex.sh uninstall kubernetes-tools"
  echo
  echo "Command to delete the bucket used by Cortex:"
  echo "  aws s3 rb s3://<bucket-name> --force"
  echo
  echo "Command to delete the log group used by Cortex:"
  echo "  aws logs delete-log-group --log-group-name cortex --region us-west-2"
  echo
  echo "Command to uninstall the Cortex CLI:"
  echo "  ./cortex.sh uninstall cli"

  if [[ -f /usr/local/bin/aws ]]; then
    echo -e "\nCommand to uninstall the AWS CLI:"
    if [ $(id -u) = 0 ]; then
      echo "  rm -rf /usr/local/aws && rm /usr/local/bin/aws && rm -rf ~/.aws"
    else
      echo "  sudo rm -rf /usr/local/aws && sudo rm /usr/local/bin/aws && rm -rf ~/.aws"
    fi
  fi
}

function uninstall_cli() {
  uninstall_cortex_cli
}

function uninstall_kubernetes_tools() {
  uninstall_kubectl
  uninstall_eksctl
  uninstall_aws_iam_authenticator
}

function update_operator() {
  check_dep_curl
  check_dep_kubectl

  echo -e "\nUpdating the Cortex operator ..."
  delete_operator
  setup_configmap
  setup_operator
  validate_cortex
}

function get_endpoints() {
  check_dep_kubectl

  operator_endpoint=$(get_operator_endpoint)
  apis_endpoint=$(get_apis_endpoint)
  echo
  echo "operator endpoint:  $operator_endpoint"
  echo "APIs endpoint:      $apis_endpoint"
}

#################
### AWS SETUP ###
#################

function setup_bucket() {
  echo
  if ! aws s3api head-bucket --bucket $CORTEX_BUCKET --output json 2>/dev/null; then
    if aws s3 ls "s3://$CORTEX_BUCKET" --output json 2>&1 | grep -q 'NoSuchBucket'; then
      echo -e "Creating S3 bucket: $CORTEX_BUCKET\n"
      aws s3api create-bucket --bucket $CORTEX_BUCKET \
                              --region $CORTEX_REGION \
                              --create-bucket-configuration LocationConstraint=$CORTEX_REGION \
                              >/dev/null
    else
      echo "A bucket named \"${CORTEX_BUCKET}\" already exists, but you do not have access to it"
      exit 1
    fi
  fi
}

#######################
### NAMESPACE SETUP ###
#######################

function setup_namespace() {
  echo "
apiVersion: v1
kind: Namespace
metadata:
  name: ${CORTEX_NAMESPACE}
" | kubectl apply -f - >/dev/null
}

#######################
### CONFIGMAP SETUP ###
#######################

function setup_configmap() {
  kubectl -n=$CORTEX_NAMESPACE create configmap 'cortex-config' \
    --from-literal='LOG_GROUP'=$CORTEX_LOG_GROUP \
    --from-literal='BUCKET'=$CORTEX_BUCKET \
    --from-literal='REGION'=$CORTEX_REGION \
    --from-literal='NAMESPACE'=$CORTEX_NAMESPACE \
    --from-literal='IMAGE_OPERATOR'=$CORTEX_IMAGE_OPERATOR \
    --from-literal='IMAGE_SPARK'=$CORTEX_IMAGE_SPARK \
    --from-literal='IMAGE_TF_TRAIN'=$CORTEX_IMAGE_TF_TRAIN \
    --from-literal='IMAGE_TF_SERVE'=$CORTEX_IMAGE_TF_SERVE \
    --from-literal='IMAGE_TF_API'=$CORTEX_IMAGE_TF_API \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

#######################
### SECRETS SETUP ###
#######################

function setup_secrets() {
  kubectl -n=$CORTEX_NAMESPACE create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

##################
### ARGO SETUP ###
##################

function setup_argo() {
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-executor
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: argo-executor
  namespace: ${CORTEX_NAMESPACE}
subjects:
- kind: ServiceAccount
  name: argo-executor
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: workflows.argoproj.io
  namespace: ${CORTEX_NAMESPACE}
spec:
  group: argoproj.io
  names:
    kind: Workflow
    plural: workflows
    shortNames:
    - wf
  scope: Namespaced
  version: v1alpha1
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-controller
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: argo-controller
  namespace: ${CORTEX_NAMESPACE}
rules:
- apiGroups: [\"\"]
  resources: [pods, pods/exec]
  verbs: [create, get, list, watch, update, patch, delete]
- apiGroups: [\"\"]
  resources: [configmaps]
  verbs: [get, list, watch]
- apiGroups: [\"\"]
  resources: [persistentvolumeclaims]
  verbs: [create, delete]
- apiGroups: [argoproj.io]
  resources: [workflows, workflows/finalizers]
  verbs: [get, list, watch, update, patch]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: argo
  namespace: ${CORTEX_NAMESPACE}
subjects:
- kind: ServiceAccount
  name: argo-controller
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: Role
  name: argo-controller
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-controller
  namespace: ${CORTEX_NAMESPACE}
data:
  config: |
    namespace: ${CORTEX_NAMESPACE}
---
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: argo-controller
  namespace: ${CORTEX_NAMESPACE}
spec:
  selector:
    matchLabels:
      app: argo-controller
  template:
    metadata:
      labels:
        app: argo-controller
    spec:
      containers:
      - args:
        - --configmap
        - argo-controller
        - --executor-image
        - ${CORTEX_IMAGE_ARGO_EXECUTOR}
        - --executor-image-pull-policy
        - Always
        command:
        - workflow-controller
        name: argo-controller
        image: ${CORTEX_IMAGE_ARGO_CONTROLLER}
        imagePullPolicy: Always
      serviceAccountName: argo-controller
" | kubectl apply -f - >/dev/null
}

###################
### SPARK SETUP ###
###################

function setup_spark() {
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: spark-operator
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: spark-operator
  namespace: ${CORTEX_NAMESPACE}
rules:
- apiGroups: [\"\"]
  resources: [pods]
  verbs: [\"*\"]
- apiGroups: [\"\"]
  resources: [services, configmaps, secrets]
  verbs: [create, get, delete]
- apiGroups: [extensions]
  resources: [ingresses]
  verbs: [create, get, delete]
- apiGroups: [\"\"]
  resources: [nodes]
  verbs: [get]
- apiGroups: [\"\"]
  resources: [events]
  verbs: [create, update, patch]
- apiGroups: [apiextensions.k8s.io]
  resources: [customresourcedefinitions]
  verbs: [create, get, update, delete]
- apiGroups: [admissionregistration.k8s.io]
  resources: [mutatingwebhookconfigurations]
  verbs: [create, get, update, delete]
- apiGroups: [sparkoperator.k8s.io]
  resources: [sparkapplications, scheduledsparkapplications]
  verbs: [\"*\"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: spark-operator
  namespace: ${CORTEX_NAMESPACE}
subjects:
  - kind: ServiceAccount
    name: spark-operator
    namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: Role
  name: spark-operator
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: spark-operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: spark-operator
    app.kubernetes.io/version: v2.4.0-v1alpha1
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: spark-operator
      app.kubernetes.io/version: v2.4.0-v1alpha1
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: spark-operator
        app.kubernetes.io/version: v2.4.0-v1alpha1
      initializers:
        pending: []
    spec:
      serviceAccountName: spark-operator
      containers:
      - name: spark-operator
        image: ${CORTEX_IMAGE_SPARK_OPERATOR}
        imagePullPolicy: Always
        command: [\"/usr/bin/spark-operator\"]
        args:
          - -namespace=${CORTEX_NAMESPACE}
          - -install-crds=false
          - -logtostderr
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: sparkapplications.sparkoperator.k8s.io
spec:
  group: sparkoperator.k8s.io
  names:
    kind: SparkApplication
    listKind: SparkApplicationList
    plural: sparkapplications
    shortNames:
    - sparkapp
    singular: sparkapplication
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            deps:
              properties:
                downloadTimeout:
                  minimum: 1
                  type: integer
                maxSimultaneousDownloads:
                  minimum: 1
                  type: integer
            driver:
              properties:
                cores:
                  exclusiveMinimum: true
                  minimum: 0
                  type: number
                podName:
                  pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
            executor:
              properties:
                cores:
                  exclusiveMinimum: true
                  minimum: 0
                  type: number
                instances:
                  minimum: 1
                  type: integer
            mode:
              enum:
              - cluster
              - client
            monitoring:
              properties:
                prometheus:
                  properties:
                    port:
                      maximum: 49151
                      minimum: 1024
                      type: integer
            pythonVersion:
              enum:
              - \"2\"
              - \"3\"
            restartPolicy:
              properties:
                onFailureRetries:
                  minimum: 0
                  type: integer
                onFailureRetryInterval:
                  minimum: 1
                  type: integer
                onSubmissionFailureRetries:
                  minimum: 0
                  type: integer
                onSubmissionFailureRetryInterval:
                  minimum: 1
                  type: integer
                type:
                  enum:
                  - Never
                  - OnFailure
                  - Always
            type:
              enum:
              - Java
              - Scala
              - Python
              - R
  version: v1alpha1
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: scheduledsparkapplications.sparkoperator.k8s.io
spec:
  group: sparkoperator.k8s.io
  names:
    kind: ScheduledSparkApplication
    listKind: ScheduledSparkApplicationList
    plural: scheduledsparkapplications
    shortNames:
    - scheduledsparkapp
    singular: scheduledsparkapplication
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            concurrencyPolicy:
              enum:
              - Allow
              - Forbid
              - Replace
            failedRunHistoryLimit:
              minimum: 1
              type: integer
            schedule:
              type: string
            successfulRunHistoryLimit:
              minimum: 1
              type: integer
            template:
              properties:
                deps:
                  properties:
                    downloadTimeout:
                      minimum: 1
                      type: integer
                    maxSimultaneousDownloads:
                      minimum: 1
                      type: integer
                driver:
                  properties:
                    cores:
                      exclusiveMinimum: true
                      minimum: 0
                      type: number
                    podName:
                      pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
                executor:
                  properties:
                    cores:
                      exclusiveMinimum: true
                      minimum: 0
                      type: number
                    instances:
                      minimum: 1
                      type: integer
                mode:
                  enum:
                  - cluster
                  - client
                monitoring:
                  properties:
                    prometheus:
                      properties:
                        port:
                          maximum: 49151
                          minimum: 1024
                          type: integer
                pythonVersion:
                  enum:
                  - \"2\"
                  - \"3\"
                restartPolicy:
                  properties:
                    onFailureRetries:
                      minimum: 0
                      type: integer
                    onFailureRetryInterval:
                      minimum: 1
                      type: integer
                    onSubmissionFailureRetries:
                      minimum: 0
                      type: integer
                    onSubmissionFailureRetryInterval:
                      minimum: 1
                      type: integer
                    type:
                      enum:
                      - Never
                      - OnFailure
                      - Always
                type:
                  enum:
                  - Java
                  - Scala
                  - Python
                  - R
  version: v1alpha1
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: spark
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: spark
  namespace: ${CORTEX_NAMESPACE}
rules:
- apiGroups:
  - \"\" # \"\" indicates the core API group
  resources: [pods]
  verbs: [\"*\"]
- apiGroups:
  - \"\" # \"\" indicates the core API group
  resources: [services]
  verbs: [\"*\"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: spark
  namespace: ${CORTEX_NAMESPACE}
subjects:
- kind: ServiceAccount
  name: spark
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: Role
  name: spark
  apiGroup: rbac.authorization.k8s.io
" | kubectl apply -f - >/dev/null
}

###################
### NGINX SETUP ###
###################

function setup_nginx() {
  echo "
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nginx
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: nginx
  namespace: ${CORTEX_NAMESPACE}
rules:
  - apiGroups: [\"\"]
    resources: [endpoints, pods, secrets]
    verbs: [list, watch]
  - apiGroups: [\"\"]
    resources: [nodes, services, ingresses]
    verbs: [get, list, watch]
  - apiGroups: [\"\"]
    resources: [events]
    verbs: [create, patch]
  - apiGroups: [\"extensions\"]
    resources: [ingresses]
    verbs: [get, list, watch]
  - apiGroups: [\"extensions\"]
    resources: [ingresses/status]
    verbs: [update]
  - apiGroups: [\"\"]
    resources: [pods, secrets, namespaces, endpoints]
    verbs: [get]
  - apiGroups: [\"\"]
    resources: [configmaps]
    resourceNames:
      # Defaults to \"<election-id>-<ingress-class>\"
      # Here: \"<ingress-controller-leader>-<nginx>\"
      # This has to be adapted if you change either parameter
      # when launching the nginx-ingress-controller.
      - \"ingress-controller-leader-operator\"
      - \"ingress-controller-leader-apis\"
    verbs: [get, update]
  - apiGroups: [\"\"]
    resources: [configmaps]
    verbs: [get, list, watch, create]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: nginx
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: nginx
subjects:
  - kind: ServiceAccount
    name: nginx
    namespace: ${CORTEX_NAMESPACE}
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-configuration
  namespace: ${CORTEX_NAMESPACE}
data:
  use-proxy-protocol: \"true\"
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx-backend-operator
  labels:
    app.kubernetes.io/name: nginx-backend-operator
    app.kubernetes.io/part-of: ingress-nginx
  namespace: ${CORTEX_NAMESPACE}
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-backend-operator
      app.kubernetes.io/part-of: ingress-nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-backend-operator
        app.kubernetes.io/part-of: ingress-nginx
    spec:
      terminationGracePeriodSeconds: 60
      containers:
      - name: nginx-backend-operator
        # Any image is permissible as long as:
        # 1. It serves a 404 page at /
        # 2. It serves 200 on a /healthz endpoint
        image: ${CORTEX_IMAGE_NGINX_BACKEND}
        imagePullPolicy: Always
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 30
          timeoutSeconds: 5
        ports:
        - containerPort: 8080
        resources:
          limits:
            cpu: 10m
            memory: 20Mi
          requests:
            cpu: 10m
            memory: 20Mi
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-backend-operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-backend-operator
    app.kubernetes.io/part-of: ingress-nginx
spec:
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app.kubernetes.io/name: nginx-backend-operator
    app.kubernetes.io/part-of: ingress-nginx
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx-controller-operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-controller-operator
    app.kubernetes.io/part-of: ingress-nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-controller-operator
      app.kubernetes.io/part-of: ingress-nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-controller-operator
        app.kubernetes.io/part-of: ingress-nginx
    spec:
      serviceAccountName: nginx
      containers:
        - name: nginx-controller
          image: ${CORTEX_IMAGE_NGINX_CONTROLLER}
          imagePullPolicy: Always
          args:
            - /nginx-ingress-controller
            - --watch-namespace=${CORTEX_NAMESPACE}
            - --default-backend-service=${CORTEX_NAMESPACE}/nginx-backend-operator
            - --configmap=${CORTEX_NAMESPACE}/nginx-configuration
            - --publish-service=${CORTEX_NAMESPACE}/nginx-controller-operator
            - --annotations-prefix=nginx.ingress.kubernetes.io
            - --ingress-class=operator
          securityContext:
            capabilities:
                drop:
                - ALL
                add:
                - NET_BIND_SERVICE
            # www-data -> 33
            runAsUser: 33
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          ports:
          - name: http
            containerPort: 80
          - name: https
            containerPort: 443
          livenessProbe:
            failureThreshold: 3
            httpGet:
              path: /healthz
              port: 10254
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 1
          readinessProbe:
            failureThreshold: 3
            httpGet:
              path: /healthz
              port: 10254
              scheme: HTTP
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 1
---
kind: Service
apiVersion: v1
metadata:
  name: nginx-controller-operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-controller-operator
    app.kubernetes.io/part-of: ingress-nginx
  annotations:
    # Enable PROXY protocol
    service.beta.kubernetes.io/aws-load-balancer-proxy-protocol: '*'
    # Ensure the ELB idle timeout is less than nginx keep-alive timeout. By default,
    # NGINX keep-alive is set to 75s. If using WebSockets, the value will need to be
    # increased to '3600' to avoid any potential issues.
    service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout: '60'
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: nginx-controller-operator
    app.kubernetes.io/part-of: ingress-nginx
  ports:
  - name: http
    port: 80
    targetPort: http
  - name: https
    port: 443
    targetPort: https
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx-backend-apis
  labels:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
  namespace: ${CORTEX_NAMESPACE}
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-backend-apis
      app.kubernetes.io/part-of: ingress-nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-backend-apis
        app.kubernetes.io/part-of: ingress-nginx
    spec:
      terminationGracePeriodSeconds: 60
      containers:
      - name: nginx-backend-apis
        # Any image is permissible as long as:
        # 1. It serves a 404 page at /
        # 2. It serves 200 on a /healthz endpoint
        image: ${CORTEX_IMAGE_NGINX_BACKEND}
        imagePullPolicy: Always
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 30
          timeoutSeconds: 5
        ports:
        - containerPort: 8080
        resources:
          limits:
            cpu: 10m
            memory: 20Mi
          requests:
            cpu: 10m
            memory: 20Mi
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-backend-apis
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
spec:
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: nginx-controller-apis
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-backend-apis
      app.kubernetes.io/part-of: ingress-nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-backend-apis
        app.kubernetes.io/part-of: ingress-nginx
    spec:
      serviceAccountName: nginx
      containers:
        - name: nginx-controller
          image: ${CORTEX_IMAGE_NGINX_CONTROLLER}
          imagePullPolicy: Always
          args:
            - /nginx-ingress-controller
            - --watch-namespace=${CORTEX_NAMESPACE}
            - --default-backend-service=${CORTEX_NAMESPACE}/nginx-backend-apis
            - --configmap=${CORTEX_NAMESPACE}/nginx-configuration
            - --publish-service=${CORTEX_NAMESPACE}/nginx-backend-apis
            - --annotations-prefix=nginx.ingress.kubernetes.io
            - --ingress-class=apis
          securityContext:
            capabilities:
                drop:
                - ALL
                add:
                - NET_BIND_SERVICE
            # www-data -> 33
            runAsUser: 33
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          ports:
          - name: http
            containerPort: 80
          - name: https
            containerPort: 443
          livenessProbe:
            failureThreshold: 3
            httpGet:
              path: /healthz
              port: 10254
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 1
          readinessProbe:
            failureThreshold: 3
            httpGet:
              path: /healthz
              port: 10254
              scheme: HTTP
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 1
---
kind: Service
apiVersion: v1
metadata:
  name: nginx-controller-apis
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
  annotations:
    # Enable PROXY protocol
    service.beta.kubernetes.io/aws-load-balancer-proxy-protocol: '*'
    # Ensure the ELB idle timeout is less than nginx keep-alive timeout. By default,
    # NGINX keep-alive is set to 75s. If using WebSockets, the value will need to be
    # increased to '3600' to avoid any potential issues.
    service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout: '60'
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: nginx-backend-apis
    app.kubernetes.io/part-of: ingress-nginx
  ports:
  - name: http
    port: 80
    targetPort: http
  - name: https
    port: 443
    targetPort: https
" | kubectl apply -f - >/dev/null
}

#####################
### FLUENTD SETUP ###
#####################

function setup_logging() {
  if ! aws logs list-tags-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION --output json 2>&1 | grep -q "\"tags\":"; then
    aws logs create-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION
  fi

  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app: fluentd
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: Role
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
rules:
- apiGroups: [\"\"]
  resources: [pods]
  verbs: [get, list, watch]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
subjects:
- kind: ServiceAccount
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: Role
  name: fluentd
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
data:
  fluent.conf: |
    <match fluent.**>
      @type null
    </match>
    <source>
      @type tail
      enable_stat_watcher false
      path /var/log/containers/**_${CORTEX_NAMESPACE}_**.log
      pos_file /var/log/fluentd-containers.log.pos
      time_format %Y-%m-%dT%H:%M:%S.%NZ
      tag *
      format json
      read_from_head true
    </source>
    <match **>
      @type cloudwatch_logs
      log_group_name \"#{ENV['LOG_GROUP_NAME']}\"
      auto_create_stream true
      use_tag_as_stream true
    </match>
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
spec:
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      serviceAccountName: fluentd
      initContainers:
        - name: copy-fluentd-config
          image: busybox
          command: ['sh', '-c', 'cp /config-volume/* /etc/fluentd']
          volumeMounts:
            - name: config-volume
              mountPath: /config-volume
            - name: config
              mountPath: /etc/fluentd
      containers:
      - name: fluentd
        image: ${CORTEX_IMAGE_FLUENTD}
        imagePullPolicy: Always
        env:
        - name: AWS_REGION
          value: ${CORTEX_REGION}
        - name: LOG_GROUP_NAME
          value: ${CORTEX_LOG_GROUP}
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: AWS_ACCESS_KEY_ID
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: AWS_SECRET_ACCESS_KEY
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: config
          mountPath: /fluentd/etc
      terminationGracePeriodSeconds: 30
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
      - name: config
        emptyDir: {}
      - name: config-volume
        configMap:
          name: fluentd
" | kubectl apply -f - >/dev/null
}

######################
### OPERATOR SETUP ###
######################

function setup_operator() {
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
subjects:
- kind: ServiceAccount
  name: operator
  namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    workloadType: operator
spec:
  replicas: 1
  template:
    metadata:
      labels:
        workloadId: operator
        workloadType: operator
    spec:
      containers:
      - name: operator
        image: ${CORTEX_IMAGE_OPERATOR}
        imagePullPolicy: Always
        env:
          - name: AWS_ACCESS_KEY_ID
            valueFrom:
              secretKeyRef:
                name: aws-credentials
                key: AWS_ACCESS_KEY_ID
          - name: AWS_SECRET_ACCESS_KEY
            valueFrom:
              secretKeyRef:
                name: aws-credentials
                key: AWS_SECRET_ACCESS_KEY
        volumeMounts:
          - name: cortex-config
            mountPath: /configs/cortex
      volumes:
        - name: cortex-config
          configMap:
            name: cortex-config
      serviceAccountName: operator
---
kind: Service
apiVersion: v1
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    workloadType: operator
spec:
  selector:
    workloadId: operator
  ports:
  - port: 8888
    targetPort: 8888
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    workloadType: operator
  annotations:
    kubernetes.io/ingress.class: operator
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: operator
          servicePort: 8888
" | kubectl apply -f - >/dev/null
}

function delete_operator() {
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true ingress operator >/dev/null 2>&1
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true service operator >/dev/null 2>&1
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true deployment operator >/dev/null 2>&1
}

function validate_cortex() {
  set -e

  echo -en "\nWaiting for the Cortex operator to be ready "

  operator_load_balancer="waiting"
  api_load_balancer="waiting"
  operator_endpoint=""

  while true; do
    echo -n "."
    sleep 5

    if [ "$operator_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      operator_load_balancer="ready"
    fi

    if [ "$api_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      api_load_balancer="ready"
    fi

    if [ "$operator_endpoint" = "" ]; then
      operator_endpoint=$(kubectl -n=cortex get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
    fi
    if ! curl $operator_endpoint >/dev/null 2>&1; then
      continue
    fi

    echo " ✓"
    break
  done

  echo -e "\nCortex is ready!"

  get_endpoints

  if command -v cortex >/dev/null; then
    echo -e "\nPlease run 'cortex configure' to make sure your CLI is configured correctly"
  else
    echo -e "\nCommand to install the Cortex CLI:"
    echo "  ./cortex.sh install cli"
  fi
}

function get_operator_endpoint() {
  set -eo pipefail
  kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_operator_endpoint_or_empty() {
  kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json 2>/dev/null | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_apis_endpoint() {
  set -eo pipefail
  kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

#############################
### DEPENDENCY MANAGEMENT ###
#############################

function check_dep_curl() {
  if ! command -v curl >/dev/null; then
    echo -e "\nerror: please install \`curl\`"
    exit 1
  fi
}

function check_dep_unzip() {
  if ! command -v unzip >/dev/null; then
    echo -e "\nerror: please install \`unzip\`"
    exit 1
  fi
}

function check_dep_kubectl() {
  set -e

  if ! command -v kubectl >/dev/null 2>&1; then
    echo
    read -p "kubectl is required. Would you like cortex.sh to install it? [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      install_kubectl
    else
      exit 1
    fi
  fi

  if ! kubectl config current-context >/dev/null 2>&1; then
    echo "\nerror: kubectl is not configured to connect with your cluster. If you are using eksctl, you can run this command to configure kubectl:"
    echo "  eksctl utils write-kubeconfig --name=cortex"
    exit 1
  fi

  jsonpath='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'
  set +e
  get_nodes_output=$(kubectl get nodes -o jsonpath="$jsonpath" 2>/dev/null)
  if [ $? -ne 0 ]; then
    echo "\nerror: kubectl is not properly configured to connect with your cluster. If you are using eksctl, you can run this command to configure kubectl:"
    echo "  eksctl utils write-kubeconfig --name=cortex"
    exit 1
  fi
  set -e
  num_nodes_ready=$(echo $get_nodes_output | tr ';' "\n" | grep "Ready=True" | wc -l)
  if ! [[ $num_nodes_ready -ge 1 ]]; then
    echo "\nerror: your cluster has no registered nodes"
    exit 1
  fi
}

function install_kubectl() {
  set -e

  if command -v kubectl >/dev/null; then
    echo -e "\nkubectl is already installed"
    return
  fi

  check_dep_curl

  echo -e "\nInstalling kubectl (/usr/local/bin/kubectl) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -Lo $CORTEX_SH_TMP_DIR/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.13.3/bin/$PARSED_OS/amd64/kubectl
  chmod +x $CORTEX_SH_TMP_DIR/kubectl

  if [ $(id -u) = 0 ]; then
    mv $CORTEX_SH_TMP_DIR/kubectl /usr/local/bin/kubectl
  else
    ask_sudo
    sudo mv $CORTEX_SH_TMP_DIR/kubectl /usr/local/bin/kubectl
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed kubectl"
}

function uninstall_kubectl() {
  set -e

  if ! command -v kubectl >/dev/null; then
    echo -e "\nkubectl is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/kubectl ]]; then
    echo -e "\nkubectl was not found at /usr/local/bin/kubectl, please uninstall it manually"
    return
  fi

  echo
  read -p "Would you like to uninstall kubectl (/usr/local/bin/kubectl)? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ $(id -u) = 0 ]; then
      rm /usr/local/bin/kubectl
    else
      ask_sudo
      sudo rm /usr/local/bin/kubectl
    fi
    rm -rf $HOME/.kube
    echo "✓ Uninstalled kubectl"
  else
    return
  fi
}

function check_dep_aws() {
  set -e

  if ! command -v aws >/dev/null 2>&1; then
    echo
    read -p "The AWS CLI is required. Would you like cortex.sh to install it? [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      install_aws
    else
      exit 1
    fi
  fi

  if [ -z "$AWS_ACCESS_KEY_ID" ]; then
    echo "\nerror: please export AWS_ACCESS_KEY_ID"
    exit 1
  fi

  if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "\nerror: please export AWS_SECRET_ACCESS_KEY"
    exit 1
  fi
}

function install_aws() {
  set -e

  if command -v aws >/dev/null; then
    echo "The AWS CLI is already installed"
    return
  fi

  check_dep_curl
  check_dep_unzip

  if command -v python >/dev/null; then
    py_path=$(which python)
  elif command -v python3 >/dev/null; then
    py_path=$(which python3)
  else
    echo "\nerror: please install python or python3 using your package manager"
    exit 1
  fi

  if ! $py_path -c "import distutils.sysconfig" >/dev/null 2>&1; then
    if command -v python3 >/dev/null; then
      echo "\nerror: please install python3-distutils using your package manager"
    else
      echo "\nerror: please install python distutils"
    fi
    exit 1
  fi

  echo -e "\nInstalling the AWS CLI (/usr/local/bin/aws) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/awscli-bundle.zip https://s3.amazonaws.com/aws-cli/awscli-bundle.zip
  unzip $CORTEX_SH_TMP_DIR/awscli-bundle.zip -d $CORTEX_SH_TMP_DIR >/dev/null

  if [ $(id -u) = 0 ]; then
    $py_path $CORTEX_SH_TMP_DIR/awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws >/dev/null
  else
    ask_sudo
    sudo $py_path $CORTEX_SH_TMP_DIR/awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws >/dev/null
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed the AWS CLI"
}

function uninstall_aws() {
  set -e

  if ! command -v aws >/dev/null; then
    echo -e "\nThe AWS CLI is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/aws ]]; then
    echo -e "\nThe AWS CLI was not found at /usr/local/bin/aws, please uninstall it manually"
    return
  fi

  echo
  read -p "Would you like to uninstall the AWS CLI (/usr/local/bin/aws)? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ $(id -u) = 0 ]; then
      rm -rf /usr/local/aws
      rm /usr/local/bin/aws
    else
      ask_sudo
      sudo rm -rf /usr/local/aws
      sudo rm /usr/local/bin/aws
    fi
    rm -rf $HOME/.aws
    echo "✓ Uninstalled the AWS CLI"
  else
    return
  fi
}

function install_eksctl() {
  set -e

  if command -v eksctl >/dev/null; then
    echo -e "\neksctl is already installed"
    return
  fi

  check_dep_curl

  echo -e "\nInstalling eksctl (/usr/local/bin/eksctl) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  (cd $CORTEX_SH_TMP_DIR && curl -s --location "https://github.com/weaveworks/eksctl/releases/download/0.1.19/eksctl_$(uname -s)_amd64.tar.gz" | tar xz)
  chmod +x $CORTEX_SH_TMP_DIR/eksctl

  if [ $(id -u) = 0 ]; then
    mv $CORTEX_SH_TMP_DIR/eksctl /usr/local/bin/eksctl
  else
    ask_sudo
    sudo mv $CORTEX_SH_TMP_DIR/eksctl /usr/local/bin/eksctl
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed eksctl"
}

function uninstall_eksctl() {
  set -e

  if ! command -v eksctl >/dev/null; then
    echo -e "\neksctl is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/eksctl ]]; then
    echo -e "\neksctl was not found at /usr/local/bin/eksctl, please uninstall it manually"
    return
  fi

  echo
  read -p "Would you like to uninstall eksctl (/usr/local/bin/eksctl)? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ $(id -u) = 0 ]; then
      rm /usr/local/bin/eksctl
    else
      ask_sudo
      sudo rm /usr/local/bin/eksctl
    fi
    echo "✓ Uninstalled eksctl"
  else
    return
  fi
}

function install_aws_iam_authenticator() {
  set -e

  if command -v aws-iam-authenticator >/dev/null; then
    echo -e "\naws-iam-authenticator is already installed"
    return
  fi

  check_dep_curl

  echo -e "\nInstalling aws-iam-authenticator (/usr/local/bin/aws-iam-authenticator) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.11.5/2018-12-06/bin/$PARSED_OS/amd64/aws-iam-authenticator
  chmod +x $CORTEX_SH_TMP_DIR/aws-iam-authenticator

  if [ $(id -u) = 0 ]; then
    mv $CORTEX_SH_TMP_DIR/aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
  else
    ask_sudo
    sudo mv $CORTEX_SH_TMP_DIR/aws-iam-authenticator /usr/local/bin/aws-iam-authenticator
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed aws-iam-authenticator"
}

function uninstall_aws_iam_authenticator() {
  set -e

  if ! command -v aws-iam-authenticator >/dev/null; then
    echo -e "\naws-iam-authenticator is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/aws-iam-authenticator ]]; then
    echo -e "\naws-iam-authenticator was not found at /usr/local/bin/aws-iam-authenticator, please uninstall it manually"
    return
  fi

  echo
  read -p "Would you like to uninstall aws-iam-authenticator (/usr/local/bin/aws-iam-authenticator)? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ $(id -u) = 0 ]; then
      rm /usr/local/bin/aws-iam-authenticator
    else
      ask_sudo
      sudo rm /usr/local/bin/aws-iam-authenticator
    fi
    echo "✓ Uninstalled aws-iam-authenticator"
  else
    return
  fi
}

function install_cortex_cli() {
  set -e

  if command -v cortex >/dev/null; then
    echo "The Cortex CLI is already installed"
    return
  fi

  check_dep_curl
  check_dep_unzip

  echo -e "\nInstalling the Cortex CLI (/usr/local/bin/cortex) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/cortex-cli-${CORTEX_VERSION_STABLE}-${PARSED_OS}.zip https://s3-us-west-2.amazonaws.com/get-cortex/cortex-cli-${CORTEX_VERSION_STABLE}-${PARSED_OS}.zip
  unzip $CORTEX_SH_TMP_DIR/cortex-cli-${CORTEX_VERSION_STABLE}-${PARSED_OS}.zip -d $CORTEX_SH_TMP_DIR >/dev/null
  chmod +x $CORTEX_SH_TMP_DIR/cortex

  if [ $(id -u) = 0 ]; then
    mv $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  else
    ask_sudo
    sudo mv $CORTEX_SH_TMP_DIR/cortex /usr/local/bin/cortex
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed the Cortex CLI"

  bash_profile_path=$(get_bash_profile)
  if [ ! "$bash_profile_path" = "" ]; then
    if ! grep -Fxq "source <(cortex completion)" "$bash_profile_path"; then
      echo
      read -p "Would you like to modify your bash profile ($bash_profile_path) to enable cortex command completion and the cx alias? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "\nsource <(cortex completion)" >> $bash_profile_path
        echo "✓ Your bash profile ($bash_profile_path) has been updated"
        echo
        echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
        echo
        echo "Command to update your current terminal session:"
        echo "  source $bash_profile_path"
      else
        echo "Your bash profile has not been modified. If you would like to modify it manually, add this line to your bash profile:"
        echo "  source <(cortex completion)"
        echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
      fi
    fi
  else
    echo -e "\nIf your would like to enable cortex command completion and the cx alias, add this line to your bash profile:"
    echo "  source <(cortex completion)"
    echo "Note: \`bash_completion\` must be installed on your system for cortex command completion to function properly"
  fi

  operator_endpoint=$(get_operator_endpoint_or_empty)
  if [ "$operator_endpoint" != "" ]; then
    export CORTEX_OPERATOR_ENDPOINT="$operator_endpoint"
  fi

  echo
  echo "Running \`cortex configure\` ..."
  /usr/local/bin/cortex configure
}

function uninstall_cortex_cli() {
  set -e

  rm -rf $HOME/.cortex

  if ! command -v cortex >/dev/null; then
    echo -e "\nThe Cortex CLI is not installed"
    return
  fi

  if [[ ! -f /usr/local/bin/cortex ]]; then
    echo -e "\nThe Cortex CLI was not found at /usr/local/bin/cortex, please uninstall it manually"
    return
  fi

  if [ $(id -u) = 0 ]; then
    rm /usr/local/bin/cortex
  else
    ask_sudo
    sudo rm /usr/local/bin/cortex
  fi
  echo -e "\n✓ Uninstalled the Cortex CLI"

  bash_profile_path=$(get_bash_profile)
  if [ ! "$bash_profile_path" = "" ]; then
    if grep -Fxq "source <(cortex completion)" "$bash_profile_path"; then
      echo
      read -p "Would you like to remove \"source <(cortex completion)\" from your bash profile ($bash_profile_path)? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        sed '/^source <(cortex completion)$/d' "$bash_profile_path" > "${bash_profile_path}_cortex_modified" && mv -f "${bash_profile_path}_cortex_modified" "$bash_profile_path"
        echo "✓ Your bash profile ($bash_profile_path) has been updated"
      fi
    fi
  fi
}

function get_bash_profile() {
  if [ "$PARSED_OS" = "darwin" ]; then
    if [ -f $HOME/.bash_profile ]; then
      echo $HOME/.bash_profile
      return
    elif [ -f $HOME/.bashrc ]; then
      echo $HOME/.bashrc
      return
    fi
  else
    if [ -f $HOME/.bashrc ]; then
      echo $HOME/.bashrc
      return
    elif [ -f $HOME/.bash_profile ]; then
      echo $HOME/.bash_profile
      return
    fi
  fi

  echo ""
}

function ask_sudo() {
  if ! sudo -n true 2>/dev/null; then
    echo -e "\nPlease enter your sudo password"
  fi
}

######################
### ARG PROCESSING ###
######################

arg1=${1:-""}
arg2=${2:-""}
arg3=${3:-""}

if [ -z "$arg1" ]; then
  show_help
  exit 0
fi

if [ "$arg1" = "install" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for install command"
    show_help
    exit 1
  elif [ "$arg2" = "operator" ]; then
    install_operator
  elif [ "$arg2" = "cli" ]; then
    install_cli
  elif [ "$arg2" = "kubernetes-tools" ]; then
    install_kubernetes_tools
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for install"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for install: $arg2"
    show_help
    exit 1
  fi
elif [ "$arg1" = "uninstall" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for uninstall command"
    show_help
    exit 1
  elif [ "$arg2" = "operator" ]; then
    uninstall_operator
  elif [ "$arg2" = "cli" ]; then
    uninstall_cli
  elif [ "$arg2" = "kubernetes-tools" ]; then
    uninstall_kubernetes_tools
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for uninstall"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for uninstall: $arg2"
    show_help
    exit 1
  fi
elif [ "$arg1" = "update" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for update command"
    show_help
    exit 1
  elif [ "$arg2" = "operator" ]; then
    update_operator
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for update"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for update: $arg2"
    show_help
    exit 1
  fi
elif [ "$arg1" = "endpoints" ]; then
  if [ ! "$arg2" = "" ]; then
    echo -e "\nerror: too many arguments for endpoints command"
    show_help
    exit 1
  else
    get_endpoints
  fi
else
  echo -e "\nerror: unknown command: $arg1"
  show_help
  exit 1
fi
