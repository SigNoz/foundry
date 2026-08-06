# Infrastructure

An Installation casting deploys SigNoz. An Infrastructure casting provisions what it runs on: the network, the machines, and the disks.

They are separate kinds, forged separately and applied separately. Infrastructure never reads your Installation, and the Installation never reads Infrastructure's state. They find each other through the names and tags Foundry puts on every resource.

## Declaring a substrate

An Infrastructure casting says where to provision, and nothing about what will run there:

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

Foundry knows what a default SigNoz installation needs and sizes a substrate for one. It cannot know where to put them, so a casting states its subnets. See [The document](#the-document).

## The document

Forging settles one document, `resource.yaml`, written to `casting.yaml.lock` under `spec.resource.status`. It has two halves.

The top half is a **declaration**, layered: foundry's baseline, then what the platform decides, then whatever you put in `spec.resource.spec.config.data`, which wins. The vocabulary follows [kOps](https://kops.sigs.k8s.io/).

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

The bottom half, `resources`, is **derived** from the settled declaration, and holds every name and tag the substrate stamps. One entry looks like this, and the rest follow the same shape:

```yaml
resources:
  vpc:
    name: signoz-vpc
    tags:
      Name: signoz-vpc
      foundry.signoz.io/kind: Infrastructure
      foundry.signoz.io/managed-by: foundry
      foundry.signoz.io/name: signoz
      foundry.signoz.io/owner: owned
```

The casting's templates interpolate `resources` and assemble no name or tag of their own. The Installation works the same tags out from the substrate name alone, and a string spelled twice is a filter that matches nothing. Stating `resources` yourself is an error, not an override.

There is one baseline: the shape a default SigNoz installation needs. A substrate that keeps nothing drops the persistent group with `persistent: null`. The merge reads that as a deletion.

### Subnets

Collections are keyed by a reference you choose. The key names the subnet's resources and is what an instance group points at. `private-a` above becomes `signoz-sub-private-a`.

| Field | Meaning |
|---|---|
| `type` | `private` or `public`. Workloads go in private subnets |
| `zone` | The provider's availability zone, verbatim |
| `cidr` | The block carved out of `networkCIDR` |
| `egress` | ID of a NAT gateway this private subnet already routes through. Empty creates one |
| `id` | ID of an existing subnet to adopt. Empty creates one |

**`zone` has no default and no fallback.** Zone letters are not contiguous within a region, and the mapping from letter to physical zone differs per account. Only you can state one. A casting with no subnets fails to forge.

A private subnet with no `egress` needs a public subnet in the same zone to hold its NAT gateway.

### Instance groups

| Field | Meaning |
|---|---|
| `storage` | `persistent` or `ephemeral`. See below |
| `machineType` | Provider machine type for each node |
| `minSize` | Smallest the group may be |
| `maxSize` | Largest the group may grow to |
| `subnets` | Subnet references to place nodes in. Empty means every private subnet |
| `rootVolume` | Boot disk per node: `size` in GB, and `type` |
| `dataVolume` | Disk that outlives the node. Persistent groups only |

Nodes in a pinned group are laid out across the group's subnets in ordinal order. Each node's data volume goes wherever the node does. A disk attaches only to a machine in its own zone.

### Storage classes

| Class | Data | Size | Used by |
|---|---|---|---|
| `persistent` | Each node carries a disk that outlives it | Fixed: `minSize` and `maxSize` must match | ClickHouse, Keeper, PostgreSQL |
| `ephemeral` | Keeps nothing | Scales between the bounds | Collector, MCP, UI |

A persistent node cannot be swapped for another: a component's data is on the disk attached to it. Its bounds are pinned, since every node owns a claimed disk.

The class is the only thing about a group the Installation can select on. Two groups may share a class, and the Installation reaches both.

### Overriding the defaults

Put your own values under `spec.resource.spec.config.data`, keyed the same way. You only state what you are changing; everything else comes from the layers underneath.

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

**If you scale SigNoz, you have to scale the persistent group yourself.** The default is three persistent nodes, which covers one Keeper, the metadata store, and one ClickHouse node. Three Keeper replicas and two ClickHouse shards need more. Infrastructure never reads your Installation and cannot work it out.

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

Infrastructure first. The Installation's lookups return nothing until the substrate exists, and an early apply produces a plan that places nothing.

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
    platform: ecs
    mode: ec2
    flavor: terraform
  infrastructure:
    name: signoz
```

That is the whole binding, and it is required. Without it the Installation has no name to derive a filter from, and forging fails.

Everything else the Installation needs, it looks up from that one name:

| What it needs | How it finds it |
|---|---|
| The cluster | the derived name, `signoz-cls` |
| Subnets to place tasks in | the tags `foundry.signoz.io/name` and `foundry.signoz.io/subnet-type: private` |
| The security group | the derived name, `signoz-sg-task` |
| The VPC for its Cloud Map namespace | the tag `foundry.signoz.io/name` |
| The task and execution roles | the derived names, `signoz-iam-task` and `signoz-iam-exec` |
| Machines to pin stateful tasks to | the tags `foundry.signoz.io/name` and `foundry.signoz.io/storage` |
| Which component owns a disk | the tag `foundry.signoz.io/identities` on the disk |

Each arrives in Terraform as a variable defaulted to what was derived. A one-off change needs no edit to a generated file. Any of them can be stated on the Installation instead, one at a time, for a cluster Foundry did not provision; what is stated becomes the variable's own value and nothing is looked up.

No outputs are wired between the two, and no state file is shared. Foundry generates files and exits. It never calls a cloud API and cannot ask the provider what it created a moment ago. Both sides work the names and tags out the same way, which is why they are derived.

A lookup that matches nothing fails the plan. That is deliberate: the alternative is a plan that succeeds and places tasks nowhere.

## Conventions

Foundry derives every name and every tag from the substrate's name and a closed set of values.

### Names

```
<substrate>-<type>[-<qualifier>...]
```

Broad to narrow: everything belonging to one deployment shares a prefix and sorts together. A qualifier that does not apply is left out, letting a security group and its rules share one form.

| Resource | Type | Qualifiers | Example |
|---|---|---|---|
| Cluster | `cls` | | `signoz-cls` |
| VPC | `vpc` | | `signoz-vpc` |
| Internet gateway | `igw` | | `signoz-igw` |
| Subnet | `sub` | subnet key | `signoz-sub-private-a` |
| Route table | `rt` | subnet key | `signoz-rt-private-a` |
| NAT gateway | `nat` | subnet key | `signoz-nat-private-a` |
| Elastic IP | `eip` | subnet key | `signoz-eip-private-a` |
| Security group | `sg` | role | `signoz-sg-task` |
| Security group rule | `sg` | role, purpose | `signoz-sg-task-intra-cluster` |
| IAM role | `iam` | role | `signoz-iam-exec` |
| IAM role policy | `iam` | role, purpose | `signoz-iam-task-appconfig-read` |
| Instance profile | `prf` | role | `signoz-prf-node` |
| Launch template | `lt` | group key | `signoz-lt-ephemeral` |
| Autoscaling group | `asg` | group key | `signoz-asg-ephemeral` |
| Node | `node` | group key, ordinal | `signoz-node-persistent-0` |
| Volume | `vol` | group key, ordinal | `signoz-vol-persistent-0` |

A NAT gateway and its address are keyed by the **private** subnet they serve, not the public one they sit in. Route tables are per subnet: a private subnet's default route is its own zone's gateway.

### Values

| Axis | Values | Where it comes from | In a tag |
|---|---|---|---|
| Subnet key | yours | `networking.subnets` | not tagged |
| Group key | yours | `instanceGroups` | not tagged |
| Subnet type | private, public | a subnet's `type` | `private`, `public` |
| Storage class | persistent, ephemeral | a group's `storage` | `persistent`, `ephemeral` |
| Role | node, task, exec | the platform | not tagged |
| Purpose | what a rule admits or a policy grants | the platform | not tagged |
| Ordinal | position in a group | derived, zero-based | not tagged |

A key you chose distinguishes one name from another and says nothing the Installation can predict. Everything it filters on is a closed enum, spelled out in full. The string has to be identical on both sides, so nothing reaching a tag is abbreviated.

Length caps belong to the platform. IAM role names cap at 64 characters on AWS, keeping roles among the shortest derivations above. The substrate name is capped at 63.

### Tags

Every tag lives under `foundry.signoz.io/` and is one segment deep. A single filter finds everything Foundry touched in an account.

| Tag | Value | Read by |
|---|---|---|
| `foundry.signoz.io/name` | The substrate's name | The Installation, to find this substrate's resources |
| `foundry.signoz.io/subnet-type` | `private` or `public` | The Installation, to pick which subnets to place a workload in |
| `foundry.signoz.io/storage` | `persistent` or `ephemeral` | The Installation, to pick which nodes a component runs on |
| `foundry.signoz.io/identities` | Which components claim a disk | The Installation, to keep a component on its own data |
| `foundry.signoz.io/owner` | `owned` or `shared` | People, to tell what Foundry may delete |
| `foundry.signoz.io/managed-by` | `foundry` | People |
| `foundry.signoz.io/kind` | The casting Kind that stamped it | People |
| `Name` | The derived name | Cloud consoles, which show this tag by convention |

The first four are how the Installation finds anything and are fixed. The rest describe a resource and are free to change.

Your own tags go in `cloudLabels` and are applied to every resource the substrate provisions. They sit underneath the derived ones. A `cloudLabels` entry cannot rename a tag the Installation matches on.

### Identities

A component that keeps data has an identity, written `<component>-<shard>-<replica>` with zero-based ordinals: `telemetrystore-0-0`, `telemetrykeeper-1`, `metastore-0`. An identity claims a disk and stays with it. A component keeps its data when the machine under it is replaced.

Claims are recorded on the disk in `foundry.signoz.io/identities`, comma-joined and sorted for a stable value between runs.

## Dependencies

### What the substrate contains

```
  vpc
   |
   +-- internet gateway            (only if a public subnet is declared)
   |
   +-- subnet                      (one per declared key)
   |     |
   |     +-- route table           (private: via its own nat)
   |     |                         (public: via igw)
   |     +-- nat gateway + address (private only, in a public subnet
   |                                of the same zone)
   +-- security group

  iam role -> instance profile

  per node of a pinned group:
      instance   --> the group's next subnet, instance profile, security group
      volume     --> that subnet's zone
      attachment --> instance + volume

  per scaling group:
      launch template   --> the group's subnets, security group, instance profile
      autoscaling group --> launch template
```

A node and its disk take their zone from the same subnet; a disk attaches only to a machine in its own zone. Ordinal 0 goes in the group's first subnet, 1 in the second, wrapping around.

Nodes in a pinned group are individual machines, not an autoscaling group. An autoscaler replacing a machine would move a disk out from under whatever component owns it.

### How a component reaches its data

```
  task  --pinned to-->  instance  --currently holds-->  volume  --claimed by-->  identity
```

Read it right to left. An identity such as `telemetrystore-0-0` claims a disk. The disk is attached to some machine. The task is pinned to that machine and starts on top of its own data.

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

Set `networking.networkID` to a network you already run, and give every subnet its own `id`:

```yaml
networking:
  networkID: vpc-0a1b2c3d
  subnets:
    private-a:
      type: private
      zone: us-east-1a
      id: subnet-0a1b2c3d
```

Foundry then references the network instead of describing it. It creates no VPC, no internet gateway, no route tables and no NAT gateways, stamps no tags on anything it did not create, and provisions only the compute placed inside. Routing an adopted subnet would replace whatever you attached to it.

A network is adopted whole. Half of one would leave Foundry routing subnets it did not create, or carving subnets out of address space it cannot see.

A private subnet that already has a way out can keep it without adopting the whole network: set `egress` to the NAT gateway it routes through, and Foundry creates none.

Persistent components are the exception. They need disks discovered by tag, which have to come from an Infrastructure casting.

## Limits

**The Installation cannot name an instance group.** You can declare two persistent groups, but the Installation selects by storage class and anything scheduled lands on either one. There is no way to put Keeper on cheaper machines than ClickHouse.

**More stateful components than persistent nodes.** Two identities end up on one disk. Both components run, both write to the same volume, and it looks like replication without being replication. Count one persistent node per identity: one per Keeper replica, one per ClickHouse node, one for the metadata store.

**A claimed disk attached to nothing.** The task stays pending rather than starting empty somewhere else, which is deliberate: starting empty would look like it worked.
