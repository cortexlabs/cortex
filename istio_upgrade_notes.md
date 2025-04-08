## 1.18

    About
    Blog
    News
    Get involved
    Documentation

Istio 1.18 Upgrade Notes

Important changes to consider when upgrading to Istio 1.18.0.

Jun 7, 2023

When you upgrade from Istio 1.17.x to Istio 1.18.0, you need to consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.17.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.17.x.
Proxy Concurrency changes

Previously, the proxy concurrency setting, which configures how many worker threads the proxy runs, was inconsistently configured between sidecars and different gateway installation mechanisms. This often led to gateways running with concurrency based on the number of physical cores on the host machine, despite having CPU limits, leading to decreased performance and increased resource usage.

In this release, concurrency configuration has been tweaked to be consistent across deployment types. The new logic will use the ProxyConfig.Concurrency setting (which can be configured mesh wide or per-pod), if set, and otherwise set concurrency based on the CPU limit allocated to the container. For example, a limit of 2500m would set concurrency to 3.

Prior to this release, sidecars followed this logic, but sometimes incorrectly determined the CPU limit. Gateways would never automatically adapt based on concurrency settings.

To retain the old gateway behavior of always utilizing all cores, proxy.istio.io/config: concurrency: 0 can be set on each gateway. However, it is recommended to instead unset CPU limits if this is desired.
Gateway API Automated Deployment changes

This change impacts you only if you use Gateway API Automated Deployment. Note that this only applies to the Kubernetes Gateway API, not the Istio Gateway. You can check if you are using this feature with the following command:

$ kubectl get gateways.gateway.networking.k8s.io -ojson | jq -r '.items[] | select(.spec.gatewayClassName == "istio") | select((.spec.addresses | length) == 0) | "Found managed gateway: " + .metadata.namespace + "/" + .metadata.name'
Found managed gateway: default/gateway

If you see “Found managed gateway”, you may be impacted by this change.

Prior to Istio 1.18, the managed gateway worked by creating a minimal Deployment configuration which was fully populated at runtime with Pod injection. To upgrade gateways, users would restart the Pods to trigger a re-injection.

In Istio 1.18, this has changed to create a fully rendered Deployment and no longer rely on injection. As a result, Gateways will be updated, via a rolling restart, when their revision changes.

Additionally, users using this feature must update their control plane to Istio 1.16.5+ or 1.17.3+ before adopting Istio 1.18. Failure to do so may lead to conflicting writes to the same resources.
1.18.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

## 1.19


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.19 Upgrade Notes

Important changes to consider when upgrading to Istio 1.19.

Sep 5, 2023

When you upgrade from Istio 1.18.x to Istio 1.19.x, you need to consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.18.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.18.x.
Use the canonical filter names for EnvoyFilter

If you are using EnvoyFilter API, please use canonical filter names. The use of deprecated filter name is not supported. See the Envoy documentation for further details.
base Helm Chart removals

A number of configurations previously present in the the base Helm chart were copied to the istiod chart in a previous releases.

In this release, the duplicated configurations are fully removed from the base chart.

Below shows a mapping of old configuration to new configuration:
Old New
ClusterRole istiod  ClusterRole istiod-clusterrole
ClusterRole istiod-reader   ClusterRole istio-reader-clusterrole
ClusterRoleBinding istiod   ClusterRoleBinding istiod-clusterrole
Role istiod Role istiod
RoleBinding istiod  RoleBinding istiod
ServiceAccount istiod-service-account   ServiceAccount istiod

Note: most resources have a suffix automatically added in addition. In the old chart, this was -{{ .Values.global.istioNamespace }}. In the new chart it is {{- if not (eq .Values.revision "") }}-{{ .Values.revision }}{{- end }} for namespace scoped resources, and {{- if not (eq .Values.revision "")}}-{{ .Values.revision }}{{- end }}-{{ .Release.Namespace }} for cluster scoped resources.
EnvoyFilter must specify the type URL for an Envoy extension injection

Previously, Istio permitted a lookup of the extension in EnvoyFilter by its internal Envoy name alone. To see if you are affected, run istioctl analyze and check for a deprecation warning using deprecated types by name without typed_config. Additionally, make sure any nested extension lists inside EnvoyFilter include both name: and typed_config: fields.
Gateway API: Service-attached parentRefs must specify empty group

