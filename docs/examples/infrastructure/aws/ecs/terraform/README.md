# ECS EC2 Infrastructure

| Field | Value |
| --- | --- |
| **Kind** | `Infrastructure` |
| **Platform** | `aws` |
| **Mode** | `ecs` |
| **Flavor** | `terraform` |

## Overview

This casting provisions the AWS infrastructure a SigNoz Installation runs on in ECS on EC2. `foundryctl` generates the Terraform and applies it.

It creates:

- A VPC with the subnets you declare, and a NAT gateway for each private subnet
- An ECS cluster named `<metadata.name>-cls`
- The `persistent` instance group: one EC2 instance per node, each with its own EBS data volume mounted at `/var/lib/foundry`. The volume outlives the instance, so a replaced node keeps its data.
- The `ephemeral` instance group: an autoscaling group whose nodes keep nothing
- The IAM role and instance profile the nodes run as
- One security group that allows all traffic between its members and all outbound traffic

How to describe the network, instance groups and disks is in [Infrastructure](../../../../../concepts/infrastructure.md).

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- AWS credentials in the environment Terraform runs in, allowed to create VPC networking, EC2 instances and autoscaling groups, EBS volumes, IAM roles, and ECS clusters, and to read the public SSM parameter that names the ECS-optimized AMI

## Configuration

This directory's `casting.yaml`:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: foundry
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: aws
    mode: ecs
    flavor: terraform
  resource:
    spec:
      config:
        data:
          resource.yaml: |
            networking:
              networkCIDR: 10.0.0.0/16
              subnets:
                private-a:
                  type: private
                  zone: us-east-1a
                  cidr: 10.0.0.0/19
                public-a:
                  type: public
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
```

`metadata.name` prefixes every resource name. An Installation runs on this infrastructure when it names it in `spec.infrastructure.name`.

`foundry.signoz.io/ecs-region` sets the AWS region, and is required. The subnet zones must belong to it.

The casting above provisions two instance groups with these defaults:

| Group | Nodes | Machine type | Root volume | Data volume |
| --- | --- | --- | --- | --- |
| `persistent` | 3 | `c5.large` | 30 GB `gp3` | 50 GB `gp3` per node |
| `ephemeral` | 1, in an autoscaling group | `t3.medium` | 30 GB `gp3` | none |

Every field of `resource.yaml`, with an example per section, is in [Infrastructure](../../../../../concepts/infrastructure.md#the-document).

### Sizing

The defaults are sized for development and boot in any region. Size production infrastructure with the [SigNoz capacity guide](https://signoz.io/docs/setup/capacity-planning/community/resources-planning/). On AWS, the guide names the C5 family, `c5.4xlarge` and up, for ClickHouse, and the T3 family, `t3.xlarge` and up, for everything else.

Production-sized infrastructure, with three nodes in each group and each persistent data volume sized for retention:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: foundry
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: aws
    mode: ecs
    flavor: terraform
  resource:
    spec:
      config:
        data:
          resource.yaml: |
            networking:
              networkCIDR: 10.0.0.0/16
              subnets:
                private-a:
                  type: private
                  zone: us-east-1a
                  cidr: 10.0.0.0/19
                public-a:
                  type: public
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
            instanceGroups:
              persistent:
                machineType: c5.4xlarge
                minSize: 3
                maxSize: 3
                dataVolume:
                  size: 500
              ephemeral:
                machineType: t3.xlarge
                minSize: 3
                maxSize: 3
```

## Deploy

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
foundryctl cast -f casting.yaml
```

`gauge` checks that Terraform is installed and usable. `forge` writes the Terraform under `pours/infrastructure/` and the resolved casting to `casting.yaml.lock`. `cast` runs `terraform init`, `terraform plan -out=tfplan` and `terraform apply tfplan` in `pours/infrastructure/`.

Apply this casting before the Installation that runs on it.

## Generated output

```text
pours/infrastructure/
  versions.tf.json          # required Terraform and provider versions
  backend.tf.json           # local state, in this directory
  providers.tf.json         # the aws provider, in var.aws_region
  variables.tf.json         # every declared value, defaulted to what resource.yaml settled
  main.tf.json              # the network, the instance groups, the node role, the security group and the cluster
  outputs.tf.json           # the cluster, VPC, subnet, security group, node role and instance group identifiers
  cloud-init/
    persistent.yaml         # persistent node boot config: cluster registration, instance attributes, data volume mount
    ephemeral.yaml          # ephemeral node boot config: cluster registration and instance attributes
```

Terraform keeps its state in `pours/infrastructure/terraform.tfstate`. Keep that file, because `terraform destroy` needs it. If you forge into another location, copy the file across, or move the state elsewhere with a patch on `backend.tf.json`.

## Validate

```bash
# The cluster name and the other outputs
terraform -chdir=pours/infrastructure output

# The cluster: 4 registered container instances once every node has registered
aws ecs describe-clusters --clusters foundry-cls --region us-east-1

# One container instance per node: three persistent, one ephemeral
aws ecs list-container-instances --cluster foundry-cls --region us-east-1

# The persistent nodes alone, the ones that hold data
aws ecs list-container-instances --cluster foundry-cls --region us-east-1 \
  --filter "attribute:foundry.signoz.io/storage == persistent"
```

Every node registers with the ECS instance attributes `foundry.signoz.io/name` and `foundry.signoz.io/storage`, which the last command filters on.

A persistent node registers a few minutes after it starts, once its data volume is attached and mounted.

## Changing the infrastructure

Edit `casting.yaml`, then forge and cast again.

A changed machine type replaces the persistent nodes, and each data volume moves to its node's replacement. The ephemeral group rolls through an instance refresh that drains the pool fully, so ingestion pauses, along with every other service on that group, until the new node registers. What happens to the data on each disk is in [Disks](../../../../../concepts/infrastructure.md#disks).

Nodes boot the ECS-optimized Amazon Linux 2023 AMI, read at plan from the public SSM parameter `/aws/service/ecs/optimized-ami/amazon-linux-2023/recommended/image_id`.

> [!NOTE]
> The nodes keep their AMI until you roll them, so roll them with the commands below to pick up a newer one.

To roll the ephemeral group onto the current AMI:

```bash
terraform -chdir=pours/infrastructure apply -replace=aws_launch_template.ephemeral
```

To move a persistent node onto it, replace that node. Its data volume moves to the replacement. Replace persistent nodes one at a time:

```bash
terraform -chdir=pours/infrastructure apply -replace=aws_instance.persistent-0
```

For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md).

## Teardown

```bash
cd pours/infrastructure && terraform destroy
```

This removes everything the casting created, including the persistent data volumes and the data on them. Remove the Installation that runs on it first.

## Annotations

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region the infrastructure is provisioned in; the subnet zones must belong to it. Required. |
