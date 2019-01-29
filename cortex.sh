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

##########################
### CHECK DEPENDENCIES ###
##########################

if ! command -v kubectl >/dev/null 2>&1 ; then
  echo "please install kubectl"
  exit 1
fi

if ! kubectl config current-context >/dev/null 2>&1 ; then
  echo "please make sure kubectl is configured correctly"
  exit 1
fi

if ! command -v aws >/dev/null 2>&1 ; then
  echo "please install the AWS CLI"
  exit 1
fi

############
### HELP ###
############

function show_help() {
  echo "
Usage:
  ./cortex.sh [command] [flags]

Available Commands:
  install    install cortex
  endpoints  get endpoints
  update     update cortex operator (without re-installing cortex)
  uninstall  uninstall cortex

Flags:
  -c, --config  path to a cortex config file
  -h, --help
"
}

###################
### ARG PARSING ###
###################

FLAG_HELP=false
POSITIONAL=()

while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -c|--config)
    export CORTEX_CONFIG="$2"
    shift
    shift
    ;;
    -h|--help)
    FLAG_HELP="true"
    shift
    ;;
    *)
    POSITIONAL+=("$1")
    shift
    ;;
  esac
done

set -- "${POSITIONAL[@]}"
POSITIONAL=()
for i in "$@"; do
  case $i in
    -c=*|--config=*)
    export CORTEX_CONFIG="${i#*=}"
    shift
    ;;
    -h=*|--help=*)
    FLAG_HELP="true"
    ;;
    *)
    POSITIONAL+=("$1")
    shift
    ;;
  esac
done

set -- "${POSITIONAL[@]}"
if [ "$FLAG_HELP" == "true" ]; then
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

ARG1=${1:-""}
ARG2=${2:-""}

if [ -z "$ARG1" ]; then
  show_help
  exit 0
fi

if [ "$ARG1" != "install" ] && [ "$ARG1" != "endpoints" ] && [ "$ARG1" != "update" ] && [ "$ARG1" != "uninstall" ] ; then
  echo "unknown command: $ARG1"
  show_help
  exit 1
fi

if [ -n "$ARG2" ]; then
  echo "too many arguments"
  show_help
  exit 1
fi

set -u

#####################
### CONFIGURATION ###
#####################

export CORTEX_CONFIG="${CORTEX_CONFIG:-config.sh}"
if [ -f "$CORTEX_CONFIG" ]; then
  source $CORTEX_CONFIG
fi

export CORTEX_VERSION=master

# Defaults
RANDOM_ID=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-z0-9' | fold -w 12 | head -n 1)

export CORTEX_LOG_GROUP="${CORTEX_LOG_GROUP:-cortex}"
export CORTEX_BUCKET="${CORTEX_BUCKET:-cortex-$RANDOM_ID}"
export CORTEX_REGION="${CORTEX_REGION:-us-west-2}"
export CORTEX_NAMESPACE="${CORTEX_NAMESPACE:-cortex}"

export CORTEX_IMAGE_ARGO_CONTROLLER="${CORTEX_IMAGE_ARGO_CONTROLLER:-cortexlabs/argo-controller:$CORTEX_VERSION}"
export CORTEX_IMAGE_ARGO_EXECUTOR="${CORTEX_IMAGE_ARGO_EXECUTOR:-cortexlabs/argo-executor:$CORTEX_VERSION}"
export CORTEX_IMAGE_FLUENTD="${CORTEX_IMAGE_FLUENTD:-cortexlabs/fluentd:$CORTEX_VERSION}"
export CORTEX_IMAGE_NGINX_BACKEND="${CORTEX_IMAGE_NGINX_BACKEND:-cortexlabs/nginx-backend:$CORTEX_VERSION}"
export CORTEX_IMAGE_NGINX_CONTROLLER="${CORTEX_IMAGE_NGINX_CONTROLLER:-cortexlabs/nginx-controller:$CORTEX_VERSION}"
export CORTEX_IMAGE_OPERATOR="${CORTEX_IMAGE_OPERATOR:-cortexlabs/operator:$CORTEX_VERSION}"
export CORTEX_IMAGE_SPARK="${CORTEX_IMAGE_SPARK:-cortexlabs/spark:$CORTEX_VERSION}"
export CORTEX_IMAGE_SPARK_OPERATOR="${CORTEX_IMAGE_SPARK_OPERATOR:-cortexlabs/spark-operator:$CORTEX_VERSION}"
export CORTEX_IMAGE_TF_SERVE="${CORTEX_IMAGE_TF_SERVE:-cortexlabs/tf-serve:$CORTEX_VERSION}"
export CORTEX_IMAGE_TF_TRAIN="${CORTEX_IMAGE_TF_TRAIN:-cortexlabs/tf-train:$CORTEX_VERSION}"
export CORTEX_IMAGE_TF_API="${CORTEX_IMAGE_TF_API:-cortexlabs/tf-api:$CORTEX_VERSION}"

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-""}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-""}"

