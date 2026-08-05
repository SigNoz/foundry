# Infrastructure

An Installation casting deploys SigNoz. An Infrastructure casting provisions what it runs on: the network, the machines, and the disks.

They are separate kinds, forged separately and applied separately. Infrastructure never reads your Installation, and the Installation never reads Infrastructure's state. They find each other through the names and tags Foundry puts on every resource.

## Declaring a substrate

An Infrastructure casting says which Kind it is provisioning for, and nothing about that Kind's internals:

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: ec2
    flavor: terraform
  resource:
    kind: Installation
```

`spec.resource.kind` is either `Installation` or `CollectionAgent`. That single field is the whole input: Foundry knows what a default SigNoz installation needs, so it can size a substrate for one without being told anything about your components.

## Requirements

Forging turns that declaration into a requirement document, written to `casting.yaml.lock` under `spec.resource.status`. For an `Installation`:

```yaml
nodeGroups:
  ephemeral:
    cpu: 2
    maxSize: 1
    memory: 4
    minSize: 1
    rootVolume:
      size: 30
  persistent:
    cpu: 2
    dataVolume:
      size: 50
    maxSize: 3
    memory: 8
    minSize: 3
    rootVolume:
      size: 30
```

Plus the ports the substrate has to admit at its edge: `4317` and `4318` for OTLP, and `8080` for the API server. A `CollectionAgent` gets the ephemeral group and the OTLP ports only, because it stores nothing.

Capacity is stated as CPU and memory rather than as `m5.large`, so the same document works on any provider. The casting resolves it to a real machine type at plan time. Set `machineType` yourself when you want an exact one.

### Node group fields

| Field | Meaning |
|---|---|
| `minSize` | Smallest the group may be |
| `maxSize` | Largest the group may grow to |
| `cpu` | CPUs per node |
| `memory` | Memory per node, in GB |
| `machineType` | Provider machine type; when set, `cpu` and `memory` are ignored |
| `rootVolume.size` | Boot disk per node, in GB |
| `dataVolume.size` | Disk that outlives the node, in GB. Persistent groups only |

### Storage classes

Node groups are keyed by storage class, and the class decides how the group behaves:

| Class | Data | Size | Used by |
|---|---|---|---|
| `persistent` | Each node carries a disk that outlives it | Fixed: `minSize` and `maxSize` must match | ClickHouse, Keeper, PostgreSQL |
| `ephemeral` | Keeps nothing | Scales between the bounds | Collector, MCP, UI |

A persistent node cannot be swapped for another, because a component's data is on the disk attached to it. That is why its bounds are pinned: there is nothing to autoscale when every node owns a claimed disk.

There is one group per class. Two persistent groups would be indistinguishable to the Installation, which selects nodes by class, so anything scheduled would land on either one at random.

### Overriding the defaults

Put your own values under `spec.resource.spec.config.data`, keyed by class. You only state what you are changing; everything else comes from the defaults above.

```yaml
apiVersion: v1alpha1
kind: Infrastructure
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: ec2
    flavor: terraform
  resource:
    kind: Installation
    spec:
      config:
        data:
          resource.yaml: |
            nodeGroups:
              persistent:
                minSize: 6
                maxSize: 6
                machineType: m5.xlarge
              ephemeral:
                minSize: 2
                maxSize: 4
```

**If you scale SigNoz, you have to scale the persistent group yourself.** The default is three persistent nodes, which covers one Keeper, the metadata store, and one ClickHouse node. Three Keeper replicas and two ClickHouse shards need more, and Infrastructure cannot work that out for you because it never reads your Installation.

## Order of operations

```
  casting.yaml  (Infrastructure)          casting.yaml  (Installation)
        |                                        |
        | forge                                  | forge
        v                                        v
  pours/infrastructure/                    pours/deployment/
        |                                        |
        | terraform apply                        | terraform apply
        v                                        v
  +------------------------------------------------------------------+
  |                          the provider                            |
  |    network, machines, disks, tagged as Foundry names them        |
  +------------------------------------------------------------------+
        stamps tags  --------------------->  reads them back
```

Infrastructure first. The Installation's lookups return nothing until the substrate exists, so applying it early produces a plan that places nothing.

The two runs keep separate Terraform state. Neither reads the other's, and Foundry passes nothing between them.

## The channel between castings

Your Installation names the substrate it runs on:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    platform: aws
    mode: ec2
    flavor: terraform
  infrastructure:
    name: signoz
```

That is the whole binding. Everything else travels as tags on the resources themselves: the Installation searches for `foundry.signoz.io/name` and `foundry.signoz.io/storage` to find machines and disks, and reads `foundry.signoz.io/identities` off a disk to learn which component owns it.

No outputs are wired between the two, and no state file is shared. Foundry generates files and exits; it never calls a cloud API, so it cannot ask the provider what it created a moment ago. Both sides have to work the names and tags out the same way, which is why they are derived rather than configured.

## Conventions

Foundry derives every name and every tag from the substrate's name and a closed set of values.

### Names

```
<substrate>-<type>[-<qualifier>...]
```

Broad to narrow, so everything belonging to one deployment shares a prefix and sorts together. A qualifier that does not apply is left out rather than padded, which is why a zone-shared route table has no zone in its name.

