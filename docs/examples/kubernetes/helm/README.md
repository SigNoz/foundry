# Kubernetes with Helm

| Field | Value |
| --- | --- |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |
| **Platform** | `-` |

## Overview

Generates a `values.yaml` for the SigNoz Helm chart. Foundry translates the casting spec into Helm values so you can deploy with a single `helm install`.

## Prerequisites

- Kubernetes cluster (1.24+)
- Helm 3.x
- SigNoz Helm chart repo (`https://charts.signoz.io`)

## Configuration

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
  telemetrykeeper:
    kind: zookeeper
```

`telemetrykeeper.kind: zookeeper` is required. The chart ships ZooKeeper as its only coordination service, and an unstated kind resolves to `clickhousekeeper`, which Foundry refuses rather than deploying ZooKeeper in its place.

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Generate values.yaml
foundryctl forge -f casting.yaml

# Add the SigNoz Helm repo
helm repo add signoz https://charts.signoz.io

# Install with Helm
helm install signoz signoz/signoz -f pours/deployment/values.yaml -n signoz --create-namespace
```

> [!NOTE]
> `foundryctl cast` is idempotent. It detects whether a Helm release already exists and runs `helm upgrade` instead of `helm install` accordingly.

## Generated output

```text
pours/deployment/
  values.yaml
  values.mcp.yaml   # only when spec.mcp.spec.enabled is true
```

## After deployment

```bash
# Check pod status
kubectl get pods -n signoz

# Port-forward the SigNoz UI
kubectl port-forward svc/signoz -n signoz 8080:8080
```

Open `http://localhost:8080` to access the SigNoz UI.

## MCP server (optional)

Foundry can deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) alongside the stack so AI clients can query your telemetry. It is disabled by default; enable it in the casting:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
  telemetrykeeper:
    kind: zookeeper
  mcp:
    spec:
      enabled: true
```

This adds a second release, `signoz-mcp-server`, with a service account and a ClusterIP service on port `8000`; the in-cluster endpoint is `http://signoz-mcp-server.signoz:8000/mcp`. Expose it outside the cluster per your setup.

```bash
kubectl -n signoz get pods -l app.kubernetes.io/instance=signoz-mcp-server
```

Clients send their SigNoz API key with each request in the `SIGNOZ-API-KEY` header. To have the server hold a key instead, keep it in a Secret and reference it from `values.mcp.yaml` with a [patch](../../../concepts/patches.md). Create the namespace and the Secret before casting:

```bash
kubectl create namespace signoz
kubectl -n signoz create secret generic signoz-mcp-server --from-literal=SIGNOZ_API_KEY=<your-key>
```

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
  telemetrykeeper:
    kind: zookeeper
  mcp:
    spec:
      enabled: true
  patches:
    - target: "deployment/values.mcp.yaml"
      operations:
        - op: add
          path: /env/-
          value:
            name: SIGNOZ_API_KEY
            valueFrom:
              secretKeyRef:
                name: signoz-mcp-server
                key: SIGNOZ_API_KEY
```

Anyone who can reach the `signoz-mcp-server` service can then query SigNoz with that key. For the authentication modes, see [HTTP Mode](https://github.com/SigNoz/signoz-mcp-server#http-mode).

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

To set resource limits, storage classes, or scheduling constraints on the generated `values.yaml`, use [patches](../../../concepts/patches.md). See the [helm-patches](../helm-patches/) example for a complete working configuration.

## Annotations

Optional annotations to place the release and override the default Helm chart source. These are not required for standard deployments.

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the release is installed into |
| `foundry.signoz.io/kubernetes-helm-chart` | `signoz` | Chart name in the repository, a URL to a chart archive, or a local chart path |
| `foundry.signoz.io/kubernetes-helm-repo-url` | `https://charts.signoz.io` | Chart repository the chart name is resolved against |
| `foundry.signoz.io/kubernetes-helm-chart-version` | `latest` | Helm chart version to install; `latest` installs the repository's latest chart |
| `foundry.signoz.io/kubernetes-helm-mcp-chart` | `signoz-mcp-server` | SigNoz MCP server chart name in the repository, a URL to a chart archive, or a local chart path |
| `foundry.signoz.io/kubernetes-helm-mcp-repo-url` | `https://charts.signoz.io` | Chart repository the SigNoz MCP server chart name is resolved against |
| `foundry.signoz.io/kubernetes-helm-mcp-chart-version` | `latest` | SigNoz MCP server chart version to install; `latest` installs the repository's latest chart |

Example with a custom chart repo:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-helm-repo-url: https://my-registry.example.com/charts
    foundry.signoz.io/kubernetes-helm-chart: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
```
