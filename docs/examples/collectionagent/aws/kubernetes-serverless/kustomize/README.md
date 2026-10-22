# EKS Fargate Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `aws` |
| **Mode** | `kubernetes-serverless` |
| **Flavor** | `kustomize` |

## Overview

Deploys the SigNoz Collection Agents on an EKS cluster whose pods run on Fargate, as plain manifests composed by kustomize, exporting to SigNoz Cloud, Self-Hosted Enterprise or Self-Hosted Community.

On the `aws` platform, `mode: kubernetes-serverless` means EKS Fargate.

Fargate schedules no DaemonSet and mounts no host path, and a pod cannot reach the kubelet of the node it runs on. Two collector kinds cover a Fargate cluster, one casting each:

| Kind | Runs as | Collects |
| --- | --- | --- |
| `deployment` | one Deployment for the cluster | kubelet metrics for every Fargate node, cluster metrics, Kubernetes events, and OTLP from any pod through a Service |
| `sidecar` | a container inside each application pod that opts in | OTLP from the containers of its own pod |

Run the deployment in every cluster. Add the sidecar to the workloads whose telemetry you want collected inside their own pods.

### Deployment

A single collector runs as a Deployment and reaches every node through the Kubernetes API server.

It collects:

- Pod, container, node and volume metrics for every Fargate node, through the `kubeletstats` receiver. The `k8s_observer` extension discovers the nodes, and `receiver_creator` starts one `kubeletstats` scrape per node, proxied through the API server. The collector's own node is included. Only nodes labelled `eks.amazonaws.com/compute-type=fargate` are scraped, so in a mixed cluster EC2 nodes get no kubelet metrics from this casting.
- Cluster metrics (workload status, pod phase, node conditions, allocatables) through the `k8s_cluster` receiver
- Kubernetes events as logs through the `k8s_events` receiver
- OTLP traces, metrics and logs from your applications on `signoz-collector-deployment.<namespace>.svc:4317` (gRPC) and `signoz-collector-deployment.<namespace>.svc:4318` (HTTP)

Kubernetes metadata lands on every signal through the `k8sattributes` processor.

It cannot collect:

