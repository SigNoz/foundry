# EKS Fargate Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `aws` |
| **Mode** | `kubernetes-serverless` |
| **Flavor** | `kustomize` |

## Overview

Deploys one SigNoz Collection Agent on an EKS cluster whose pods run on Fargate, as plain manifests composed by kustomize, exporting to SigNoz Cloud, Self-Hosted Enterprise or Self-Hosted Community.

On the `aws` platform, `mode: kubernetes-serverless` means EKS Fargate.

Fargate schedules no DaemonSet and mounts no host path, and a pod cannot reach the kubelet of the node it runs on. So a single collector runs as a Deployment and reaches every node through the Kubernetes API server instead. Only the `deployment` collector kind is supported.

It collects:

- Pod, container, node and volume metrics for every Fargate node, through the `kubeletstats` receiver. The `k8s_observer` extension discovers the nodes, and `receiver_creator` starts one `kubeletstats` scrape per node, proxied through the API server. The collector's own node is included.
- Cluster metrics (workload status, pod phase, node conditions, allocatables) through the `k8s_cluster` receiver
- Kubernetes events as logs through the `k8s_events` receiver
- OTLP traces, metrics and logs from your applications on `signoz-collector-deployment.<namespace>.svc:4317` (gRPC) and `signoz-collector-deployment.<namespace>.svc:4318` (HTTP)

Kubernetes metadata lands on every signal through the `k8sattributes` processor.

It cannot collect:

- Host metrics. Fargate exposes no host to a pod, so no `hostmetrics` receiver runs and the SigNoz Hosts view stays empty.
- Container stdout and stderr. Fargate mounts no `/var/log/pods`, so no `filelog` receiver runs. See [Logs](#logs) for the Fargate log router route.

The collector runs as one replica with the `Recreate` strategy: a rollout never runs an old and a new pod side by side, so no node is scraped twice.

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- An EKS cluster with a Fargate profile that selects the collector's namespace (`signoz` by default, see [Annotations](#annotations))
- `kubectl` configured against the cluster
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

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

Values under `spec.collector.spec.env` are visible in the Deployment manifest.

### Self-Hosted Community

Set `SIGNOZ_INGESTION_ENDPOINT` to `http://<signoz-host>:4318` and set no key, as in `deployment/casting.yaml`.

## Deploy

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
  kubeconfig.yaml           # in-cluster kubeconfig the kubeletstats scrapes authenticate with
```

The configMapGenerator mounts `deployment.yaml` and `kubeconfig.yaml` at `/conf`. A config change re-hashes the ConfigMap name and rolls the collector on the next apply.

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

4. Annotate the service account with that role through a patch, and add the `awscloudwatch` receiver to the logs pipeline:

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

The Fluent Bit Kubernetes filter records the pod as `kubernetes.pod_id` inside the log record, while `k8sattributes` associates logs by the `k8s.pod.uid` resource attribute, so these logs carry no Kubernetes metadata from the collector unless you lift that field yourself.

Kinesis Data Firehose is an alternative log router output and avoids polling CloudWatch, but Firehose delivers to an HTTP endpoint, so the collector would need an endpoint reachable from AWS.

## When the API-server proxy does not serve kubelet stats

The collector logs errors from the `kubeletstats` scrapes, and the SigNoz pods view stays empty. Read the collector's logs:

```bash
kubectl logs -n <namespace> deployment/signoz-collector-deployment
```

Two errors point here:

- A `404` on `/api/v1/nodes/<node>/proxy/stats/summary`: the kubelet on that node does not expose the summary through the API-server proxy.
- A `403`: the collector's service account is not allowed through the proxy.

Check first that the poured RBAC and environment are in place. The ClusterRole grants `get` on `nodes/stats` and `nodes/proxy`; this must print `yes`:

```bash
kubectl auth can-i get nodes/proxy --as=system:serviceaccount:<namespace>:signoz-collector-deployment
```

The collector container must also carry `KUBECONFIG=/conf/kubeconfig.yaml`, which the pour sets by default.

If both are in place and the scrapes still fail, scrape the kubelets directly instead. The override switches the `kubeletstats` scrape to the service account and the kubelet's own address, and skips the node the collector runs on, since a Fargate pod cannot reach the kubelet of its own node:

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
                    rule: type == "k8s.node" && labels["eks.amazonaws.com/compute-type"] == "fargate" && name != "${env:K8S_NODE_NAME}"
```

Every node is scraped except the one hosting the collector, so the collector's own pod has no pod or container metrics. The `KUBECONFIG` env and the `kubeconfig.yaml` key stay in the pour and go unused.

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

For changes to the generated manifests themselves, use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the collector is deployed into. Your Fargate profile must select it. |

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
