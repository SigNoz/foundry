# Annotations

Annotations are key-value pairs in `metadata.annotations` that provide deployment-specific configuration to Foundry. They are inputs to the generation process - Foundry reads them during forge and uses them to populate templates and configure behavior.

## How annotations work

Annotations live in the casting file's metadata block:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/some-key: some-value
spec:
  deployment:
    mode: systemd
    flavor: binary
```

During forge, Foundry reads annotation values and injects them into the generated output. For example, systemd annotations set binary paths in the generated `.service` files. ECS annotations populate Terraform variables in the generated `terraform.tfvars.json`.

## Annotations vs patches

Annotations and patches serve different roles in the generation pipeline:

| | Annotations | Patches |
|---|---|---|
| **Where** | `metadata.annotations` | `spec.patches` |
| **When** | Before generation - Foundry reads them as input | After generation - Foundry applies them to output |
| **What** | Deployment-specific parameters (paths, IDs, ARNs) that Foundry needs to generate files correctly | Modifications to already-generated files (resource limits, storage classes, scheduling) |
| **Validated** | Yes - Foundry reads and uses them during template execution | No - Foundry passes them through as-is |

In short: annotations tell Foundry *how to generate*, patches tell Foundry *what to change after generating*.

### When to use which

Use **annotations** when Foundry needs the value during generation. These are typically infrastructure identifiers, file paths, or settings that determine the shape of the generated output. You cannot achieve the same result with patches because the value must exist before files are generated.

Use **patches** when you want to modify a generated file after the fact. These are typically platform-specific tuning (resource limits, scheduling, service types) that Foundry doesn't need to understand.

## Which castings use annotations

Not all deployment modes require annotations. Most modes (Docker Compose, Docker Swarm, Kubernetes Kustomize, Render, Coolify, Railway) work with just `metadata.name` and `spec.deployment`.

| Casting | Required | Annotations |
|---|---|---|
| [Systemd (binary)](../examples/systemd/binary/) | Yes | Binary paths for SigNoz, ingester, and PostgreSQL |
| [ECS EC2 (Terraform)](../examples/ecs/ec2/terraform/) | Yes | AWS region, cluster ID, subnets, security groups, IAM roles, S3 bucket |
| [Kubernetes (Helm)](../examples/kubernetes/helm/) | No | Optional chart repo and chart name overrides |

For the complete list of annotation keys and their descriptions, see [Casting File Reference](../reference/casting-file.md#annotations).

## Next steps

- [Patches](patches.md) - post-generation modifications
- [Casting](casting.md) - the full casting file structure
- [Casting file reference](../reference/casting-file.md#annotations) - complete annotation reference