As a result of updates to the Gateway API conformance tests, Istio will no longer accept the default group of gateway.networking.k8s.io for a Service parentRef in a Gateway API route (e.g. HTTPRoute, TCPRoute, etc). Instead, you must explicitly set group: "" like so:

apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: productpage
spec:
  parentRefs:
  - group: ""
    kind: Service
    name: productpage
    port: 9080

1.19.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

## 1.20


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.20 Upgrade Notes

Important changes to consider when upgrading to Istio 1.20.

Nov 14, 2023

When you upgrade from Istio 1.19.x to Istio 1.20.x, you need to consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.19.x. The notes also mention changes that preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.19.x.
Upcoming ExternalName support changes

The following information describes upcoming changes to ExternalName.

In this release, there are no behavioral changes by default. However, you can explicitly opt in to the new behavior early if desired, and prepare your environments for the upcoming change.

Kubernetes ExternalName Services allow users to create new DNS entries. For example, you can create an example service that points to example.com. This is implemented by a DNS CNAME redirect.

In Istio, the implementation of ExternalName, historically, was substantially different. Each ExternalName represented its own service, and traffic matching the service was sent to the configured DNS name.

This caused a few issues:

    Ports are required in Istio, but not in Kubernetes. This can result in broken traffic if ports are not configured as Istio expects, despite them working without Istio.
    Ports not declared as HTTP would match all traffic on that port, making it easy to accidentally send all traffic on a port to the wrong place.
    Because the destination DNS name is treated as opaque, we cannot apply Istio policies to it as expected. For example, if an external name points to another in-cluster Service (for example, example.default.svc.cluster.local), mTLS would not be.

ExternalName support has been revamped to fix these problems. ExternalNames are now simply treated as aliases. Wherever we would match Host: <concrete service> we will additionally match Host: <external name service>. Note that the primary implementation of ExternalName DNS is handled outside of Istio in the Kubernetes DNS implementation, and remains unchanged.

If you are using ExternalName with Istio, please be advised of the following behavioral changes:

    The ports field is no longer needed, matching Kubernetes behavior. If it is set, it will have no impact.
    VirtualServices that match on an ExternalName service will generally no longer match. Instead, the match should be rewritten to the referenced service.
    DestinationRule can no longer apply to ExternalName services. Instead, create rules where the host references the service.

These changes are off-by-default in this release, but will be on-by-default in the near future. To opt in early, the ENABLE_EXTERNAL_NAME_ALIAS=true environment variable can be set.
Envoy filter ordering

This change impacts internal implementation of how Envoy filters are ordered. These filters run in order to implement various functionality.

The ordering is now consistent across inbound, outbound, and gateway proxy modes, as well as HTTP and TCP protocols:

    Metadata Exchange
    CUSTOM Authz
    WASM Authn
    Authn
    WASM Authz
    Authz
    WASM Stats
    Stats
    WASM unspecified

This changes the following areas:

    Inbound TCP filters now place Metadata Exchange before Authn.
    Gateway TCP filters now place stats after Authz, and CUSTOM Authz before Authn.

startupProbe added to sidecar by default

The sidecar container now comes with a startupProbe enabled by default. Startup probes run only at the start of the pod. Once the startup probe completes, readiness probes will continue.

By using a startup probe, we can poll for the sidecar to start more aggressively, without polling as aggressively throughout the entire pod’s lifecycle. On average, this improves pod startup time by roughly one second.

If the startup probe does not pass after 10 minutes, the pod will be terminated. Previously, the pod would never be terminated even if it was unable to start indefinitely.

If you do not want this feature, it can be disabled. However, you will want to tune the readiness probe accordingly.

The recommended values with the startup probe enabled (the new defaults):

readinessInitialDelaySeconds: 0
readinessPeriodSeconds: 15
readinessFailureThreshold: 4
startupProbe:
enabled: true
failureThreshold: 600

The recommended values to disable the startup probe (reverting the behavior to match older Istio versions):

readinessInitialDelaySeconds: 1
readinessPeriodSeconds: 2
readinessFailureThreshold: 30
startupProbe:
enabled: false

1.20.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

## 1.21


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.21 Upgrade Notes

