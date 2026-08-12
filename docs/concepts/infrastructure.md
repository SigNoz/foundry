# Infrastructure

An Infrastructure casting provisions what SigNoz runs on: the network, the machines, and the disks. An Installation casting deploys SigNoz onto it.

The two are separate castings, forged and applied separately. Apply Infrastructure first.

## The casting

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
```

| Field | Meaning |
|---|---|
| `metadata.name` | Names everything provisioned. Up to 63 characters, lowercase alphanumeric with interior hyphens |
| `spec.deployment` | Which casting provisions. Each `platform`, `mode` and `flavor` combination has its own |
| `spec.resource` | What to provision. See [The document](#the-document) |
| `spec.patches` | RFC 6902 patches applied to the generated files |

Forging writes the generated files to `pours/infrastructure/` and the resolved casting to `casting.yaml.lock`.

## The document

`resource.yaml` says what to provision. Foundry starts from a default sized for a standard SigNoz installation, the casting fills in what the platform decides, and anything you put in `spec.resource.spec.config.data` wins. The field names follow [kOps](https://kops.sigs.k8s.io/).

```yaml
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
    storage: persistent
    machineType: m5.large
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
    machineType: c5.large
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
| `networkID` | ID of an existing network to use. Empty creates one. See [Adopting](#adopting-what-you-already-run) |
| `subnets` | Subnets, keyed by a name you choose |

The subnet's key names its resources and is what an instance group points at. `private-a` becomes `signoz-sub-private-a`.

| Subnet field | Meaning |
|---|---|
| `type` | `private` or `public`. Workloads go in private subnets |
| `zone` | The provider's availability zone, as written |
| `cidr` | The block carved out of `networkCIDR` |
| `egress` | ID of a gateway this private subnet already routes out through. Empty creates one |
| `id` | ID of an existing subnet to use. Empty creates one |

Forging fails unless:

- Every subnet states a `zone`. There is no default.
- There is at least one subnet, and at least one of them is private.
- Every private subnet without an `egress` has a public subnet in the same zone.
- Every subnet states its own `id` when `networkID` is set.

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

Nodes are laid out across the group's subnets in order, and a node's data volume goes wherever the node does.

### Storage classes

| Class | Data | Size | Used by |
|---|---|---|---|
| `persistent` | Each node carries a disk that outlives it | Fixed: `minSize` and `maxSize` must match | ClickHouse, Keeper, PostgreSQL |
| `ephemeral` | Keeps nothing | Scales between the bounds | Collector, MCP, UI |

The class is the only thing about a group an Installation can select on. Two groups may share a class, and the Installation reaches both.

### Everything else

| Field | Meaning |
|---|---|
| `iam.permissionsBoundary` | Policy ARN attached as the permissions boundary of every role created |
| `cloudLabels` | Your own tags, added to every resource provisioned. They cannot rename a tag an Installation matches on |

### Changing the defaults

State only what you are changing. To drop the persistent group entirely, set `persistent: null`.

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: ec2
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
                minSize: 6
                maxSize: 6
                machineType: m5.xlarge
              ephemeral:
                minSize: 2
                maxSize: 4
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

The first four are what an Installation searches on. The rest describe a resource.

Where a provider rejects a dot or a slash in a tag key, it is rendered in whatever that provider accepts.

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

Each one becomes a variable in the generated files, defaulted to what was worked out. For infrastructure Foundry did not provision, set the variable instead and nothing is looked up.

A search that matches nothing fails the plan.

## Disks

A persistent node's disk outlives the machine it is attached to, so a component keeps its data when its machine is replaced. The disk is tagged with the components that own it, such as `telemetrystore-0-0`, and the component is placed on whichever machine currently holds it.

| Change | What happens |
|---|---|
| Add a ClickHouse replica | It takes a free disk, and runs on the machine holding it |
| Resize a machine | The machine is replaced, its disk moves to the new one, and the component follows |
| Change a disk's size | The disk is grown in place |
| Remove a replica | Its disk keeps the tag until something else takes it |
| Destroy a disk | The data is gone, and the replica starts empty on another disk |

## Adopting what you already run

Set `networking.networkID` to a network you already run, and give every subnet its own `id`:

```yaml
networking:
  networkID: <existing network>
  subnets:
    private-a:
      type: private
      zone: us-east-1a
      id: <existing subnet>
```

Foundry then creates none of the networking, tags nothing it did not create, and provisions only the machines and disks inside.

A private subnet that already has a way out can keep it without adopting the whole network: set `egress` to the gateway it routes through.

Persistent components still need an Infrastructure casting. Their disks are found by tag.

## Where things live

| Package | What it holds |
|---|---|
| `api/v1alpha1/infrastructure` | `Casting`, `Spec`, `Resource` and `ResourceConfig`, plus the generated schema |
| `internal/config/yamlconfig` | Loads each document in a file by its `kind` |
| `internal/molding/infrastructure` | The `Molding` and `MoldingEnricher` contracts for this kind |
| `internal/molding/infrastructure/resourcemolding` | Settles and validates `resource.yaml` |
| `internal/casting/infrastructure` | The `Casting` contract, the planner, and the registry |
| `internal/contract` | Substrate, keys, identities, selections, tag keys, storage classes, subnet types |
| `internal/contract/aws` | The descriptor, and the AWS name and tag grammar |
| `internal/contract/aws/ecs`, `internal/contract/aws/eks` | `Derive` per mode |
