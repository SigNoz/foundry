# AWS Kubernetes with Terraform (Infrastructure)

| Field | Value |
| --- | --- |
| **Kind** | `Infrastructure` |
| **Platform** | `aws` |
| **Mode** | `kubernetes` |
| **Flavor** | `terraform` |

## Overview

Provisions an EKS substrate shaped for a SigNoz Installation. The infrastructure never reads the installation's casting: the resource declaration names what the substrate is shaped for, and foundry's own kind-level knowledge (the requirement set) drives what gets provisioned.

Resources:
- VPC with public and private subnets across two availability zones, internet and NAT gateways
- EKS cluster with IAM roles for the control plane and nodes
- One managed node group sized from the requirement set, in private subnets
- EBS CSI driver addon (the installation's components request storage through PVCs)

## Prerequisites

- AWS credentials with permissions to create VPC, EKS, and IAM resources
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## Configuration

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes
    flavor: terraform
  resource:
    kind: Installation
    spec:
      name: signoz
```

`spec.resource` declares the kind of resource the substrate serves and the name of the casting embodying it. It is a declaration, not a reference: nothing is read from the installation's casting.

## Deploy

```bash
# 1. Generate Terraform files
foundryctl forge -f casting.yaml

# 2. Initialize and apply Terraform
cd pours/infrastructure
terraform init
terraform apply
```

## Generated output

```text
pours/infrastructure/
  providers.tf.json
  main.tf.json
  variables.tf.json
  outputs.tf.json
```

## Customization

To pin an exact instance type instead of resolving the criteria, set the instance type variable through `spec.patches`:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: kubernetes
    flavor: terraform
  resource:
    kind: Installation
    spec:
      name: signoz
  patches:
    - target: "infrastructure/variables.tf.json"
      operations:
        - op: replace
          path: /variable/node_default_instance_type/default
          value: t3.large
```

Any generated value can be changed the same way; run `foundryctl forge` and inspect the files under `pours/infrastructure/` to identify the JSON paths. 

## Platform details

### Variables

| Variable | Default | Description |
| --- | --- | --- |
| `aws_region` | `us-east-1` | AWS region |
| `vpc_cidr` | `10.0.0.0/16` | CIDR block for the VPC |
| `az_count` | `2` | Number of availability zones |
| `name` | `signoz` | Name of the deployment |
| `kubernetes_version` | `1.33` | Kubernetes version for the EKS cluster |
| `node_default_instance_type` | `""` | Instance type; empty resolves the declared criteria against the platform's instance catalog |
| `node_default_count` | `2` | Number of nodes; elastic bounds belong with a cluster autoscaler |
| `node_default_disk_size` | `50` | Root volume size (GB) for the nodes |

### Outputs

| Output | Description |
| --- | --- |
| `cluster_name` | Name of the EKS cluster |
| `cluster_endpoint` | EKS API server endpoint |
| `cluster_ca_certificate` | Base64-encoded CA data (sensitive) |
| `cluster_version` | Kubernetes version of the cluster |
| `vpc_id` | ID of the VPC |
| `private_subnet_ids` | IDs of the private subnets |
| `public_subnet_ids` | IDs of the public subnets |
| `node_group_default_arn` | ARN of the default node group |
| `node_group_default_status` | Status of the default node group |

### Tags

Every resource carries the discovery tags, so consumers can find the substrate by name:

| Tag | Value |
| --- | --- |
| `app.kubernetes.io/managed-by` | `foundry` |
| `foundry.signoz.io/name` | `signoz` |
| `foundry.signoz.io/resource-kind` | `Installation` |