Important changes to consider when upgrading to Istio 1.21.x.

Mar 13, 2024

When you upgrade from Istio 1.20.x to Istio 1.21.0, you need to consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.20.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.20.x.
Default value of the feature flag ENABLE_AUTO_SNI to true

auto-sni is enabled by default. This means SNI will be set automatically based on the downstream HTTP host/authority header if DestinationRule does not explicitly set the same.

If this is not desired, use the new compatibilityVersion feature to fallback to old behavior.
Default value of the feature flag VERIFY_CERT_AT_CLIENT is set to true

This means server certificates will be automatically verified using the OS CA certificates when not using a DestinationRule caCertificates field. If this is not desired, use the new compatibilityVersion feature to fallback to old behavior, or use the insecureSkipVerify field in DestinationRule to skip the verification.
ExternalName support changes

Kubernetes ExternalName Services allow users to create new DNS entries. For example, you can create an example service that points to example.com. This is implemented by a DNS CNAME redirect.

In Istio, the implementation of ExternalName, historically, was substantially different. Each ExternalName represented its own service, and traffic matching the service was sent to the configured DNS name.

This caused a few issues:

    Ports are required in Istio, but not in Kubernetes. This can result in broken traffic if ports are not configured as Istio expects, despite them working without Istio.
    Ports not declared as HTTP would match all traffic on that port, making it easy to accidentally send all traffic on a port to the wrong place.
    Because the destination DNS name is treated as opaque, we cannot apply Istio policies to it as expected. For example, if I point an external name at another in-cluster Service (for example, example.default.svc.cluster.local), mTLS would not be used.

ExternalName support has been revamped to fix these problems. ExternalNames are now simply treated as aliases. Wherever we would match Host: <concrete service> we additionally will match Host: <external name service>. Note that the primary implementation of ExternalName – DNS – is handled outside of Istio in the Kubernetes DNS implementation, and remains unchanged.

If you are using ExternalName with Istio, please be advised of the following behavioral changes:

    The ports field is no longer needed, matching Kubernetes behavior. If it is set, it will have no impact.
    VirtualServices that route to an ExternalName service will no longer work unless the referenced service exists (as a Service or ServiceEntry).
    DestinationRule can no longer apply to ExternalName services. Instead, create rules where the host references service.

To opt-out, the ENABLE_EXTERNAL_NAME_ALIAS=false environment variable can be set.

Note: the same change was introduced in the previous release, but off by default. This release turns the flag on by default.
Gateway Name label modified

If you are using the Kubernetes Gateway to manage your Istio gateways, the label key used to identify the gateway name is changing from istio.io/gateway-name to gateway.networking.k8s.io/gateway-name. The old label will continue to be appended to the relevant label sets for backwards compatibility, but it will be removed in a future release. Furthermore, istiod’s gateway controller will automatically detect and continue to use the old label for label selectors belonging to existing Deployment and Service resources.

Therefore, once you’ve completed your Istio upgrade, you can change the label selector in Deployment and Service resources whenever you are ready to use the new label.

Additionally, please upgrade any other policies, resources, or scripts that rely on the old label.
1.21.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases


## 1.22


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.22 Upgrade Notes

Important changes to consider when upgrading to Istio 1.22.x.

May 13, 2024

When you upgrade from Istio 1.21.x to Istio 1.22.0, you need to consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.21.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.21.x.
Delta xDS on by default

In previous versions, Istio used the “State of the world” xDS protocol to configure Envoy. In this release, the “Delta” xDS protocol is enabled by default.

This should be an internal implementation detail, but because this controls the core configuration protocol in Istio, an upgrade notice is present in an abundance of caution.

The expected impacts of this change is improved performance of configuration distribution. This may result in reduced CPU and memory utilization in Istiod and proxies, as well as less network traffic between the two. Note that while this release changes the protocol to be incremental, Istio does not yet send perfect minimal incremental updates. However, there are already optimizations in place for a variety of critical code paths, and this change enables us to continue optimizations.

If you experience unexpected impacts of this change, please set the ISTIO_DELTA_XDS=false environment variable in proxies and file a GitHub issue.
Default tracing to zipkin.istio-system.svc removed

In previous versions of Istio, tracing was automatically configured to send traces to zipkin.istio-system.svc. This default setting has been removed; users will need to explicitly configure where to send traces moving forward.