- Host metrics. Fargate exposes no host to a pod, so no `hostmetrics` receiver runs and the SigNoz Hosts view stays empty.
- Container stdout and stderr. Fargate mounts no `/var/log/pods`, so no `filelog` receiver runs. See [Logs](#logs) for the Fargate log router route.

The collector runs as one replica with the `Recreate` strategy: a rollout never runs an old and a new pod side by side, so no node is scraped twice.

The collector requests 0.25 vCPU and 512 MiB with equal limits. Fargate adds 256 MB to the request and rounds up, so the pod lands in the 0.25 vCPU / 1 GB tier.

This size covers a small cluster of a few dozen Fargate pods. For a larger cluster, raise it with a patch on the Deployment. The memory limiter is a percentage of the limit, so it follows:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
  patches:
    - target: collectionagent/collector/deployment/workload.yaml
      operations:
        - op: replace
          path: /spec/template/spec/containers/0/resources
          value:
            requests:
              cpu: 500m
              memory: 1Gi
            limits:
              cpu: 500m
              memory: 1Gi
```

With this patch the pod lands in the 0.5 vCPU / 2 GB tier.

### Sidecar

The sidecar runs as a native sidecar container inside each application pod that opts in, and exports to SigNoz directly. It collects application telemetry only:

- OTLP traces, metrics and logs from the pod's containers on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

The pod's name, UID, namespace and node land on every signal, and the `k8sattributes` processor adds the owning Deployment, StatefulSet, DaemonSet, Job or CronJob.

A workload opts in through two steps in your application's kustomization:

1. List the sidecar's pour once under `components:`.
2. Label each workload `foundry.signoz.io/kubernetes-collector-sidecar: enabled` on the object's own `metadata.labels`.

Labelled Deployment, StatefulSet, Job, CronJob, DaemonSet and ReplicaSet objects are patched. A label on the pod template alone patches nothing.

Pods without a sidecar keep sending to the deployment's Service. The usage metrics of every pod, with or without a sidecar, still come from the deployment's kubelet scrapes.

The sidecar requests 0.2 vCPU and 256 MiB with equal limits, so each injected pod grows by 0.2 vCPU and 0.25 GB before Fargate rounds it up to a tier.

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- An EKS cluster with a Fargate profile that selects the deployment's namespace (`signoz` by default, see [Annotations](#annotations))
- `kubectl` configured against the cluster
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)
- For the sidecar: Kubernetes 1.29 or later, which runs native sidecar containers, and applications deployed with kustomize from a kustomization that sets `namespace`
- For the `opentelemetry-operator` controller: the OpenTelemetry Operator, installed as in the [operator example](../../../kubernetes/kustomize-opentelemetry-operator/README.md#prerequisites)

## Configuration

### Deployment

This directory's `deployment/casting.yaml`:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
```

`SIGNOZ_INGESTION_ENDPOINT` is the OTLP endpoint the collector exports to. `K8S_CLUSTER_NAME` names the cluster on every signal as `k8s.cluster.name`. Set it: the SigNoz Kubernetes views require it on every entity. `spec.collector.spec.cluster.replicas` must stay unstated or `1`, since one collector scrapes every node.

### Sidecar

This directory's `sidecar/casting.yaml`:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
```

The variables mean the same as for the deployment. `spec.collector.spec.cluster.replicas` must stay unstated or `1`, since a sidecar runs once in every pod it joins.

### SigNoz Cloud or Self-Hosted Enterprise

Both authenticate ingestion with an [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/). For SigNoz Cloud, use the [endpoint for your region](https://signoz.io/docs/ingestion/signoz-cloud/overview/#endpoint); for Self-Hosted Enterprise, use your deployment's ingestion endpoint. Send the key as an exporter header:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: https://ingest.us.signoz.cloud:443
        SIGNOZ_INGESTION_KEY: <your-ingestion-key>
        K8S_CLUSTER_NAME: <your-cluster>
      config:
        data:
          collector/deployment/deployment.yaml: |
            exporters:
              otlphttp/signoz:
                headers:
                  signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
```

For the sidecar, state `kind: sidecar` and key the same override as `collector/sidecar/sidecar.yaml`.

Values under `spec.collector.spec.env` are visible in the Deployment manifest, and for the sidecar in every labelled workload.

### Self-Hosted Community

Set `SIGNOZ_INGESTION_ENDPOINT` to `http://<signoz-host>:4318` and set no key, as in `deployment/casting.yaml` and `sidecar/casting.yaml`.

## Deploy

### Deployment

```bash
foundryctl gauge -f deployment/casting.yaml
foundryctl forge -f deployment/casting.yaml -p deployment/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

Cast runs `kubectl apply -k` against the pours using your current kubeconfig context. To inspect what would be applied first:

```bash
kubectl kustomize deployment/pours/collectionagent/collector/deployment
```

Then point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://signoz-collector-deployment.<namespace>.svc:4317` (gRPC) or `http://signoz-collector-deployment.<namespace>.svc:4318` (HTTP).

### Sidecar

```bash
foundryctl gauge -f sidecar/casting.yaml
foundryctl forge -f sidecar/casting.yaml -p sidecar/pours
foundryctl cast -f sidecar/casting.yaml -p sidecar/pours
```

Cast applies nothing and prints the path of the component, `sidecar/pours/collectionagent/collector/sidecar`. List it under `components:` in your application's kustomization by a path relative to that kustomization, since kustomize refuses an absolute one:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: shop
resources:
  - checkout.yaml
components:
  - ../signoz/sidecar/pours/collectionagent/collector/sidecar
```

Then label each workload that takes a sidecar:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
  labels:
    foundry.signoz.io/kubernetes-collector-sidecar: enabled
spec:
  selector:
    matchLabels:
      app: checkout
  template:
    metadata:
      labels:
        app: checkout
    spec:
      containers:
        - name: checkout
          image: <your application image>
```

Apply your application as usual, for example with `kubectl apply -k`. The component adds to it:

- A Role and a RoleBinding that let every service account in the namespace read the pods, ReplicaSets, Jobs and CronJobs there, which `k8sattributes` reads the owners from
- The collector config as a ConfigMap
- The sidecar container and its config volume in each labelled workload

> [!NOTE]
> The component binds the namespace's service accounts through a kustomize `var`, so kustomize prints a deprecation warning for `vars` when it builds your application. The build is unaffected.

Then point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

## Generated output

```text
deployment/pours/collectionagent/collector/deployment/
  kustomization.yaml        # labels, resources, and the configMapGenerator
  namespace.yaml            # the collector's namespace
  serviceaccount.yaml       # the collector's service account
  clusterrole.yaml          # read access to cluster objects, plus nodes/stats and nodes/proxy
  clusterrolebinding.yaml   # binds the role to the service account
  service.yaml              # OTLP intake on 4317 and 4318
  workload.yaml             # the Deployment, one replica, Recreate strategy
  deployment.yaml           # the collector config
```

The configMapGenerator mounts `deployment.yaml` at `/conf`. A config change re-hashes the ConfigMap name and rolls the collector on the next apply.

```text
sidecar/pours/collectionagent/collector/sidecar/
  kustomization.yaml        # the component: resources, the configMapGenerator, the var, and the patches
  kustomizeconfig.yaml      # points the var at the RoleBinding's subject
  role.yaml                 # read access to pods, ReplicaSets, Jobs and CronJobs in the namespace
  rolebinding.yaml          # binds the role to every service account in the namespace
  patch-template.yaml       # the sidecar for labelled Deployments, StatefulSets, Jobs, DaemonSets and ReplicaSets
  patch-cronjob.yaml        # the sidecar for labelled CronJobs
  sidecar.yaml              # the collector config
```

A change to the sidecar's config re-hashes its ConfigMap name and rolls the labelled workloads the next time your application is applied.

## Controller

The `foundry.signoz.io/kubernetes-collector-controller` annotation names the controller that owns the collector workload. `default` pours the Deployment or the sidecar container for the built-in Kubernetes controllers. `opentelemetry-operator` pours an `OpenTelemetryCollector` resource for the OpenTelemetry Operator to reconcile. State it on either casting.

Under the operator:

- The deployment pours `opentelemetrycollector.yaml` in place of `service.yaml` and `workload.yaml`, with the config inline, and cast applies it. The operator runs it with the same `Recreate` rollout and size.
- The sidecar pours an `OpenTelemetryCollector` in sidecar mode into the collector's namespace, which cast applies, and the component in a `component/` directory beside it. The component's patches add `sidecar.opentelemetry.io/inject: <namespace>/signoz-collector-sidecar` to each labelled workload's pod template, and the operator injects the sidecar into each new pod. List `sidecar-operator/pours/collectionagent/collector/sidecar/component` under `components:` and label the workloads as for the default controller.

This directory's `sidecar-operator/casting.yaml`:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
```

```text
sidecar-operator/pours/collectionagent/collector/sidecar/
  kustomization.yaml            # namespace, labels, and resources, applied by cast
  namespace.yaml                # the collector's namespace
  opentelemetrycollector.yaml   # the sidecar collector, config inline
  component/
    kustomization.yaml          # the component: resources, the var, and the patches
    kustomizeconfig.yaml        # points the var at the RoleBinding's subject
    role.yaml                   # read access to pods, ReplicaSets, Jobs and CronJobs in the namespace
    rolebinding.yaml            # binds the role to every service account in the namespace
    patch-template.yaml         # the injection annotation for labelled Deployments, StatefulSets, Jobs, DaemonSets and ReplicaSets
    patch-cronjob.yaml          # the injection annotation for labelled CronJobs
```

## Logs

Nothing in the pour collects container logs. Fargate ships a managed Fluent Bit log router that can write them to CloudWatch Logs, and the collector can then read them from there. This takes four steps.

1. Create the `aws-observability` namespace the log router reads its config from:

```yaml
kind: Namespace
apiVersion: v1
metadata:
  name: aws-observability
  labels:
    aws-observability: enabled
```

2. Create the `aws-logging` ConfigMap in it, and attach the [CloudWatch Logs permissions](https://raw.githubusercontent.com/aws-samples/amazon-eks-fluent-logging-examples/mainline/examples/fargate/cloudwatchlogs/permissions.json) to the pod execution role of your Fargate profile:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: aws-logging
  namespace: aws-observability # Namespace should be aws-observability
data:
  flb_log_cw: "false"  # Set to true to ship Fluent Bit process logs to CloudWatch.
  filters.conf: |
    [FILTER]
        Name parser
        Match *
        Key_name log
        Parser crio
    [FILTER]
        Name kubernetes # Kubernetes Log Filter
        Match kube.*
        Merge_Log On
        Keep_Log Off
        Buffer_Size 0
        Kube_Meta_Cache_TTL 300s
        K8S-Logging.Parser On
        K8S-Logging.Exclude On
  output.conf: |
    [OUTPUT]
        Name cloudwatch_logs
        Match   kube.*
        region us-east-1
        log_group_name <LOG_GROUP> # Set to log group
        log_stream_prefix <LOG_STREAM_PREFIX> # Set the prefix you want to use for Log Stream
        log_retention_days 60
        auto_create_group false # Set to true to auto-create the log group if it does not exist
  parsers.conf: |
    [PARSER]
        Name crio
        Format Regex
        Regex ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>P|F) (?<log>.*)$
        Time_Key    time
        Time_Format %Y-%m-%dT%H:%M:%S.%L%z
```

3. Create an IAM role for the collector's service account (IRSA) with the `CloudWatchReadOnlyAccess` policy, trusted by your cluster's OIDC provider for `system:serviceaccount:<namespace>:signoz-collector-deployment`.

4. Annotate the service account with that role through a patch, and state the `awscloudwatch` receiver in the logs pipeline. Overrides replace lists wholesale, so the pipeline's receivers are restated in full:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
      config:
        data:
          collector/deployment/deployment.yaml: |
            receivers:
              awscloudwatch:
                region: us-east-1
                logs:
                  poll_interval: 1m
                  groups:
                    autodiscover:
                      limit: 100
                      prefix: <LOG_GROUP>
                      streams:
                        prefixes: [<LOG_STREAM_PREFIX>]
            service:
              pipelines:
                logs:
                  receivers:
                    - otlp/http
                    - otlp/grpc
                    - k8s_events
                    - awscloudwatch
  patches:
    - target: collectionagent/collector/deployment/serviceaccount.yaml
      operations:
        - op: add
          path: /metadata/annotations
          value:
            eks.amazonaws.com/role-arn: arn:aws:iam::<account-id>:role/<collector-role>
```

Forge and cast again, then restart your application deployments so the log router starts for their pods.

The Fluent Bit Kubernetes filter records the pod as `kubernetes.pod_id` inside the log record, while `k8sattributes` associates logs by pod IP, pod UID or connection, so these logs carry no Kubernetes metadata from the collector unless you lift that field yourself.

Kinesis Data Firehose is an alternative log router output and avoids polling CloudWatch, but Firehose delivers to an HTTP endpoint, so the collector would need an endpoint reachable from AWS.

## When the API-server proxy does not serve kubelet stats

The collector logs errors from the `kubeletstats` scrapes, and the SigNoz pods view stays empty. Read the collector's logs:

```bash
kubectl logs -n <namespace> deployment/signoz-collector-deployment
```

Two errors point here:

- A `404` on `/api/v1/nodes/<node>/proxy/stats/summary`: the kubelet on that node does not expose the summary through the API-server proxy.
- A `403`: the collector's service account is not allowed through the proxy.

Check first that the poured RBAC is in place. The ClusterRole grants `get` on `nodes/stats` and `nodes/proxy`; this must print `yes`:

```bash
kubectl auth can-i get nodes/proxy --as=system:serviceaccount:<namespace>:signoz-collector-deployment
```

If it is in place and the scrapes still fail, scrape the kubelets directly instead. The override switches the `kubeletstats` scrape to the service account and the kubelet's own address, skips certificate verification since the kubelets' certificates are self-signed, and skips the node the collector runs on, since a Fargate pod cannot reach the kubelet of its own node:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
      config:
        data:
          collector/deployment/deployment.yaml: |
            receivers:
              receiver_creator:
                receivers:
                  kubeletstats:
                    config:
                      auth_type: serviceAccount
                      endpoint: '`endpoint`:`kubelet_endpoint_port`'
                      insecure_skip_verify: true
                    rule: type == "k8s.node" && labels["eks.amazonaws.com/compute-type"] == "fargate" && name != "${env:K8S_NODE_NAME}"
```

Every node is scraped except the one hosting the collector, so the collector's own pod has no pod or container metrics.

Covering the collector's own node needs a second collector on another node that scrapes only that one node. A Fargate node is replaced whenever its pod is rescheduled, so the second collector has to be re-pointed each time the first one moves. Foundry does not provide this. Without it, what is lost is the collector's own resource usage, nothing of your applications.

## Customization

Override any collector setting through `spec.collector.spec.config.data`, keyed by the generated file's path. For example, to scrape the kubelets every minute:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
      config:
        data:
          collector/deployment/deployment.yaml: |
            receivers:
              receiver_creator:
                receivers:
                  kubeletstats:
                    config:
                      collection_interval: 60s
```

For the sidecar, key the override as `collector/sidecar/sidecar.yaml`.

For changes to the generated manifests themselves, use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-collector-controller` | `default` | Controller that owns the collector workload: `default` pours the Deployment or the sidecar container for the built-in Kubernetes controllers, `opentelemetry-operator` pours an `OpenTelemetryCollector` resource for the operator to reconcile. See [Controller](#controller). |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the deployment, or the sidecar's `OpenTelemetryCollector` under the operator, is deployed into. Your Fargate profile must select it for the deployment. The sidecar's component takes the namespace of the kustomization that includes it. |

Example deploying into a namespace of its own:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-namespace: observability
spec:
  deployment:
    platform: aws
    mode: kubernetes-serverless
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <your-cluster>
```
