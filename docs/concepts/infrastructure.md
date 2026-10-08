# Infrastructure

An Infrastructure casting provisions what SigNoz runs on: the network, the machines, and the disks. An Installation casting deploys SigNoz onto it.

The two can share one casting file or live in separate ones.

> [!IMPORTANT]
> Cast follows document order, so put the Infrastructure document first in a shared file, or cast the Infrastructure file before the Installation's.

## The casting

Each `platform`, `mode` and `flavor` combination has its own casting.

For example, ECS:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: aws
    mode: ecs
    flavor: terraform
```

> [!NOTE]
> This shows only the shape of the casting, and forging it fails because it states no subnets; start from the full, forgeable file in [the ECS example](../examples/infrastructure/aws/ecs/terraform/README.md).

| Field | Meaning |
|---|---|
| `metadata.name` | Names everything provisioned. Up to 63 characters, lowercase alphanumeric with interior hyphens |
| `metadata.annotations` | Annotations a casting needs, such as `foundry.signoz.io/ecs-region` for ECS; see the [casting file reference](../reference/casting-file.md#ecs-annotations) |
| `spec.deployment` | Which casting provisions. Each `platform`, `mode` and `flavor` combination has its own |
| `spec.resource` | What to provision. See [The document](#the-document) |
| `spec.patches` | RFC 6902 patches applied to the generated files. See [Patches](patches.md) |

Forging writes the generated files to `pours/infrastructure/` and the resolved casting to `casting.yaml.lock`.

## The document

`resource.yaml` says what to provision. Foundry starts from defaults, the casting fills in what the platform decides, and anything you put in `spec.resource.spec.config.data` wins. The defaults are small machines, so the infrastructure boots in any region at little cost. State production sizing in the casting, as [the ECS example](../examples/infrastructure/aws/ecs/terraform/README.md#sizing) shows. The field names follow [kOps](https://kops.sigs.k8s.io/).

Subnets have no default: you always state them, because a zone belongs to your account. The ECS defaults:

```yaml
networking:
  networkCIDR: 10.0.0.0/16
instanceGroups:
  persistent:
    storage: persistent
    machineType: c5.large
    minSize: 3
    maxSize: 3
    rootVolume:
      size: 30
      type: gp3
    dataVolume:
      size: 50
      type: gp3
  ephemeral:
    storage: ephemeral
    machineType: t3.medium
    minSize: 1
    maxSize: 1
    rootVolume:
      size: 30
      type: gp3
```

Zones, machine types and volume types are the provider's own words, passed through as written.

### Networking

| Field | Meaning |
|---|---|
| `networkCIDR` | The block every subnet is carved out of |
| `subnets` | Subnets, keyed by a name you choose |

The subnet's key names its resources and is what an instance group points at. `private-a` becomes `signoz-sub-private-a`.

| Subnet field | Meaning |
|---|---|
| `type` | `private` or `public`. Workloads go in private subnets |
| `zone` | The provider's availability zone, as written |
| `cidr` | The block carved out of `networkCIDR` |
| `egress` | ID of a gateway this private subnet already routes out through. Empty creates one |

A private subnet that already has a way out can keep it: set `egress` to the gateway it routes through.

Forging fails unless:

- Every subnet states a `zone`.
- There is at least one subnet, and at least one of them is private.
- Every private subnet without an `egress` has a public subnet in the same zone.

For example, ECS:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
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
              networkCIDR: 10.0.0.0/16           # every subnet comes from it
              subnets:
                private-a:
                  type: private                  # workloads run here
                  zone: us-east-1a               # a zone of the stated region
                  cidr: 10.0.0.0/19              # inside networkCIDR
                private-b:
                  type: private
                  zone: us-east-1b
                  cidr: 10.0.32.0/19
                  egress: nat-0a1b2c3d4e5f67890  # an existing NAT gateway
                public-a:
                  type: public                   # the zone's NAT gateway sits here
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
                public-b:
                  type: public
                  zone: us-east-1b
                  cidr: 10.0.100.0/22
```

### Instance groups

| Field | Meaning |
|---|---|
| `storage` | `persistent` or `ephemeral`. See below |
| `machineType` | Provider machine type for each node |
| `minSize` | Smallest the group may be |
| `maxSize` | Largest the group may grow to |
| `subnets` | Subnet keys to place nodes in. Empty means every private subnet |
| `rootVolume` | Boot disk per node: `size` in GB, and `type` |
| `dataVolume` | Disk that outlives the node. Persistent groups only |

Nodes are spread across the group's subnets in order, and a node's data volume goes wherever the node does.

For example, ECS:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
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
              subnets:
                private-a:
                  type: private
                  zone: us-east-1a
                  cidr: 10.0.0.0/19
                private-b:
                  type: private
                  zone: us-east-1b
                  cidr: 10.0.32.0/19
                public-a:
                  type: public
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
                public-b:
                  type: public
                  zone: us-east-1b
                  cidr: 10.0.100.0/22
            instanceGroups:
              persistent:
                machineType: c5.xlarge           # a larger machine
                minSize: 6                       # must equal maxSize
                maxSize: 6                       # must equal minSize
                dataVolume:
                  size: 200                      # GB per node
              batch:
                storage: ephemeral               # persistent or ephemeral
                machineType: t3.large            # every field is stated
                minSize: 1                       # smallest the group may be
                maxSize: 3                       # largest it may grow to
                subnets:                         # only these subnets
                  - private-b
                rootVolume:
                  size: 30                       # GB per node
                  type: gp3