istioctl x precheck --from-version=1.21 can automatically detect if you may be impacted by this change.

If you previously had tracing enabled implicitly, you can enable it by doing one of:

    Installing with --set compatibilityVersion=1.21.
    Following Configure tracing with Telemetry API.

Default value of the feature flag ENHANCED_RESOURCE_SCOPING to true

ENHANCED_RESOURCE_SCOPING is enabled by default. This means that the pilot will processes only the Istio Custom Resource configurations that are in scope of what is specified from meshConfig.discoverySelectors. Root-ca certificate distribution is also affected.

If this is not desired, use the new compatibilityVersion feature to fallback to old behavior.
ServiceEntry with resolution: NONE now respects targetPort

ServiceEntry with resolution: NONE previously ignored any targetPort specifier. In this release, the targetPort is now respected. If undesired set --compatibilityVersion=1.21 to revert to the old behavior, or remove the targetPort specification.
New ambient mode waypoint attachment method

Waypoints in Istio’s ambient mode no longer use the original service account or namespace attachment semantics. If you were using a namespace-scope waypoint previously migration should be fairly straight forward. Label your namespace with the appropriate waypoint and it should function in a similar way. Please check the doc. If you were using service account attachment there will be more to understand.

Under the old waypoint logic all types of traffic, both addressed to a service as well as addressed to a workload, were treated similarly because there wasn’t a good way to properly associate a waypoint to a service. With the new attachment this limitation has been resolved. This includes adding a distinction between service addressed and workload addressed traffic. Annotating a service, or service-like kind, will redirect traffic which is service addressed to your waypoint. Likewise annotating a workload will redirect workload addressed traffic. It is therefore important to understand how consumers address your providers and select a waypoint attachment method which corresponds to this method of access.
1.22.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

## 1.23


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.23 Upgrade Notes

Important changes to consider when upgrading to Istio 1.23.0.

Aug 14, 2024

When upgrading from Istio 1.22.x to Istio 1.23.x, please consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.22.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.22.x.
Internal API protobuf changes

If you do not use Istio APIs from Go (via istio.io/api or istio.io/client-go) or Protobuf (from istio.io/api), this change does not impact you.

In prior versions, Istio APIs had identical contents replicated across multiple versions. For example, the same VirtualService protobuf message is defined 3 times (v1alpha3, v1beta1, and v1). These schemas are identical except in the package they reside in.

In this version of Istio, these have been consolidated down to a single version. For resources that had multiple versions, the oldest version is retained.

    If you use Istio APIs only via Kubernetes (YAML), there is no impact at all.
    If you use Istio APIs by Go types, there is essentially no impact. Each removed version has been replaced with type aliases to the remaining version, ensuring backwards compatibility. However, niche use cases (reflection, etc) may have some impact.
    If you use Istio APIs directly by Protobuf, and use newer versions, these will no longer be included as part of the API. Please reach out to the team if you are impacted.

1.23.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

## 1.24


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.24 Upgrade Notes

Important changes to consider when upgrading to Istio 1.24.0.

Nov 7, 2024

When upgrading from Istio 1.23.x to Istio 1.24.x, please consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.23.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.23.x.
Updated compatibility profiles

To support compatibility with older versions, Istio 1.24 introduces a new 1.23 compatibility profile and updates its other profiles to account for changes in Istio 1.24.

This profile sets the following values:

ENABLE_INBOUND_RETRY_POLICY: "false"
EXCLUDE_UNSAFE_503_FROM_DEFAULT_RETRY: "false"
PREFER_DESTINATIONRULE_TLS_FOR_EXTERNAL_SERVICES: "false"
ENABLE_ENHANCED_DESTINATIONRULE_MERGE: "false"
PILOT_UNIFIED_SIDECAR_SCOPE: "false"
ENABLE_DEFERRED_STATS_CREATION: "false"
BYPASS_OVERLOAD_MANAGER_FOR_STATIC_LISTENERS: "false"

See the individual change and upgrade notes for more information.
Ambient upgrade with DNS proxy

For upgrades to Istio 1.24.0 when using Ambient mode, with cni.ambient.dnsCapture=true configured, users will need to follow a specific set of upgrade steps:

    Upgrade Istio CNI
    Restart any workloads enrolled into ambient mode
    Upgrade Ztunnel

