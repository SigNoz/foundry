# ECS EC2 Infrastructure

| Field | Value |
| --- | --- |
| **Kind** | `Infrastructure` |
| **Platform** | `aws` |
| **Mode** | `ecs` |
| **Flavor** | `terraform` |

## Overview

The Infrastructure casting provisions the substrate a SigNoz Installation runs on in ECS on EC2: the network, the cluster, the machines and their disks. `foundryctl` generates the Terraform and applies it.

It provisions:

- A VPC with the subnets you declare. Public subnets route through an internet gateway. Each private subnet routes through its own NAT gateway, which sits in a public subnet of the same zone. Every subnet gets its own route table.
- An ECS cluster
- The `persistent` instance group: one pinned EC2 instance per node, each with its own EBS data volume attached at `/dev/xvdf` and mounted at `/var/lib/foundry`. The volume outlives the instance, so a replaced node keeps its data.
- The `ephemeral` instance group: a launch template and an autoscaling group whose nodes keep nothing
- The node IAM role and instance profile, with the `AmazonEC2ContainerServiceforEC2Role` managed policy the ECS agent needs to register an instance
- One security group for the tasks and the container instances, allowing all traffic between its members and all outbound traffic

Every node boots the ECS-optimized Amazon Linux 2023 AMI, registers with the cluster, and advertises its group's selector as ECS instance attributes: `foundry.signoz.io/name` and `foundry.signoz.io/storage`. A consuming Installation's placement constraints filter on these, so each component lands on the group whose storage it needs. A persistent node registers with the cluster only once its data volume is mounted.

The task and execution roles belong to the Installation that runs the tasks. See [Binding an Installation](../../../../../concepts/infrastructure.md#binding-an-installation).

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- AWS credentials in the environment Terraform runs in, with permission to create VPCs, subnets, internet and NAT gateways, Elastic IPs and route tables, EC2 instances, launch templates, autoscaling groups, security groups and EBS volumes, IAM roles and instance profiles, and ECS clusters, and to read the public SSM parameter that names the ECS-optimized AMI

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

`metadata.name` names everything provisioned, and is the name an Installation binds to.

`foundry.signoz.io/ecs-region` is the AWS region the substrate is provisioned in, and is required. The declared subnet zones must belong to it. `terraform plan` rejects an empty or malformed region.

`resource.yaml` declares what to provision. `networking` sets the VPC's CIDR block and its subnets, keyed by a reference of your choosing. Each subnet states its `type` (`private` or `public`), its `zone` and its `cidr`. A zone is specific to your account, so it has no default.

`instanceGroups` is keyed the same way. The `persistent` and `ephemeral` groups come from the baseline, so the casting above provisions them without naming them:

| Group | Nodes | Machine type | Root volume | Data volume |
| --- | --- | --- | --- | --- |
| `persistent` | 3, pinned | `m5.large` | 30 GB `gp3` | 50 GB `gp3` per node |
| `ephemeral` | 1, in an autoscaling group | `c5.large` | 30 GB `gp3` | none |

State any field under `instanceGroups.persistent` or `instanceGroups.ephemeral` to change it. A new key adds a group, and states every field itself, `storage` (`persistent` or `ephemeral`) included. A persistent group's `minSize` and `maxSize` must match, because each node is its own instance with its own disk. Groups are placed in every private subnet unless they state `subnets`. The full field reference is in [Infrastructure](../../../../../concepts/infrastructure.md#the-document).

## Deploy

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
foundryctl cast -f casting.yaml
```

`gauge` checks that Terraform is installed and usable. `forge` writes the Terraform root under `pours/infrastructure/` and the resolved casting to `casting.yaml.lock`. `cast` runs `terraform init`, `terraform plan -out=tfplan` and `terraform apply tfplan` in that root.

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

Terraform keeps `terraform.tfstate` in that directory. Keep it: `terraform destroy` needs it to know what to remove. Forging into a different location leaves the state behind, so carry the file across, or hold the state remotely by patching `backend.tf.json`.

## After deployment

```bash
# The cluster's name and the rest of the outputs
terraform -chdir=pours/infrastructure output

# The cluster, with its registered container instance count
aws ecs describe-clusters --clusters foundry-cls --region us-east-1

# One container instance per node: three persistent, one ephemeral
aws ecs list-container-instances --cluster foundry-cls --region us-east-1

# The persistent nodes alone, selected the way an Installation places onto them
aws ecs list-container-instances --cluster foundry-cls --region us-east-1 \
  --filter "attribute:foundry.signoz.io/storage == persistent"
```

A persistent node registers a few minutes after it starts, once Terraform has attached its data volume and the node has mounted it.

To change the substrate, edit `casting.yaml`, forge again, and cast. A changed machine type replaces the persistent nodes it applies to, and each data volume moves to its node's replacement.

## Teardown

```bash
cd pours/infrastructure && terraform destroy
```

This removes everything the casting created, the persistent data volumes and the data on them included. Remove the Installation that runs on the substrate first.

## Customization

Override any part of the resource document through `spec.resource.spec.config.data`. Your keys win over the baseline and the casting's own values. For example, to run larger persistent nodes with bigger disks:

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
                machineType: m5.xlarge
                dataVolume:
                  size: 200
```

For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region the substrate is provisioned in; the declared subnet zones must belong to it. Required. |
