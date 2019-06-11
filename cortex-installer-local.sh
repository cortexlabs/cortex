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
  ./cortex-installer.sh command [sub-command] [flags]

Available Commands:
  install all                 install operator, CLI, kubectl and minikube
  install operator            install the operator
  install cli                 install the CLI
  install kubernetes-tools    install kubectl, minikube

  uninstall operator          uninstall the operator
  uninstall cli               uninstall the CLI
  uninstall kubernetes-tools  uninstall kubectl
  uninstall all               uninstall operator, CLI, kubectl and minikube

  update operator             update the operator config and restart the operator

  get endpoints               show the operator and API endpoints

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

#####################
### CONFIGURATION ###
#####################

if [ "$CORTEX_CONFIG" != "" ] && [ -f "$CORTEX_CONFIG" ]; then
  source $CORTEX_CONFIG
fi

set -u

export CORTEX_VERSION_STABLE="master"
# Defaults
random_id=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-z0-9' | fold -w 12 | head -n 1)

export CORTEX_CLOUD_PROVIDER_TYPE="local"
export CORTEX_BUCKET="/mount"
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
export CORTEX_IMAGE_PYTHON_PACKAGER="${CORTEX_IMAGE_PYTHON_PACKAGER:-cortexlabs/python-packager:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_SERVE_GPU="${CORTEX_IMAGE_TF_SERVE_GPU:-cortexlabs/tf-serve-gpu:$CORTEX_VERSION_STABLE}"
export CORTEX_IMAGE_TF_TRAIN_GPU="${CORTEX_IMAGE_TF_TRAIN_GPU:-cortexlabs/tf-train-gpu:$CORTEX_VERSION_STABLE}"


export CORTEX_ENABLE_TELEMETRY=${CORTEX_ENABLE_TELEMETRY:-""}
export IMAGE_PULL_SECRETS=${IMAGE_PULL_SECRETS:-""}
################
### CHECK OS ###
################

case "$OSTYPE" in
  darwin*)  PARSED_OS="darwin" ;;
  linux*)   PARSED_OS="linux" ;;
  *)        echo -e "\nerror: only mac and linux are supported"; exit 1 ;;
esac

##########################
### TOP-LEVEL COMMANDS ###
##########################

function install_operator() {
  check_dep_curl
  check_dep_kubectl
  check_minikube_dep
  prompt_for_telemetry

  echo -e "\nInstalling the Cortex operator ..."

  
  setup_namespace

  if LC_ALL=C type setup_default_awsecr_creds >/dev/null 2>&1; then 
      setup_default_awsecr_creds
  fi

  setup_configmap
  local_mount
  setup_spark
  setup_argo
  setup_nginx
  setup_fluentd
  setup_operator

  validate_cortex
  get_endpoints
  auto_cortex_configure_cli
}

function install_kubernetes_tools() {
  install_kubectl
  # if [ "$PARSED_OS" = "darwin" ]; then
  #   install_hyperkit
  # fi
  install_minikube
}

function install_cli() {
  install_cortex_cli
}

function uninstall_operator() {
  check_dep_kubectl

  echo
  if kubectl get namespace $CORTEX_NAMESPACE >/dev/null 2>&1 || kubectl get customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1 || kubectl get customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io >/dev/null 2>&1 || kubectl get customresourcedefinition workflows.argoproj.io >/dev/null 2>&1; then
    echo "Uninstalling the Cortex operator from your Kubernetes cluster ..."

    # Remove finalizers on sparkapplications (they sometimes create deadlocks)
    if kubectl get namespace $CORTEX_NAMESPACE >/dev/null 2>&1 && kubectl get customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1; then
      set +e
      kubectl -n=$CORTEX_NAMESPACE get sparkapplications.sparkoperator.k8s.io -o name | xargs -L1 \
        kubectl -n=$CORTEX_NAMESPACE patch -p '{"metadata":{"finalizers": []}}' --type=merge >/dev/null 2>&1
      set -e
    fi

    kubectl delete --ignore-not-found=true customresourcedefinition scheduledsparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
    kubectl delete --ignore-not-found=true customresourcedefinition sparkapplications.sparkoperator.k8s.io >/dev/null 2>&1
    kubectl delete --ignore-not-found=true customresourcedefinition workflows.argoproj.io >/dev/null 2>&1
    echo "Deleting namespace $CORTEX_NAMESPACE"
    kubectl delete --ignore-not-found=true namespace $CORTEX_NAMESPACE >/dev/null 2>&1
    kubectl delete --ignore-not-found=true pv,pvc --all -n=$CORTEX_NAMESPACE >/dev/null 2>&1
    echo "✓ Uninstalled the Cortex operator"
  else
    echo "The Cortex operator is not installed on your Kubernetes cluster"
  fi
}