Failure to do so will result in DNS resolution failures. If this occurs, you can restart the workloads to resolve the issue.

This is expected to be improved in future patch releases; follow the issue for more information.
Istio CRDs are templated by default and can be installed and upgraded via helm install istio-base

This changes how CRDs are upgraded. Previously, we recommended and documented:

    Install: helm install istio-base
    Upgrade: kubectl apply -f manifests/charts/base/files/crd-all.gen.yaml or similar.
    Uninstall: kubectl get crd -oname | grep --color=never 'istio.io' | xargs kubectl delete

This change allows:

    Install: helm install istio-base
    Upgrade: helm upgrade istio-base
    Uninstall: kubectl get crd -oname | grep --color=never 'istio.io' | xargs kubectl delete

Previously this only worked under certain conditions, and when certain install flags were used, could result in non-Helm-upgradable CRDs being generated that required manual intervention to fix.

With this change, out-of-band install and upgrade of Istio CRDs with the kubectl command when using Helm is no longer required.

If you do not use Helm to install, template, or manage Istio resources, you can continue to do so and install CRDs manually with kubectl apply -f manifests/charts/base/files/crd-all.gen.yaml

If you previously installed CRDs with helm install istio-base OR kubectl apply, you can begin safely upgrading Istio CRDs with only helm upgrade istio-base from this and all subsequent releases after running the below kubectl commands as a one-time migration:

    kubectl label $(kubectl get crds -l chart=istio -o name && kubectl get crds -l app.kubernetes.io/part-of=istio -o name) "app.kubernetes.io/managed-by=Helm"
    kubectl annotate $(kubectl get crds -l chart=istio -o name && kubectl get crds -l app.kubernetes.io/part-of=istio -o name) "meta.helm.sh/release-name=istio-base" (replace with actual istio-base Helm release name)
    kubectl annotate $(kubectl get crds -l chart=istio -o name && kubectl get crds -l app.kubernetes.io/part-of=istio -o name) "meta.helm.sh/release-namespace=istio-system" (replace with actual istio namespace)

If desired, the legacy labels can be generated by setting base.enableCRDTemplates=false during helm install base, but this option will be removed in a future release.
istiod-remote chart replaced with remote profile

Installing istio clusters with a remote/external control plane via Helm has never been officially documented or stable. This changes how clusters that use a remote istio instance are installed, in preparation for documenting this.

The istiod-remote Helm chart has been merged with the regular istio-discovery Helm chart.

Previously:

    helm install istiod-remote istio/istiod-remote

With this change:

    helm install helm install istiod istio/istiod --set profile=remote

Note that, as per the above upgrade note, installing istio-base chart is now required in both local and remote clusters.
Sidecar scoping changes

During processing of services, Istio has a variety of conflict resolution strategies. Historically, these have subtly differed when a user has a Sidecar resource defined, compared to when they do not. This applied even if the Sidecar resource with just egress: "*/*", which should be the same as not having one defined.

In this version, the behavior between the two has been unified:

Multiple services defined with the same hostname Behavior before, without Sidecar: prefer a Kubernetes Service (rather than a ServiceEntry), else pick an arbitrary one. Behavior before, with Sidecar: prefer the Service in the same namespace as the proxy, else pick an arbitrary one. New behavior: prefer the Service in the same namespace as the proxy, then the Kubernetes Service (not ServiceEntry), else pick an arbitrary one.

Multiple Gateway API Route defined for the same service Behavior before, without Sidecar: prefer the local proxy namespace, to allow consumer overrides. Behavior before, with Sidecar: arbitrary order. New behavior: prefer the local proxy namespace, to allow consumer overrides.

The old behavior can be retained, temporarily, by setting PILOT_UNIFIED_SIDECAR_SCOPE=false.
Standardization of the peer metadata attributes

CEL expressions in the telemetry API must use the standard Envoy attributes instead of the custom Wasm extended attributes.

Peer metadata is now stored in filter_state.downstream_peer and filter_state.upstream_peer instead of filter_state["wasm.downstream_peer"] andfilter_state["wasm.upstream_peer"]. Node metadata is stored in xds.node instead of node. Wasm attributes must be fully qualified, e.g. use filter_state["wasm.istio_responseClass"] instead of istio_responseClass.