| Resource | Type | Qualifiers | Example |
|---|---|---|---|
| Cluster | `cls` | | `signoz-cls` |
| VPC | `vpc` | | `signoz-vpc` |
| Internet gateway | `igw` | | `signoz-igw` |
| Subnet | `sub` | visibility, zone | `signoz-sub-prv-east1a` |
| Route table, per zone | `rt` | visibility, zone | `signoz-rt-prv-east1a` |
| Route table, zone-shared | `rt` | visibility | `signoz-rt-pub` |
| NAT gateway | `nat` | zone | `signoz-nat-east1a` |
| Security group | `sg` | role | `signoz-sg-task` |
| IAM role | `iam` | role | `signoz-iam-exec` |
| Node | `node` | storage class, ordinal | `signoz-node-persistent-0` |
| Volume | `vol` | storage class, ordinal | `signoz-vol-persistent-0` |

### Values

| Axis | Values | In a name | In a tag |
|---|---|---|---|
| Visibility | private, public | `prv`, `pub` | `private`, `public` |
| Storage class | persistent, ephemeral | `persistent`, `ephemeral` | same |
| Role | node, task, exec | `node`, `task`, `exec` | not tagged |
| Zone | the provider's zone | locale dropped: `us-east-1a` becomes `east1a` | provider's own form |
| Ordinal | position in a group | zero-based: `0`, `1`, `2` | not tagged |

A value the Installation matches on is never abbreviated, because the string has to be identical on both sides. A value only a person reads can be short where space is tight, which is why visibility has two forms and the storage class has one.

Length caps belong to the platform. IAM role names cap at 64 characters on AWS, which is why roles are the shortest derivation above; the substrate name is capped at 63.

### Tags

Every tag lives under `foundry.signoz.io/` and is one segment deep, so a single filter finds everything Foundry touched in an account.

| Tag | Value | Read by |
|---|---|---|
| `foundry.signoz.io/name` | The substrate's name | The Installation, to find this substrate's resources |
| `foundry.signoz.io/storage` | `persistent` or `ephemeral` | The Installation, to pick which nodes a component runs on |
| `foundry.signoz.io/identities` | Which components claim a disk | The Installation, to keep a component on its own data |
| `foundry.signoz.io/resource-kind` | The Kind the substrate serves | People |
| `foundry.signoz.io/owner` | `owned` or `shared` | People, to tell what Foundry may delete |
| `foundry.signoz.io/visibility` | `private` or `public` | People |
| `Name` | The derived name | Cloud consoles, which show this tag by convention |

The first three are how the Installation finds anything, so they are fixed. The rest describe a resource and are free to change.

### Identities

A component that keeps data has an identity, written `<component>-<shard>-<replica>` with zero-based ordinals: `telemetrystore-0-0`, `telemetrykeeper-1`, `metastore-0`. An identity claims a disk and stays with it, so a component keeps its data when the machine under it is replaced.

Claims are recorded on the disk in `foundry.signoz.io/identities`, comma-joined and sorted, so the value is stable between runs.

## Dependencies

### What the substrate contains

```
  vpc
   |
   +-- internet gateway
   |
   +-- subnet (one per zone, private and public)
   |     |
   |     +-- nat gateway            (public subnet, one per zone)
   |     +-- route table            (private: per zone, via nat)
   |                                (public: shared, via igw)
   +-- security group

  iam role -- instance profile

  per persistent ordinal:
      instance   --> subnet in zone N, instance profile, security group
      volume     --> zone N
      attachment --> instance + volume

  per ephemeral group:
      launch template   --> subnets, security group, instance profile
      autoscaling group --> launch template
```

A persistent node and its disk are placed in the same zone, because a disk can only attach to a machine in its own zone. Ordinal 0 goes in the first zone, 1 in the second, and so on.

Persistent nodes are individual machines rather than an autoscaling group. An autoscaler replacing a machine would move a disk out from under whatever component owns it.

### How a component reaches its data

```
  task  --pinned to-->  instance  --currently holds-->  volume  --claimed by-->  identity
```

Read it right to left. An identity such as `telemetrystore-0-0` claims a disk. The disk is attached to some machine. The task is pinned to that machine, so it starts on top of its own data.

The claim is recorded on the **disk**, not the machine. Machines get replaced routinely, by a resize, an image update, or a failure. A claim written on the machine would be lost every time one was replaced. Written on the disk it survives, and each plan works out which machine currently holds it.

### Effects of a change

| Change | What follows |
|---|---|
| Add a ClickHouse replica | A new identity appears and claims a free disk. Its task is pinned to whichever machine holds that disk. |
| Resize a machine | The machine is replaced. Its disk detaches and reattaches, the claim is untouched, and the task re-pins to the new machine. |
| Change a disk's size | The disk is grown in place. Nothing else moves. |
| Remove a replica | The identity goes away. Its disk keeps the old claim tag until something claims it again. |
| Destroy a disk | The data and the claim are both gone. The identity claims a different disk and starts empty. |

## Adopting resources you already have

A resource tagged `foundry.signoz.io/owner: shared` keeps the name it already had and is never deleted. This is how an existing VPC gets used rather than replaced.

You can point an Installation at a network, subnets and a cluster you already run. Persistent components are the exception: they need disks discovered by tag, so those have to come from an Infrastructure casting.

## Limits

**One node group per storage class.** There is no way to put Keeper on cheaper machines than ClickHouse, because the Installation selects nodes by class and has no vocabulary for naming a group.

**More stateful components than persistent nodes.** Two identities end up on one disk. Both components run, both write to the same volume, and it looks like replication without being replication. Count one persistent node per identity: one per Keeper replica, one per ClickHouse node, one for the metadata store.

**A claimed disk attached to nothing.** The task stays pending rather than starting empty somewhere else, which is deliberate: starting empty would look like it worked.