if [ -z "$AWS_ACCESS_KEY_ID" ]; then
  echo "please set AWS_ACCESS_KEY_ID"
  exit 1
fi

if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
  echo "please set AWS_SECRET_ACCESS_KEY"
  exit 1
fi

#################
### AWS SETUP ###
#################

function setup_bucket() {
  if ! aws s3api head-bucket --bucket $CORTEX_BUCKET --output json 2>/dev/null; then
    if aws s3 ls "s3://$CORTEX_BUCKET" --output json 2>&1 | grep -q 'NoSuchBucket'; then
      echo "Creating S3 bucket: $CORTEX_BUCKET"
      aws s3api create-bucket --bucket $CORTEX_BUCKET \
                              --region $CORTEX_REGION \
                              --create-bucket-configuration LocationConstraint=$CORTEX_REGION \
                              > /dev/null
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
" | kubectl apply -f -
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
    -o yaml --dry-run | kubectl apply -f -
}

#######################
### SECRETS SETUP ###
#######################

function setup_secrets() {
  kubectl -n=$CORTEX_NAMESPACE create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run | kubectl apply -f -
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
" | kubectl apply -f -
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
" | kubectl apply -f -
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
" | kubectl apply -f -
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
" | kubectl apply -f -
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
" | kubectl apply -f -
}

function delete_operator() {
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true ingress operator
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true service operator
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true deployment operator
}

function validate_cortex() {
  echo ""
  echo "Validating cluster..."
  validation_errors="init"

  until [ "$validation_errors" == "" ]; do
    validation_errors=""

    out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]')
    if ! [[ $out = *'"loadBalancer":{"ingress":[{"'* ]]; then
      validation_errors="${validation_errors}\n  -> Waiting for load balancer (operator)"
    fi

    out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]')
    if ! [[ $out = *'"loadBalancer":{"ingress":[{"'* ]]; then
      validation_errors="${validation_errors}\n  -> Waiting for load balancer (APIs)"
    fi

    if [ "$validation_errors" != "" ]; then
      echo -e "${validation_errors:2}"
      sleep 15
    fi
  done

  status="down"
  operator_endpoint=$(kubectl -n=cortex get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
  until [ "$status" == "up" ]; do
    if curl $operator_endpoint > /dev/null 2>&1; then
      status="up"
    else
      echo "  -> Waiting for DNS (operator)"
      sleep 15
    fi
  done

  echo ""
  echo "Your cluster is ready!"

  get_endpoints

  echo ""
  echo "Please run 'cortex configure' to make sure your CLI is configured correctly"
}

function get_endpoints() {
  operator_endpoint=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
  apis_endpoint=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
  echo "operator endpoint:    $operator_endpoint"
  echo "APIs endpoint:        $apis_endpoint"
}

function install() {
  setup_bucket
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

function uninstall() {
  kubectl delete --ignore-not-found=true --wait=false namespace $CORTEX_NAMESPACE > /dev/null 2>&1
  kubectl delete --ignore-not-found=true --wait=false customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io > /dev/null 2>&1
  kubectl delete --ignore-not-found=true --wait=false customresourcedefinition sparkapplications.sparkoperator.k8s.io > /dev/null 2>&1
  kubectl delete --ignore-not-found=true --wait=false customresourcedefinition workflows.argoproj.io > /dev/null 2>&1
}

function update_operator() {
  delete_operator
  setup_configmap
  setup_operator
  validate_cortex
}

########################
### ARGUMENT PARSING ###
########################

if [ "$1" = "install" ]; then
  echo ""
  install

elif [ "$1" = "endpoints" ]; then
  echo ""
  get_endpoints

elif [ "$1" = "update" ]; then
  echo ""
  update_operator

elif [ "$1" = "uninstall" ]; then
  echo ""
  uninstall
  echo "Uninstalling Cortex"
fi
