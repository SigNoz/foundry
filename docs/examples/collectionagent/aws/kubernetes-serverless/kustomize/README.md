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

Two collector kinds cover a Fargate cluster, one casting each:

| Kind | Runs as | Collects |
| --- | --- | --- |
| `deployment` | one Deployment for the cluster | kubelet metrics for every Fargate node, cluster metrics, Kubernetes events, and OTLP from any pod through a Service |
| `sidecar` | a container inside each application pod that opts in | OTLP from the containers of its own pod |

Run the deployment in every cluster. Add the sidecar to the workloads whose telemetry you want collected inside their own pods.

### Deployment

A single collector runs as a Deployment and reaches every node through the Kubernetes API server. It collects:

- Pod, container, node and volume metrics for every node labelled `eks.amazonaws.com/compute-type=fargate`, through the `kubeletstats` receiver
- Cluster metrics (workload status, pod phase, node conditions, allocatables) through the `k8s_cluster` receiver
- Kubernetes events as logs through the `k8s_events` receiver
- OTLP traces, metrics and logs from your applications on `signoz-collector-deployment.<namespace>.svc:4317` (gRPC) and `signoz-collector-deployment.<namespace>.svc:4318` (HTTP), or on `signoz-collector-deployment-collector.<namespace>.svc` under the [OpenTelemetry Operator](#controller)

The `k8sattributes` processor adds the Kubernetes resource attributes to every signal.

The SigNoz Hosts view stays empty on Fargate. Container stdout and stderr reach SigNoz through the Fargate log router, as shown in [Logs](#logs).

The collector requests 0.25 vCPU and 512 MiB with equal limits, so its pod lands in the 0.25 vCPU / 1 GB Fargate tier. This size covers a small cluster of a few dozen Fargate pods. For a larger cluster, raise it with a patch on the Deployment; the memory limiter follows the limit:

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

The sidecar runs as a native sidecar container inside each application pod that opts in. It collects OTLP traces, metrics and logs from the pod's containers on `localhost:4317` (gRPC) and `localhost:4318` (HTTP), and exports them to SigNoz.

Every signal carries the pod's name, UID, namespace and node, and its owning Deployment, StatefulSet, DaemonSet, Job or CronJob, as resource attributes.

Your application's kustomization includes the sidecar's pour, and each workload takes the sidecar through a patch, or through an annotation under the [OpenTelemetry Operator](#controller), as shown in [Deploy](#deploy).

Pods without a sidecar keep sending to the deployment's Service, and the deployment's kubelet scrapes cover the usage metrics of every pod.

The sidecar requests 0.2 vCPU and 256 MiB with equal limits, so each pod that runs it grows by 0.2 vCPU and 0.25 GB before Fargate rounds it up to a tier.

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- An EKS cluster with a Fargate profile that selects the deployment's namespace (`signoz` by default, see [Annotations](#annotations))
- `kubectl` configured against the cluster
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)
- For the sidecar: Kubernetes 1.29 or later, which runs native sidecar containers
- For the `opentelemetry-operator` controller: cert-manager and the OpenTelemetry Operator, installed as in the [operator example](../../../kubernetes/kustomize-opentelemetry-operator/README.md#prerequisites), with two Fargate specifics:
  - cert-manager's webhook must listen on a port other than 10250, since on Fargate the pod shares its IP with the kubelet, which holds 10250. With Helm, set `webhook.securePort=10260`; with the static manifest, patch the `cert-manager-webhook` Deployment's `--secure-port` argument and the Service's `targetPort` to 10260.
  - Apply the operator manifest only after cert-manager's webhook answers (`kubectl -n cert-manager rollout status deploy/cert-manager-webhook`). Applied earlier, its `Issuer` and `Certificate` are rejected by the webhook and the operator pod waits forever for its serving certificate.

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

`SIGNOZ_INGESTION_ENDPOINT` is the OTLP endpoint the collector exports to. `K8S_CLUSTER_NAME` names the cluster on every signal as `k8s.cluster.name`, which the SigNoz Kubernetes views need. Leave `spec.collector.spec.cluster.replicas` unstated or set it to `1`.

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

The variables mean the same as for the deployment. Leave `spec.collector.spec.cluster.replicas` unstated or set it to `1`.

### Sidecar under the OpenTelemetry Operator

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

The `foundry.signoz.io/kubernetes-collector-controller` annotation hands the sidecar to the OpenTelemetry Operator, see [Controller](#controller). The same annotation on `deployment/casting.yaml` runs the deployment under the operator.

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

### Self-Hosted Community

Set `SIGNOZ_INGESTION_ENDPOINT` to `http://<signoz-host>:4318` and set no key, as in `deployment/casting.yaml` and `sidecar/casting.yaml`.

## Deploy

### Deployment

```bash
foundryctl gauge -f deployment/casting.yaml
foundryctl forge -f deployment/casting.yaml -p deployment/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

Cast applies the pour with `kubectl apply -k` against your current kubectl context. To inspect what it applies first:

```bash
kubectl kustomize deployment/pours/collectionagent/collector/deployment
```

Then point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://signoz-collector-deployment.<namespace>.svc:4317` (gRPC) or `http://signoz-collector-deployment.<namespace>.svc:4318` (HTTP). Under the operator, the Service is `signoz-collector-deployment-collector.<namespace>.svc`.

### Sidecar

These steps add the sidecar through your application's kustomization, in the directory `<overlay>`.

1. Gauge the casting and forge it into your application's kustomization directory:

   ```bash
   foundryctl gauge -f sidecar/casting.yaml
   foundryctl forge -f sidecar/casting.yaml -p <overlay>/signoz/pours
   ```

   To forge anywhere else, copy `patch.yaml` from the pour beside your kustomization after each forge, and list it as `path: patch.yaml`.

2. List the pour under `resources:` and `patch.yaml` under `patches:`, with a `target` for each workload that takes the sidecar. A CronJob takes `patch-cronjob.yaml`:

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   namespace: <your-namespace>
   resources:
     - checkout.yaml
     - report.yaml
     - signoz/pours/collectionagent/collector/sidecar
   patches:
     - path: signoz/pours/collectionagent/collector/sidecar/patch.yaml
       target:
         kind: Deployment
         name: checkout
     - path: signoz/pours/collectionagent/collector/sidecar/patch-cronjob.yaml
       target:
         kind: CronJob
         name: report
   ```

   The kustomization's `namespace:` places the sidecar's config and permissions in your application's namespace.

3. The poured RoleBinding grants the sidecar's permissions to the namespace's `default` service account. If your workloads run under service accounts of their own, add `signoz-rolebinding.yaml` beside your kustomization, listing every account that runs the sidecar, `default` included while any workload that takes the sidecar runs under it:

   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: RoleBinding
   metadata:
     name: signoz-collector-sidecar
   subjects:
     - kind: ServiceAccount
       name: checkout
     - kind: ServiceAccount
       name: default
   ```

   Add it to the kustomization's `patches:`:

   ```yaml
   patches:
     - path: signoz-rolebinding.yaml
   ```

4. Apply your application as usual, for example:

   ```bash
   kubectl apply -k <overlay>
   ```

   Then point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

5. Verify:

   - Each pod of a workload that takes the sidecar runs the `signoz-collector-sidecar` container. This prints it once per pod:

     ```bash
     kubectl -n <your-namespace> get pods -l app=checkout -o jsonpath='{.items[*].spec.initContainers[*].name}'
     ```

   - Your application's traces, metrics and logs appear in SigNoz with the `k8s.pod.name` and `k8s.namespace.name` resource attributes.

After a config change, forge again, apply your application again, and restart each workload that runs the sidecar:

```bash
kubectl -n <your-namespace> rollout restart deployment/checkout
```

For a workload managed without kustomize, forge with `-p sidecar/pours`, copy the `initContainers` and `volumes` entries from `patch.yaml` into the workload's pod template, and apply the pour into its namespace:

```bash
kubectl apply -n <your-namespace> -k sidecar/pours/collectionagent/collector/sidecar
```

### Sidecar under the OpenTelemetry Operator

1. Gauge the casting and forge it into your application's kustomization directory:

   ```bash
   foundryctl gauge -f sidecar-operator/casting.yaml
   foundryctl forge -f sidecar-operator/casting.yaml -p <overlay>/signoz/pours
   ```

2. List the pour under `resources:`:

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   namespace: <your-namespace>
   resources:
     - checkout.yaml
     - signoz/pours/collectionagent/collector/sidecar
   ```

   Add the `sidecar.opentelemetry.io/inject: signoz-collector-sidecar` annotation to the pod template of each workload that takes the sidecar:

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: checkout
   spec:
     selector:
       matchLabels:
         app: checkout
     template:
       metadata:
         labels:
           app: checkout
         annotations:
           sidecar.opentelemetry.io/inject: signoz-collector-sidecar
       spec:
         containers:
           - name: checkout
             image: <your application image>
   ```

3. If your workloads run under service accounts of their own, add `signoz-rolebinding.yaml` as in step 3 of the sidecar steps above.

4. Apply your application as usual. The operator adds the sidecar to each new pod, so restart the workloads whose pods were already running:

   ```bash
   kubectl -n <your-namespace> rollout restart deployment/checkout
   ```

5. Verify:

   - `kubectl -n <your-namespace> get pods -l app=checkout` shows each pod with one more ready container than your application runs, `2/2` for the Deployment above.
   - Your application's traces, metrics and logs appear in SigNoz with the `k8s.pod.name` and `k8s.namespace.name` resource attributes.

After a config change, forge again, apply your application again, and restart the workloads that take the sidecar.

## Generated output

```text
deployment/pours/collectionagent/collector/deployment/
  kustomization.yaml        # the kustomization cast applies
  namespace.yaml            # the collector's namespace
  serviceaccount.yaml       # the collector's service account
  clusterrole.yaml          # read access to cluster objects and the kubelet stats
  clusterrolebinding.yaml   # grants the cluster role to the collector's service account
  service.yaml              # receives OTLP on 4317 and 4318
  workload.yaml             # runs the collector as a Deployment
  deployment.yaml           # the collector config
```

A config change rolls the collector on the next cast. Under the operator, `opentelemetrycollector.yaml` holds the collector and its config in place of `service.yaml`, `workload.yaml` and `deployment.yaml`.

```text
sidecar/pours/collectionagent/collector/sidecar/
  kustomization.yaml        # the pour your application's kustomization lists under resources
  role.yaml                 # read access to pods, ReplicaSets, Jobs and CronJobs in the namespace
  rolebinding.yaml          # grants the role to the namespace's default service account
  patch.yaml                # adds the sidecar to a workload's pod template
  patch-cronjob.yaml        # adds the sidecar to a CronJob's pod template
  sidecar.yaml              # the collector config
```

```text
sidecar-operator/pours/collectionagent/collector/sidecar/
  kustomization.yaml            # the pour your application's kustomization lists under resources
  role.yaml                     # read access to pods, ReplicaSets, Jobs and CronJobs in the namespace
  rolebinding.yaml              # grants the role to the namespace's default service account
  opentelemetrycollector.yaml   # the sidecar the operator adds to each pod, with its config
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

Covering the collector's own node needs a second collector on another node that scrapes only that one node. A Fargate node is replaced whenever its pod is rescheduled, so the second collector has to be re-pointed each time the first one moves. Without it, what is lost is the collector's own resource usage, nothing of your applications.

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

## Controller

The `foundry.signoz.io/kubernetes-collector-controller` annotation names the controller that runs the collector, on either casting:

- `default` pours the Deployment, or the sidecar's `patch.yaml`, for the built-in Kubernetes controllers.
- `opentelemetry-operator` pours an `OpenTelemetryCollector` resource with the config inline, which the OpenTelemetry Operator runs: the deployment as a Deployment that cast applies, and the sidecar as a container the operator adds to each pod that carries the inject annotation.

The operator needs cert-manager, as listed in [Prerequisites](#prerequisites). [Deploy](#deploy) covers the steps for each kind.

## Annotations

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-collector-controller` | `default` | Controller that runs the collector: `default` for the built-in Kubernetes controllers, `opentelemetry-operator` for the OpenTelemetry Operator. See [Controller](#controller). |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the deployment runs in. Your Fargate profile must select it. On a sidecar casting it is optional and pins the sidecar's pour to that namespace. |

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
