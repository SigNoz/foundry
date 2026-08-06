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

The deployment picks the casting that provisions. Everything below is the same whichever one you pick.

Foundry knows what a default SigNoz installation needs and sizes a substrate for one. It cannot know where to put them, so a casting states its subnets. See [The document](#the-document).

## The document

Forging settles one document, `resource.yaml`, written to `casting.yaml.lock` under `spec.resource.status`. It has two halves.

The top half is a **declaration**, layered: Foundry's baseline, then what the platform decides, then whatever you put in `spec.resource.spec.config.data`, which wins. The vocabulary follows [kOps](https://kops.sigs.k8s.io/).

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

Zones, machine types and volume types are the provider's own vocabulary, passed through verbatim. Foundry does not translate them.

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
| `egress` | ID of a gateway this private subnet already routes out through. Empty creates one |
| `id` | ID of an existing subnet to adopt. Empty creates one |

**`zone` has no default and no fallback.** Zone letters are not contiguous within a region, and the mapping from letter to physical zone differs per account. Only you can state one. A casting with no subnets fails to forge.

A private subnet with no `egress` needs a public subnet in the same zone to hold its gateway.

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
        | apply                                  | apply
        v                                        v
  +------------------------------------------------------------------+
  |                          the provider                            |
  |    network, machines, disks, tagged as Foundry names them        |
  +------------------------------------------------------------------+
        stamps tags  --------------------->  reads them back
```

Infrastructure first. The Installation's lookups return nothing until the substrate exists, and an early apply produces a plan that places nothing.

The two runs keep separate state. Neither reads the other's, and Foundry passes nothing between them.

## The channel between castings

Your Installation names the substrate it runs on:

```yaml
spec:
  infrastructure:
    name: signoz
```

That is the whole binding, and it is required. Without it the Installation has no name to derive a filter from, and forging fails.

Everything else follows from that one name. A resource Foundry names is found by its derived name; a resource selected out of a set is found by tag:

| What the Installation needs | How it finds it |
|---|---|
| Something Foundry named | the same derivation, run again |
| Which subnets to place a workload in | `foundry.signoz.io/name` and `foundry.signoz.io/subnet-type` |
| Which machines a component runs on | `foundry.signoz.io/name` and `foundry.signoz.io/storage` |
| Which component owns a disk | `foundry.signoz.io/identities` on the disk |

Each arrives in the generated stack as a variable defaulted to what was derived. A one-off change needs no edit to a generated file. Any of them can be stated on the Installation instead, one at a time, for a substrate Foundry did not provision; what is stated becomes the variable's own value and nothing is looked up.

No outputs are wired between the two, and no state is shared. Foundry generates files and exits. It never calls a cloud API and cannot ask the provider what it created a moment ago. Both sides work the names and tags out the same way, which is why they are derived.

A lookup that matches nothing fails the plan. That is deliberate: the alternative is a plan that succeeds and places workloads nowhere.

## Conventions

Foundry derives every name and every tag from the substrate's name and a closed set of values.

### Names

```
<substrate>-<type>[-<qualifier>...]
```

Broad to narrow: everything belonging to one deployment shares a prefix and sorts together. A qualifier that does not apply is left out, letting a resource and its children share one form.

The type token and the set of resources are the provider casting's, since only it knows what it provisions. The qualifiers are not:

| Qualifier | Where it comes from |
|---|---|
| Subnet key | `networking.subnets` |
| Group key | `instanceGroups` |
| Ordinal | position in a group, zero-based |
| Role | the platform's own set of identities |
| Purpose | what a rule admits or a policy grants |

A subnet keyed `private-a` becomes `signoz-sub-private-a`. The first node of a group keyed `persistent` becomes `signoz-node-persistent-0`.

Length caps belong to the platform. The substrate name is capped at 63 characters, which leaves room under the shortest cap a provider imposes on a resource name.

### Values

| Axis | Values | Where it comes from | In a tag |
|---|---|---|---|
| Subnet key | yours | `networking.subnets` | not tagged |
| Group key | yours | `instanceGroups` | not tagged |
| Subnet type | private, public | a subnet's `type` | `private`, `public` |
| Storage class | persistent, ephemeral | a group's `storage` | `persistent`, `ephemeral` |
| Role | the platform's | the platform | not tagged |
| Purpose | the platform's | the platform | not tagged |
| Ordinal | position in a group | derived, zero-based | not tagged |

A key you chose distinguishes one name from another and says nothing the Installation can predict. Everything it filters on is a closed enum, spelled out in full. The string has to be identical on both sides, so nothing reaching a tag is abbreviated.

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
| `Name` | The derived name | Cloud consoles, which show a display name by convention |

The first four are how the Installation finds anything and are fixed. The rest describe a resource and are free to change.

The tag key's spelling is the provider's. A label key that rejects a dot or a slash is rendered in whatever grammar that provider accepts, and both castings render it the same way.

Your own tags go in `cloudLabels` and are applied to every resource the substrate provisions. They sit underneath the derived ones. A `cloudLabels` entry cannot rename a tag the Installation matches on.

## Claims

### How a component reaches its data

```
  workload  --pinned to-->  machine  --currently holds-->  disk  --claimed by-->  identity
```

Read it right to left. An identity such as `telemetrystore-0-0` claims a disk. The disk is attached to some machine. The workload is pinned to that machine and starts on top of its own data.

The claim is recorded on the **disk**, not the machine. Machines get replaced routinely, by a resize, an image update, or a failure. A claim written on the machine would be lost every time one was replaced. Written on the disk it survives, and each plan works out which machine currently holds it.

Nodes in a persistent group are individual machines, never members of a scaling group. An autoscaler replacing a machine would move a disk out from under whatever component owns it.

### Effects of a change

| Change | What follows |
|---|---|
| Add a ClickHouse replica | A new identity appears and claims a free disk. Its workload is pinned to whichever machine holds that disk. |
| Resize a machine | The machine is replaced. Its disk detaches and reattaches, the claim is untouched, and the workload re-pins to the new machine. |
| Change a disk's size | The disk is grown in place. Nothing else moves. |
| Remove a replica | The identity goes away. Its disk keeps the old claim tag until something claims it again. |
| Destroy a disk | The data and the claim are both gone. The identity claims a different disk and starts empty. |

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

Foundry then references the network instead of describing it. It creates none of the networking the casting would otherwise create, stamps no tags on anything it did not create, and provisions only the compute placed inside. Routing an adopted subnet would replace whatever you attached to it.

A network is adopted whole. Half of one would leave Foundry routing subnets it did not create, or carving subnets out of address space it cannot see.

A private subnet that already has a way out can keep it without adopting the whole network: set `egress` to the gateway it routes through, and Foundry creates none.

Persistent components are the exception. They need disks discovered by tag, which have to come from an Infrastructure casting.