Presence operator can be used for backwards compatible expressions in a mixed proxy scenario, e.g. has(filter_state.downstream_peer) ? filter_state.downstream_peer.namespace : filter_state["wasm.downstream_peer"].namespace to read the namespace of the peer.

The peer metadata uses baggage encoding with the following field attributes:

    namespace
    cluster
    service
    revision
    app
    version
    workload
    type (e.g. "deployment")
    name (e.g. "pod-foo-12345")

Compatibility with cert-manager’s istio-csr

In this release, Istio introduces increased validation checks in gRPC communication to the control plane. Note this only impacts Istio’s own internal gRPC usage, not users’ traffic.

While Istio’s control plane is not impacted by this, a popular third-party CA implementation, istio-csr is. While this has been fixed upstream, there is not yet a released version with the fix at the time of writing (v0.12.0 does not have the fix).

This can be worked around in the meantime by installing Istio with the following settings:

meshConfig:
  defaultConfig:
    proxyMetadata:
      GRPC_ENFORCE_ALPN_ENABLED: "false"

If you are impacted by this issue, you will see an error message like "transport: authentication handshake failed: credentials: cannot check peer: missing selected ALPN property".
1.24.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases


## 1.25


    About
    Blog
    News
    Get involved
    Documentation

Istio 1.25 Upgrade Notes

Important changes to consider when upgrading to Istio 1.25.0.

Mar 3, 2025

When upgrading from Istio 1.24.x to Istio 1.25.x, please consider the changes on this page. These notes detail the changes which purposefully break backwards compatibility with Istio 1.24.x. The notes also mention changes which preserve backwards compatibility while introducing new behavior. Changes are only included if the new behavior would be unexpected to a user of Istio 1.24.x.
Ambient mode pod upgrade reconciliation

When a new istio-cni DaemonSet pod starts up, it will inspect pods that were previously enrolled in the ambient mesh, and upgrade their in-pod iptables rules to the current state if there is a diff or delta. This is off by default as of 1.25.0, but will eventually be enabled by default. This feature can be enabled by helm install cni --set ambient.reconcileIptablesOnStartup=true (Helm) or istioctl install --set values.cni.ambient.reconcileIptablesOnStartup=true (istioctl).
DNS traffic (TCP and UDP) now respects traffic exclusion annotations

DNS traffic (UDP and TCP) now respects pod-level traffic annotations like traffic.sidecar.istio.io/excludeOutboundIPRanges and traffic.sidecar.istio.io/excludeOutboundPorts. Before, UDP/DNS traffic would uniquely ignore these traffic annotations, even if a DNS port was specified, because of the rule structure. This behavior change actually happened in the 1.23 release series, but was left out of the release notes for 1.23.
Ambient mode DNS capture on by default

DNS proxying is enabled by default for ambient mode workloads in this release. Note that only new pods will have DNS enabled: existing pods will not have their DNS traffic captured. To enable this feature for existing pods, they must either be manually restarted, or alternatively the iptables reconciliation feature can be enabled when upgrading istio-cni via --set cni.ambient.reconcileIptablesOnStartup=true. This will reconcile existing pods automatically on upgrade.

Individual pods may opt-out of global ambient mode DNS capture by applying theambient.istio.io/dns-capture=false annotation.
Grafana dashboard changes

The dashboards shipped with Istio 1.25 require version 7.2 or later of Grafana.
OpenCensus support has been removed

Because Envoy has removed the OpenCensus tracing extension, we have removed OpenCensus support from Istio. If you are using OpenCensus, you should migrate to OpenTelemetry. Learn more about the deprecation of OpenCensus.
ztunnel Helm chart changes

In previous releases, resources in the ztunnel Helm chart were always named ztunnel. In this release, they are now named .Resource.Name.

If you are installing the chart with a release name other than ztunnel, the resource names will change, triggering downtime. In this scenario, it is recommended to set --set resourceName=ztunnel to override back to the previous default.
1.25.0
English
中文
Українська

    Terms and Conditions
    |
    Privacy policy
    |
    Trademarks
    |
    Edit this Page on GitHub

© 2025 the Istio Authors. Version Istio 1.25.1

    next releaseolder releases