function uninstall_kubernetes_tools() {
  uninstall_kubectl
  uninstall_minikube
}

function uninstall_cli() {
  uninstall_cortex_cli
}

function install_all() {
  install_kubernetes_tools
  start_minikube
  install_operator
  auto_cortex_configure_cli
  install_cli
}

function uninstall_all() {
  uninstall_cli
  uninstall_operator
  uninstall_minikube
  uninstall_kubernetes_tools
}

# Note: if namespace is changed, the old namespace will not be deleted
function update_operator() {
  check_dep_curl
  check_dep_kubectl

  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true deployment operator >/dev/null 2>&1
  kubectl -n=$CORTEX_NAMESPACE delete --ignore-not-found=true daemonset fluentd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
  install_operator
}

function get_endpoints() {
  check_dep_kubectl

  operator_endpoint=$(get_operator_endpoint)
  api_port=$(kubectl -n=$CORTEX_NAMESPACE get svc nginx-controller-apis -o go-template='{{range.spec.ports}}{{if eq .name "https" }}{{ .nodePort }}{{"\n"}}{{end}}{{end}}')
  minikube_ip=$(minikube ip)
  apis_endpoint="http://$minikube_ip:$api_port"
  echo
  echo "Operator endpoint:  $operator_endpoint"
  echo "APIs endpoint:      $apis_endpoint"
}

function auto_cortex_configure_cli() {
  operator_endpoint=$(get_operator_endpoint)
echo -e "{
    \"cortex_url\": \"${operator_endpoint}\",
    \"cloud_provider_type\": \"local\",
    \"local\": {
        \"mount\": \"${HOME}/.cortex\"
    },
    \"aws\": null
}" > $HOME/.cortex/dev.json
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
    --from-literal='CLOUD_PROVIDER_TYPE'=$CORTEX_CLOUD_PROVIDER_TYPE \
    --from-literal='BUCKET'=$CORTEX_BUCKET \
    --from-literal='NAMESPACE'=$CORTEX_NAMESPACE \
    --from-literal='IMAGE_OPERATOR'=$CORTEX_IMAGE_OPERATOR \
    --from-literal='IMAGE_SPARK'=$CORTEX_IMAGE_SPARK \
    --from-literal='IMAGE_TF_TRAIN'=$CORTEX_IMAGE_TF_TRAIN \
    --from-literal='IMAGE_TF_SERVE'=$CORTEX_IMAGE_TF_SERVE \
    --from-literal='IMAGE_TF_API'=$CORTEX_IMAGE_TF_API \
    --from-literal='IMAGE_PYTHON_PACKAGER'=$CORTEX_IMAGE_PYTHON_PACKAGER \
    --from-literal='IMAGE_TF_TRAIN_GPU'=$CORTEX_IMAGE_TF_TRAIN_GPU \
    --from-literal='IMAGE_TF_SERVE_GPU'=$CORTEX_IMAGE_TF_SERVE_GPU \
    --from-literal='ENABLE_TELEMETRY'=$CORTEX_ENABLE_TELEMETRY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

##################
### ARGO SETUP ###
##################

# function setup_argo_service_accounts() {

# }