```

### Storage classes

| Class | Data | Size | Used by |
|---|---|---|---|
| `persistent` | Each node carries a disk that outlives it | Fixed: `minSize` and `maxSize` must match | ClickHouse, Keeper, PostgreSQL, UI on SQLite |
| `ephemeral` | Keeps nothing | Scales between the bounds | Collector, MCP, UI on PostgreSQL |

The class is the only thing about a group an Installation can select on. Two groups may share a class, and the Installation reaches both.

The [instance groups example](#instance-groups) sets a group's class with `storage`.

### Everything else

| Field | Meaning |
|---|---|
| `iam.permissionsBoundary` | Policy attached as the permissions boundary of every role created |
| `cloudLabels` | Your own tags, added to every resource provisioned. They cannot rename a tag an Installation matches on |

For example, ECS:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
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
              subnets:
                private-a:
                  type: private
                  zone: us-east-1a
                  cidr: 10.0.0.0/19
                public-a:
                  type: public
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
            iam:
              permissionsBoundary: arn:aws:iam::123456789012:policy/boundary  # on every role created
            cloudLabels:
              team: observability                # added to every resource
              cost-center: "1234"                # added to every resource
```

### Changing the defaults

State only what you are changing. To drop the persistent group entirely, set `persistent: null`. For example, ECS:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
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
              subnets:
                private-a:
                  type: private
                  zone: us-east-1a
                  cidr: 10.0.0.0/19
                private-b:                       # a second zone
                  type: private
                  zone: us-east-1b
                  cidr: 10.0.32.0/19
                public-a:
                  type: public
                  zone: us-east-1a
                  cidr: 10.0.96.0/22
                public-b:                        # a public subnet for it
                  type: public
                  zone: us-east-1b
                  cidr: 10.0.100.0/22
            instanceGroups:
              persistent:
                minSize: 6                       # six persistent nodes
                maxSize: 6                       # six persistent nodes
                machineType: c5.xlarge           # a larger machine
              ephemeral:
                minSize: 2                       # scales from two
                maxSize: 4                       # up to four
```

**If you scale SigNoz, scale the persistent group yourself.** Three persistent nodes cover one Keeper, the metadata store, and one ClickHouse node.

## Names

```
<name>-<type>[-<extra>...]
```

Everything provisioned starts with the casting's `metadata.name`, then a short word for what it is, then whatever tells it apart from its siblings: the subnet or group key, a node's position in its group, or what a rule admits.

A subnet keyed `private-a` becomes `signoz-sub-private-a`. The first node of a group keyed `persistent` becomes `signoz-node-persistent-0`.

## Tags

Every resource is tagged. Tags live under `foundry.signoz.io/` and are one segment deep.

| Tag | Value | Read by |
|---|---|---|
| `foundry.signoz.io/name` | The casting's `metadata.name` | An Installation, to find these resources |
| `foundry.signoz.io/subnet-type` | `private` or `public` | An Installation, to pick subnets for a workload |
| `foundry.signoz.io/storage` | `persistent` or `ephemeral` | An Installation, to pick nodes for a component |
| `foundry.signoz.io/identities` | Which components own a disk | An Installation, to keep a component on its own data |
| `foundry.signoz.io/owner` | `owned` or `shared` | People, to tell what Foundry may delete |
| `foundry.signoz.io/managed-by` | `foundry` | People |
| `foundry.signoz.io/kind` | The casting Kind that tagged it | People |
| `Name` | The name above | Cloud consoles, which show it as a display name |

An Installation searches on the first four. The rest describe a resource.

Where a provider rejects a dot or a slash in a tag key, the key is rendered in whatever that provider accepts.

## Binding an Installation

An Installation names the infrastructure it runs on:

```yaml
spec:
  infrastructure:
    name: signoz
```

Everything else follows from that name:

| What the Installation needs | How it finds it |
|---|---|
| Something Foundry named | Builds the same name again |
| Which subnets to place a workload in | `foundry.signoz.io/name` and `foundry.signoz.io/subnet-type` |
| Which machines a component runs on | `foundry.signoz.io/name` and `foundry.signoz.io/storage` |
| Which component owns a disk | `foundry.signoz.io/identities` on the disk |

The Installation's casting resolves what it needs through the names and tags above. What it looks up is in that casting's documentation: see [the ECS example](../examples/ecs/ec2/terraform/README.md).

A search that matches nothing fails the plan.

## Disks

A persistent node's disk outlives the machine it is attached to, so a component keeps its data when its machine is replaced. The disk is tagged with the components that own it, such as `telemetrystore-clickhouse-0-0`, and the component is placed on whichever machine currently holds it.

| Change | What happens |
|---|---|
| Resize a machine | The machine is replaced, its disk moves to the new one, and the component follows |
| Change a disk's size | The disk is grown in place |