function setup_argo() {
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-executor
  namespace: ${CORTEX_NAMESPACE}
${IMAGE_PULL_SECRETS}
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
${IMAGE_PULL_SECRETS}
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
### LOCAL MOUNT ###
###################

function local_mount() {
  echo "
kind: PersistentVolume
apiVersion: v1
metadata:
  namespace: ${CORTEX_NAMESPACE}
  name: cortex-local-mount
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: \"${CORTEX_BUCKET}\"
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  namespace: ${CORTEX_NAMESPACE}
  name: cortex-shared-volume-claim
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
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
  name: sparkoperator
  namespace: ${CORTEX_NAMESPACE}
${IMAGE_PULL_SECRETS}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: sparkoperator
rules:
- apiGroups: ['']
  resources: ['pods']
  verbs: ['*']
- apiGroups: ['']
  resources: ['services', 'configmaps', 'secrets']
  verbs: ['create', 'get', 'delete']
- apiGroups: ['extensions']
  resources: ['ingresses']
  verbs: ['create', 'get', 'delete']
- apiGroups: ['']
  resources: ['nodes']
  verbs: ['get']
- apiGroups: ['']
  resources: ['events']
  verbs: ['create', 'update', 'patch']
- apiGroups: ['apiextensions.k8s.io']
  resources: ['customresourcedefinitions']
  verbs: ['create', 'get', 'update', 'delete']
- apiGroups: ['admissionregistration.k8s.io']
  resources: ['mutatingwebhookconfigurations']
  verbs: ['create', 'get', 'update', 'delete']
- apiGroups: ['sparkoperator.k8s.io']
  resources: ['sparkapplications', 'scheduledsparkapplications']
  verbs: ['*']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: sparkoperator
subjects:
  - kind: ServiceAccount
    name: sparkoperator
    namespace: ${CORTEX_NAMESPACE}
roleRef:
  kind: ClusterRole
  name: sparkoperator
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sparkoperator
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: sparkoperator
    app.kubernetes.io/version: v2.4.0-v1alpha1
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: sparkoperator
      app.kubernetes.io/version: v2.4.0-v1alpha1
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: sparkoperator
        app.kubernetes.io/version: v2.4.0-v1alpha1
      initializers:
        pending: []
    spec:
      serviceAccountName: sparkoperator
      volumes:
      - name: webhook-certs
        secret:
          secretName: spark-webhook-certs
      containers:
      - name: sparkoperator
        image: ${CORTEX_IMAGE_SPARK_OPERATOR}
        imagePullPolicy: Always
        volumeMounts:
        - name: webhook-certs
          mountPath: /etc/webhook-certs
        command: [\"/usr/bin/spark-operator\"]
        args:
          - -namespace=${CORTEX_NAMESPACE}
          - -enable-webhook=true
          - -webhook-svc-namespace=${CORTEX_NAMESPACE}
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
${IMAGE_PULL_SECRETS}
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
---
apiVersion: batch/v1
kind: Job
metadata:
  name: sparkoperator-init
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app.kubernetes.io/name: sparkoperator
    app.kubernetes.io/version: v2.4.0-v1alpha1
spec:
  backoffLimit: 3
  template:
    metadata:
      labels:
        app.kubernetes.io/name: sparkoperator
        app.kubernetes.io/version: v2.4.0-v1alpha1
    spec:
      serviceAccountName: sparkoperator
      restartPolicy: Never
      containers:
      - name: main
        image: gcr.io/spark-operator/spark-operator:v2.4.0-v1alpha1-latest
        imagePullPolicy: IfNotPresent
        command: ['/usr/bin/gencerts.sh', '-p', '-n', 'cortex']
---
kind: Service
apiVersion: v1
metadata:
  name: spark-webhook
  namespace: ${CORTEX_NAMESPACE}
spec:
  ports:
    - port: 443
      targetPort: 8080
      name: webhook
  selector:
    app.kubernetes.io/name: sparkoperator
    app.kubernetes.io/version: v2.4.0-v1alpha1
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
${IMAGE_PULL_SECRETS}
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
  use-proxy-protocol: \"false\"
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
    app.kubernetes.io/name: nginx-controller-apis
    app.kubernetes.io/part-of: ingress-nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-controller-apis
      app.kubernetes.io/part-of: ingress-nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-controller-apis
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
    app.kubernetes.io/name: nginx-controller-apis
    app.kubernetes.io/part-of: ingress-nginx
spec:
  type: NodePort
  selector:
    app.kubernetes.io/name: nginx-controller-apis
    app.kubernetes.io/part-of: ingress-nginx
  ports:
  - name: http
    port: 8080
    targetPort: 80
  - name: https
    port: 4433
    targetPort: 443
" | kubectl apply -f - >/dev/null
}

#####################
### FLUENTD SETUP ###
#####################

function setup_fluentd() {
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluentd
  namespace: ${CORTEX_NAMESPACE}
  labels:
    app: fluentd
${IMAGE_PULL_SECRETS}
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
      @type file
      path ${CORTEX_BUCKET}/local_logs/\${tag}
      append true
      add_path_suffix false
      format json
      <buffer tag>
        @type file
        path ${CORTEX_BUCKET}/local_logs/buffer
        flush_mode interval
      </buffer>
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
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: config
          mountPath: /fluentd/etc
        - name: cortex-shared-volume
          mountPath: ${CORTEX_BUCKET}
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
      - name: cortex-shared-volume
        persistentVolumeClaim:
          claimName: cortex-shared-volume-claim
" | kubectl apply -f - >/dev/null
}

######################
### OPERATOR SETUP ###
######################

function setup_operator() {
  check_minikube_dep
  operator_host_ip=$(minikube ip)
  echo "
apiVersion: v1
kind: ServiceAccount
metadata:
  name: operator
  namespace: ${CORTEX_NAMESPACE}
${IMAGE_PULL_SECRETS}
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
        - name: CORTEX_OPERATOR_HOST_IP
          value: $operator_host_ip
        volumeMounts:
          - name: cortex-config
            mountPath: /configs/cortex
          - name: cortex-shared-volume
            mountPath: ${CORTEX_BUCKET}  
      volumes:
        - name: cortex-config
          configMap:
            name: cortex-config
        - name: cortex-shared-volume
          persistentVolumeClaim:
            claimName: cortex-shared-volume-claim
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
  type: NodePort
  selector:
    workloadId: operator
  ports:
  - port: 8888
    targetPort: 8888
" | kubectl apply -f - >/dev/null
}

function validate_cortex() {
  set +e

  echo -en "\nWaiting for the Cortex operator to be ready "

  operator_pod_ready_cycles=0
  operator_endpoint=""

  while true; do
    echo -n "."
    sleep 5

    operator_pod_name=$(kubectl -n=$CORTEX_NAMESPACE get pods -o=name --sort-by=.metadata.creationTimestamp | grep "^pod/operator-" | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      operator_pod_ready_cycles=0
    else
      is_ready=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
      if [ "$is_ready" == "true" ]; then
        ((operator_pod_ready_cycles++))
      else
        operator_pod_ready_cycles=0
      fi
    fi

    if [ "$operator_pod_ready_cycles" == "0" ] && [ "$operator_pod_name" != "" ]; then
      num_restart=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restart -ge 2 ]]; then
        echo -e "\n\nAn error occurred when starting the Cortex operator. View the logs with:"
        echo "  kubectl logs $operator_pod_name --namespace=$CORTEX_NAMESPACE"
        exit 1
      fi
      continue
    fi

    if [[ $operator_pod_ready_cycles -lt 3 ]]; then
      continue
    fi

    echo " ✓"
    break
  done

  echo -e "\nCortex is ready!"
}

function get_operator_endpoint() {
  set -eo pipefail
  minikube service operator --url -n=$CORTEX_NAMESPACE 
}

# function get_apis_endpoint() {
#   set -eo pipefail
#   minikube service nginx-controller-apis --url -n=$CORTEX_NAMESPACE 
#   kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
# }

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
    read -p "kubectl is required. Would you like cortex-installer.sh to install it? [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      install_kubectl
    else
      exit 1
    fi
  fi

  if ! kubectl config current-context >/dev/null 2>&1; then
    echo -e "\nerror: kubectl is not configured to connect with your cluster. If you are using eksctl, you can run this command to configure kubectl:"
    echo "  eksctl utils write-kubeconfig --name=cortex"
    exit 1
  fi

  set +e
  get_nodes_output=$(kubectl get nodes -o jsonpath='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}' 2>/dev/null)
  if [ $? -ne 0 ]; then
    echo -e "\nerror: either your AWS credentials are incorrect or kubectl is not properly configured to connect with your cluster"
    echo "If you are using eksctl, you can run this command to re-configure kubectl:"
    echo "  eksctl utils write-kubeconfig --name=cortex"
    echo "If you are changing IAM users, you must edit the aws-auth ConfigMap (using your previous IAM credentials) to add the new IAM user; see https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html"
    exit 1
  fi
  set -e
  num_nodes_ready=$(echo $get_nodes_output | tr ';' "\n" | grep "Ready=True" | wc -l)
  if ! [[ $num_nodes_ready -ge 1 ]]; then
    echo -e "\nerror: your cluster has no registered nodes"
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

function check_virtualbox_dep() {
  if ! command -v virtualbox >/dev/null 2>&1; then
    echo
    read -p "virtualbox is a prerequisite. please see docs.cortex.dev/operator/virtualbox for more details" -n 1 -r
    echo
    exit 1
  fi
}

function check_minikube_dep() {
  if ! command -v minikube >/dev/null 2>&1; then
    echo
    read -p "minikube is required. Would you like cortex-installer-local.sh to install it? [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      install_minikube
    else
      exit 1
    fi
  fi
}

function install_minikube() {
  set -e

  if command -v minikube >/dev/null; then
    echo -e "\minikube is already installed"
    return
  fi

  check_dep_curl
  check_virtualbox_dep

  echo -e "\nInstalling minikube (/usr/local/bin/minikube) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -Lo $CORTEX_SH_TMP_DIR/minikube https://storage.googleapis.com/minikube/releases/latest/minikube-$PARSED_OS-amd64
  chmod +x $CORTEX_SH_TMP_DIR/minikube

  if [ $(id -u) = 0 ]; then
    mv $CORTEX_SH_TMP_DIR/minikube /usr/local/bin/minikube
  else
    ask_sudo
    sudo mv $CORTEX_SH_TMP_DIR/minikube /usr/local/bin/minikube
  fi

  rm -rf $CORTEX_SH_TMP_DIR
  echo "✓ Installed minikube"


}

function start_minikube() {
  check_minikube_dep

  mkdir -p $HOME/.cortex${CORTEX_BUCKET}
  minikube start --mount-string=$HOME/.cortex${CORTEX_BUCKET}:${CORTEX_BUCKET} --cpus=4 --memory=4096 --disk-size=30000mb --mount
}

function uninstall_minikube() {
  set -e

  if ! command -v minikube >/dev/null; then
    echo -e "\nminikube is not installed"
    return
  fi

  minikube stop
  minikube delete

  if [[ ! -f /usr/local/bin/minikube ]]; then
    echo -e "\nminikube was not found at /usr/local/bin/minikube, please uninstall it manually"
    return
  fi

  echo
  read -p "Would you like to uninstall minikube (/usr/local/bin/minikube)? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ $(id -u) = 0 ]; then
      rm /usr/local/bin/minikube
    else
      ask_sudo
      sudo rm /usr/local/bin/minikube
    fi
    rm -rf $HOME/.minikube
    echo "✓ Uninstalled minikube"
  else
    return
  fi
}

# function check_hyperkit_dep() {
#   if command -v hyperkit >/dev/null 2>&1; then
#     echo
#     read -p "hyperkit is required. Would you like cortex-installer-local.sh to install hyperkit? [Y/n] " -n 1 -r
#     echo
#     if [[ $REPLY =~ ^[Yy]$ ]]; then
#       install_hyperkit
#     else
#       exit 1
#     fi
#   fi
# }

# function install_hyperkit() {
#   set -e

#   if command -v hyperkit >/dev/null; then
#     echo -e "\nhyperkit is already installed"
#     return
#   fi

#   check_dep_curl

#   # https://www.virtualbox.org/svn/vbox/trunk/src/VBox/Installer/linux/install.sh
#   echo -e "\nInstalling hyperkit (/usr/local/bin/hyperkit) ..."

#   rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
#   git clone https://github.com/moby/hyperkit $CORTEX_SH_TMP_DIR

#   if [ $(id -u) = 0 ]; then
#     install -o root -g wheel -m 4755 $CORTEX_SH_TMP_DIR/docker-machine-driver-hyperkit /usr/local/bin/
#   else
#     ask_sudo
#     sudo install -o root -g wheel -m 4755 $CORTEX_SH_TMP_DIR/docker-machine-driver-hyperkit /usr/local/bin/
#   fi

#   rm -rf $CORTEX_SH_TMP_DIR
#   echo "✓ Installed hyperkit"
# }


# function uninstall_hyperkit() {
#   set -e

#   if ! command -v hyperkit >/dev/null; then
#     echo -e "\nhyperkit is not installed"
#     return
#   fi

#   if [[ ! -f /usr/local/bin/hyperkit ]]; then
#     echo -e "\nhyperkit was not found at /usr/local/bin/hyperkit, please uninstall it manually"
#     return
#   fi

#   echo
#   read -p "Would you like to uninstall hyperkit (/usr/local/bin/hyperkit)? [Y/n] " -n 1 -r
#   echo
#   if [[ $REPLY =~ ^[Yy]$ ]]; then
#     if [ $(id -u) = 0 ]; then
#       rm /usr/local/bin/hyperkit
#     else
#       ask_sudo
#       sudo rm /usr/local/bin/hyperkit
#     fi
#   else
#     return
#   fi
# }

function install_cortex_cli() {
  set -e

  if command -v cortex >/dev/null; then
    echo "The Cortex CLI is already installed"
    return
  fi

  check_dep_curl

  echo -e "\nInstalling the Cortex CLI (/usr/local/bin/cortex) ..."

  rm -rf $CORTEX_SH_TMP_DIR && mkdir -p $CORTEX_SH_TMP_DIR
  curl -s -o $CORTEX_SH_TMP_DIR/cortex https://s3-us-west-2.amazonaws.com/get-cortex/$CORTEX_VERSION_STABLE/cli/$PARSED_OS/cortex
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

function prompt_for_telemetry() {
  if [ "$CORTEX_ENABLE_TELEMETRY" != "true" ] && [ "$CORTEX_ENABLE_TELEMETRY" != "false" ]; then
    while true
    do
      echo
      read -p "Would you like to help improve Cortex by anonymously sending error reports and usage stats to the dev team? [Y/n] " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        export CORTEX_ENABLE_TELEMETRY=true
        break
      elif [[ $REPLY =~ ^[Nn]$ ]]; then
        export CORTEX_ENABLE_TELEMETRY=false
        break
      fi
      echo "Unexpected value, please enter \"Y\" or \"n\""
    done
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
  elif [ "$arg2" = "all" ]; then
    install_all
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
elif [ "$arg1" = "get" ]; then
  if [ ! "$arg3" = "" ]; then
    echo -e "\nerror: too many arguments for get command"
    show_help
    exit 1
  elif [ "$arg2" = "test" ]; then
    check_virtualization
  elif [ "$arg2" = "endpoints" ]; then
    get_endpoints
  elif [ "$arg2" = "" ]; then
    echo -e "\nerror: missing subcommand for get"
    show_help
    exit 1
  else
    echo -e "\nerror: invalid subcommand for get: $arg2"
    show_help
    exit 1
  fi
else
  echo -e "\nerror: unknown command: $arg1"
  show_help
  exit 1
fi
